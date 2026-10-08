package repositary

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/db"
	"domain-connect-backend/internal/mailforge/model"
	"time"

	"go.opentelemetry.io/otel"
)

var analyticsTracer = otel.Tracer("analytics_repositary")

type AccountTotals struct {
	Campaigns      int64 `gorm:"column:campaigns"`
	ActiveCampaign int64 `gorm:"column:active_campaigns"`
	TotalEmails    int64 `gorm:"column:total_emails"`
	SentEmails     int64 `gorm:"column:sent_emails"`
	FailedEmails   int64 `gorm:"column:failed_emails"`
	OpenedEmails   int64 `gorm:"column:opened_emails"`
}

func GetAccountTotals(ctx context.Context, userID uint) (AccountTotals, *apperror.AppError) {
	_, span := analyticsTracer.Start(ctx, "GetAccountTotals")
	defer span.End()

	var totals AccountTotals
	err := db.DB.WithContext(ctx).
		Model(&model.Campaign{}).
		Select(`
			COUNT(*) AS campaigns,
			COUNT(*) FILTER (WHERE status IN ('queued','in_progress')) AS active_campaigns,
			COALESCE(SUM(total_emails), 0) AS total_emails,
			COALESCE(SUM(sent_emails), 0) AS sent_emails,
			COALESCE(SUM(failed_emails), 0) AS failed_emails,
			COALESCE(SUM(opened_emails), 0) AS opened_emails`).
		Where("user_id = ?", userID).
		Scan(&totals).Error
	if err != nil {
		return AccountTotals{}, &apperror.InternalServerError
	}
	return totals, nil
}

type RecipientBreakdown struct {
	Queued      int64 `gorm:"column:queued"`
	Sent        int64 `gorm:"column:sent"`
	Failed      int64 `gorm:"column:failed"`
	UniqueOpens int64 `gorm:"column:unique_opens"`
	TotalOpens  int64 `gorm:"column:total_opens"`
}

func GetRecipientBreakdown(ctx context.Context, userID, campaignID uint) (RecipientBreakdown, *apperror.AppError) {
	_, span := analyticsTracer.Start(ctx, "GetRecipientBreakdown")
	defer span.End()

	var out RecipientBreakdown
	err := db.DB.WithContext(ctx).
		Model(&model.Recipient{}).
		Select(`
			COUNT(*) FILTER (WHERE status = 'queued') AS queued,
			COUNT(*) FILTER (WHERE status = 'sent') AS sent,
			COUNT(*) FILTER (WHERE status = 'failed') AS failed,
			COUNT(*) FILTER (WHERE first_opened_at IS NOT NULL) AS unique_opens,
			COALESCE(SUM(open_count), 0) AS total_opens`).
		Where("campaign_id = ? AND user_id = ?", campaignID, userID).
		Scan(&out).Error
	if err != nil {
		return RecipientBreakdown{}, &apperror.InternalServerError
	}
	return out, nil
}

type TimeBucket struct {
	Bucket time.Time `gorm:"column:bucket"`
	Opens  int64     `gorm:"column:opens"`
	Unique int64     `gorm:"column:uniques"`
}

func GetOpenTimeline(ctx context.Context, userID, campaignID uint, interval string, since time.Time) ([]TimeBucket, *apperror.AppError) {
	_, span := analyticsTracer.Start(ctx, "GetOpenTimeline")
	defer span.End()

	if interval != "hour" && interval != "day" {
		interval = "day"
	}

	var buckets []TimeBucket
	err := db.DB.WithContext(ctx).
		Model(&model.Track{}).
		Select(`
			date_trunc('`+interval+`', occurred_at) AS bucket,
			COUNT(*) AS opens,
			COUNT(DISTINCT recipient) AS uniques`).
		Where("campaign_id = ? AND user_id = ? AND event_type = ? AND occurred_at >= ?",
			campaignID, userID, model.TrackEventOpen, since).
		Group("bucket").
		Order("bucket asc").
		Scan(&buckets).Error
	if err != nil {
		return nil, &apperror.InternalServerError
	}
	return buckets, nil
}

func GetAccountOpenTimeline(ctx context.Context, userID uint, interval string, since time.Time) ([]TimeBucket, *apperror.AppError) {
	_, span := analyticsTracer.Start(ctx, "GetAccountOpenTimeline")
	defer span.End()

	if interval != "hour" && interval != "day" {
		interval = "day"
	}

	var buckets []TimeBucket
	err := db.DB.WithContext(ctx).
		Model(&model.Track{}).
		Select(`
			date_trunc('`+interval+`', occurred_at) AS bucket,
			COUNT(*) AS opens,
			COUNT(DISTINCT recipient) AS uniques`).
		Where("user_id = ? AND event_type = ? AND occurred_at >= ?",
			userID, model.TrackEventOpen, since).
		Group("bucket").
		Order("bucket asc").
		Scan(&buckets).Error
	if err != nil {
		return nil, &apperror.InternalServerError
	}
	return buckets, nil
}

type TopRecipient struct {
	Email     string     `gorm:"column:email"`
	OpenCount int        `gorm:"column:open_count"`
	LastOpen  *time.Time `gorm:"column:last_opened_at"`
}

func GetTopRecipients(ctx context.Context, userID, campaignID uint, limit int) ([]TopRecipient, *apperror.AppError) {
	_, span := analyticsTracer.Start(ctx, "GetTopRecipients")
	defer span.End()

	var out []TopRecipient
	err := db.DB.WithContext(ctx).
		Model(&model.Recipient{}).
		Select("email, open_count, last_opened_at").
		Where("campaign_id = ? AND user_id = ? AND open_count > 0", campaignID, userID).
		Order("open_count desc, last_opened_at desc").
		Limit(limit).
		Scan(&out).Error
	if err != nil {
		return nil, &apperror.InternalServerError
	}
	return out, nil
}

type FailureReason struct {
	Reason string `gorm:"column:reason"`
	Count  int64  `gorm:"column:count"`
}

func GetFailureReasons(ctx context.Context, userID, campaignID uint, limit int) ([]FailureReason, *apperror.AppError) {
	_, span := analyticsTracer.Start(ctx, "GetFailureReasons")
	defer span.End()

	var out []FailureReason
	err := db.DB.WithContext(ctx).
		Model(&model.Recipient{}).
		Select("COALESCE(NULLIF(error_message, ''), 'unknown') AS reason, COUNT(*) AS count").
		Where("campaign_id = ? AND user_id = ? AND status = ?", campaignID, userID, model.RecipientStatusFailed).
		Group("reason").
		Order("count desc").
		Limit(limit).
		Scan(&out).Error
	if err != nil {
		return nil, &apperror.InternalServerError
	}
	return out, nil
}
