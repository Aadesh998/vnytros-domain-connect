package worker

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/models"

	"gorm.io/gorm"
)

func (r *Register) HandleCampaignTrack(ctx context.Context, body []byte) error {
	var job jobs.CampaignTrackJob
	if err := json.Unmarshal(body, &job); err != nil {
		return Permanent(fmt.Errorf("campaign track: unmarshal: %w", err))
	}
	if job.CampaignID == 0 || job.Recipient == "" {
		return Permanent(fmt.Errorf("campaign track: invalid job: %+v", job))
	}
	if r.db == nil {
		return fmt.Errorf("campaign track: worker has no database handle")
	}

	occurredAt := time.Now()
	if job.OccurredAt > 0 {
		occurredAt = time.Unix(job.OccurredAt, 0)
	}

	eventType := job.EventType
	if eventType == "" {
		eventType = models.TrackEventOpen
	}

	// The tracking pixel is public, so mail_forge cannot include a trustworthy
	// user id. Ownership is resolved here from the campaign itself; without it
	// the event would be written with user_id 0 and never appear in the owner's
	// analytics, which are scoped by user.
	var campaign models.MailCampaign
	if err := r.db.WithContext(ctx).
		Select("id", "user_id", "template_id").
		Where("id = ?", job.CampaignID).
		First(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Permanent(fmt.Errorf("campaign track: campaign %d not found", job.CampaignID))
		}
		return fmt.Errorf("campaign track: load campaign: %w", err)
	}

	templateID := job.TemplateID
	if templateID == 0 {
		templateID = campaign.TemplateID
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		track := models.MailTrack{
			CampaignID: job.CampaignID,
			UserID:     campaign.UserID,
			TemplateID: templateID,
			Recipient:  job.Recipient,
			EventType:  eventType,
			UserAgent:  job.UserAgent,
			IPAddress:  job.IPAddress,
			OccurredAt: occurredAt,
		}
		if err := tx.Create(&track).Error; err != nil {
			return fmt.Errorf("insert track: %w", err)
		}

		if eventType != models.TrackEventOpen {
			return nil
		}

		var recipient models.MailRecipient
		err := tx.Where("campaign_id = ? AND email = ?", job.CampaignID, job.Recipient).
			First(&recipient).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("load recipient: %w", err)
		}

		isFirstOpen := recipient.FirstOpenedAt == nil

		updates := map[string]any{
			"last_opened_at": &occurredAt,
			"open_count":     gorm.Expr("open_count + 1"),
			"updated_at":     time.Now(),
		}
		if isFirstOpen {
			updates["first_opened_at"] = &occurredAt
		}

		if err := tx.Model(&models.MailRecipient{}).
			Where("id = ?", recipient.ID).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("update recipient: %w", err)
		}

		if !isFirstOpen {
			return nil
		}

		return tx.Model(&models.MailCampaign{}).
			Where("id = ?", job.CampaignID).
			Update("opened_emails", gorm.Expr("opened_emails + 1")).Error
	})
}
