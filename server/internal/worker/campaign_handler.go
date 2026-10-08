package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/mail"
	"domain-connect-backend/internal/models"

	gomail "gopkg.in/mail.v2"
	"gorm.io/gorm"
)

// HandleCampaignSend delivers one batch of a mail-forge campaign.
//
// Parallelism: the recipients in a batch are split into shards, and each shard
// gets its own SMTP connection driven by its own goroutine. A single SMTP
// connection is not safe for concurrent use, so this is where the speed-up
// comes from — one connection reused for many messages, several connections at
// once. Shard count is bounded by MAIL_SEND_CONCURRENCY.
func (r *Register) HandleCampaignSend(ctx context.Context, body []byte) error {
	var job jobs.CampaignSendJob
	if err := json.Unmarshal(body, &job); err != nil {
		return Permanent(fmt.Errorf("campaign send: unmarshal: %w", err))
	}
	if job.CampaignID == 0 || job.UserID == 0 || len(job.Recipients) == 0 {
		return Permanent(fmt.Errorf("campaign send: invalid job: %+v", job))
	}
	if r.db == nil {
		return fmt.Errorf("campaign send: worker has no database handle")
	}

	var smtpCfg models.MailSmtpConfig
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", job.SmtpConfigID, job.UserID).
		First(&smtpCfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.failBatch(ctx, job, "sender credential no longer exists")
			return Permanent(fmt.Errorf("campaign send: smtp config %d not found for user %d", job.SmtpConfigID, job.UserID))
		}
		return fmt.Errorf("campaign send: load smtp config: %w", err)
	}

	var tmpl models.MailTemplate
	if err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", job.TemplateID, job.UserID).
		First(&tmpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.failBatch(ctx, job, "template no longer exists")
			return Permanent(fmt.Errorf("campaign send: template %d not found for user %d", job.TemplateID, job.UserID))
		}
		return fmt.Errorf("campaign send: load template: %w", err)
	}

	shards := shardRecipients(job.Recipients, campaignConcurrency(len(job.Recipients)))

	var (
		mu     sync.Mutex
		sent   int
		failed int
		wg     sync.WaitGroup
	)

	for _, shard := range shards {
		wg.Add(1)
		go func(recipients []string) {
			defer wg.Done()
			s, f := r.sendShard(ctx, job, smtpCfg, tmpl.Body, recipients)
			mu.Lock()
			sent += s
			failed += f
			mu.Unlock()
		}(shard)
	}
	wg.Wait()

	if err := r.applyBatchResult(ctx, job, sent, failed); err != nil {
		// The mail already went out; losing the counter update must never cause
		// a resend, so this is logged rather than returned.
		log.Printf("worker[campaign.send]: campaign %d batch %d: counter update failed: %v",
			job.CampaignID, job.BatchIndex, err)
	}

	log.Printf("worker[campaign.send]: campaign %d batch %d/%d done — sent=%d failed=%d",
		job.CampaignID, job.BatchIndex+1, job.TotalBatches, sent, failed)
	return nil
}

// sendShard opens one SMTP connection and walks its slice of recipients over
// it. A dial failure fails the whole shard rather than requeueing: bad
// credentials would otherwise poison the queue forever, and the error is more
// useful recorded against each recipient where the user can see it.
func (r *Register) sendShard(
	ctx context.Context,
	job jobs.CampaignSendJob,
	smtpCfg models.MailSmtpConfig,
	templateBody string,
	recipients []string,
) (sent int, failed int) {
	dialer := mail.NewCampaignDialer(smtpCfg)

	closer, err := dialer.Dial()
	if err != nil {
		reason := fmt.Sprintf("smtp connection failed: %v", err)
		log.Printf("worker[campaign.send]: campaign %d: %s", job.CampaignID, reason)
		r.recordSmtpError(ctx, smtpCfg.ID, reason)
		for _, to := range recipients {
			r.markRecipient(ctx, job, to, models.RecipientStatusFailed, reason)
		}
		return 0, len(recipients)
	}
	defer func() {
		if cerr := closer.Close(); cerr != nil {
			log.Printf("worker[campaign.send]: campaign %d: smtp close: %v", job.CampaignID, cerr)
		}
	}()

	for _, to := range recipients {
		if ctx.Err() != nil {
			return sent, failed
		}

		msg := mail.NewCampaignMessage(smtpCfg, job, templateBody, to)
		if err := gomail.Send(closer, msg); err != nil {
			log.Printf("worker[campaign.send]: campaign %d: send to %s failed: %v", job.CampaignID, to, err)
			r.markRecipient(ctx, job, to, models.RecipientStatusFailed, err.Error())
			failed++
			continue
		}

		r.markRecipient(ctx, job, to, models.RecipientStatusSent, "")
		sent++
	}

	return sent, failed
}

func (r *Register) markRecipient(ctx context.Context, job jobs.CampaignSendJob, email, status, errMsg string) {
	updates := map[string]any{
		"status":        status,
		"error_message": errMsg,
		"updated_at":    time.Now(),
	}
	if status == models.RecipientStatusSent {
		now := time.Now()
		updates["sent_at"] = &now
	}

	if err := r.db.WithContext(ctx).
		Model(&models.MailRecipient{}).
		Where("campaign_id = ? AND email = ?", job.CampaignID, email).
		Updates(updates).Error; err != nil {
		log.Printf("worker[campaign.send]: campaign %d: mark recipient %s: %v", job.CampaignID, email, err)
	}
}

func (r *Register) recordSmtpError(ctx context.Context, smtpConfigID uint, reason string) {
	if err := r.db.WithContext(ctx).
		Model(&models.MailSmtpConfig{}).
		Where("id = ?", smtpConfigID).
		Update("last_error", reason).Error; err != nil {
		log.Printf("worker[campaign.send]: record smtp error on config %d: %v", smtpConfigID, err)
	}
}

// applyBatchResult folds this batch's tallies into the campaign row and closes
// the campaign out once the last batch reports in. The counter arithmetic runs
// in SQL so concurrent batches cannot clobber each other.
func (r *Register) applyBatchResult(ctx context.Context, job jobs.CampaignSendJob, sent, failed int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.MailCampaign{}).
			Where("id = ?", job.CampaignID).
			Updates(map[string]any{
				"sent_emails":   gorm.Expr("sent_emails + ?", sent),
				"failed_emails": gorm.Expr("failed_emails + ?", failed),
				"batches_done":  gorm.Expr("batches_done + 1"),
				"status":        models.CampaignStatusInProgress,
				"updated_at":    time.Now(),
			}).Error; err != nil {
			return err
		}

		var campaign models.MailCampaign
		if err := tx.Where("id = ?", job.CampaignID).First(&campaign).Error; err != nil {
			return err
		}

		if campaign.BatchesTotal > 0 && campaign.BatchesDone < campaign.BatchesTotal {
			remaining := campaign.TotalEmails - (campaign.SentEmails + campaign.FailedEmails)
			return tx.Model(&models.MailCampaign{}).
				Where("id = ?", job.CampaignID).
				Update("estimated_time", estimateRemaining(campaign, remaining)).Error
		}

		status := models.CampaignStatusCompleted
		if campaign.TotalEmails > 0 && campaign.SentEmails == 0 {
			status = models.CampaignStatusFailed
		}
		now := time.Now()

		return tx.Model(&models.MailCampaign{}).
			Where("id = ?", job.CampaignID).
			Updates(map[string]any{
				"status":         status,
				"estimated_time": "0s",
				"completed_at":   &now,
				"updated_at":     now,
			}).Error
	})
}

// failBatch records an unrecoverable batch (missing template or credential)
// against every recipient in it, so the failure is visible in analytics.
func (r *Register) failBatch(ctx context.Context, job jobs.CampaignSendJob, reason string) {
	for _, to := range job.Recipients {
		r.markRecipient(ctx, job, to, models.RecipientStatusFailed, reason)
	}
	if err := r.applyBatchResult(ctx, job, 0, len(job.Recipients)); err != nil {
		log.Printf("worker[campaign.send]: campaign %d: fail batch bookkeeping: %v", job.CampaignID, err)
	}
}

func estimateRemaining(c models.MailCampaign, remaining int) string {
	if remaining <= 0 || c.StartedAt == nil {
		return "0s"
	}
	done := c.SentEmails + c.FailedEmails
	if done <= 0 {
		return "calculating..."
	}

	perEmail := time.Since(*c.StartedAt) / time.Duration(done)
	return formatDuration(perEmail * time.Duration(remaining))
}

func formatDuration(d time.Duration) string {
	switch {
	case d <= 0:
		return "0s"
	case d.Hours() >= 1:
		return fmt.Sprintf("%.1fh", d.Hours())
	case d.Minutes() >= 1:
		return fmt.Sprintf("%.1fm", d.Minutes())
	default:
		return fmt.Sprintf("%.0fs", d.Seconds())
	}
}

func campaignConcurrency(recipientCount int) int {
	limit := 8
	if config.AppConfig != nil && config.AppConfig.MailSendConcurrency > 0 {
		limit = config.AppConfig.MailSendConcurrency
	}
	if recipientCount < limit {
		return recipientCount
	}
	return limit
}

// shardRecipients deals recipients round-robin into n slices, dropping any
// shard that ends up empty.
func shardRecipients(recipients []string, n int) [][]string {
	if n <= 1 {
		return [][]string{recipients}
	}

	shards := make([][]string, n)
	for i, to := range recipients {
		idx := i % n
		shards[idx] = append(shards[idx], to)
	}

	out := shards[:0]
	for _, s := range shards {
		if len(s) > 0 {
			out = append(out, s)
		}
	}
	return out
}
