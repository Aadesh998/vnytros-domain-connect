package services

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/authctx"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/repositary"
	"math"
	"time"
)

const (
	defaultTimelineDays = 30
	maxTimelineDays     = 365
)

func GetAccountAnalytics(ctx context.Context, days int, interval string) (dto.AccountAnalyticsResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetAccountAnalytics")
	defer span.End()

	userID := authctx.UserID(ctx)
	days, interval = normaliseWindow(days, interval)
	since := time.Now().AddDate(0, 0, -days)

	totals, err := repositary.GetAccountTotals(ctx, userID)
	if err != nil {
		return dto.AccountAnalyticsResponse{}, err
	}

	buckets, err := repositary.GetAccountOpenTimeline(ctx, userID, interval, since)
	if err != nil {
		return dto.AccountAnalyticsResponse{}, err
	}

	return dto.AccountAnalyticsResponse{
		TotalCampaigns:  totals.Campaigns,
		ActiveCampaigns: totals.ActiveCampaign,
		TotalEmails:     totals.TotalEmails,
		SentEmails:      totals.SentEmails,
		FailedEmails:    totals.FailedEmails,
		UniqueOpens:     totals.OpenedEmails,
		DeliveryRate:    rate(totals.SentEmails, totals.TotalEmails),
		FailureRate:     rate(totals.FailedEmails, totals.TotalEmails),
		OpenRate:        rate(totals.OpenedEmails, totals.SentEmails),
		Timeline:        mapTimeline(buckets),
	}, nil
}

func GetCampaignAnalytics(ctx context.Context, campaignID uint, days int, interval string) (dto.CampaignAnalyticsResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetCampaignAnalytics")
	defer span.End()

	userID := authctx.UserID(ctx)

	campaign, err := repositary.GetCampaignByID(ctx, userID, campaignID)
	if err != nil {
		return dto.CampaignAnalyticsResponse{}, err
	}

	days, interval = normaliseWindow(days, interval)
	since := time.Now().AddDate(0, 0, -days)

	breakdown, err := repositary.GetRecipientBreakdown(ctx, userID, campaignID)
	if err != nil {
		return dto.CampaignAnalyticsResponse{}, err
	}

	buckets, err := repositary.GetOpenTimeline(ctx, userID, campaignID, interval, since)
	if err != nil {
		return dto.CampaignAnalyticsResponse{}, err
	}

	top, err := repositary.GetTopRecipients(ctx, userID, campaignID, 10)
	if err != nil {
		return dto.CampaignAnalyticsResponse{}, err
	}

	reasons, err := repositary.GetFailureReasons(ctx, userID, campaignID, 10)
	if err != nil {
		return dto.CampaignAnalyticsResponse{}, err
	}

	total := breakdown.Queued + breakdown.Sent + breakdown.Failed

	resp := dto.CampaignAnalyticsResponse{
		CampaignID:    campaign.ID,
		CampaignName:  campaign.CampaignName,
		Status:        campaign.Status,
		TotalEmails:   total,
		QueuedEmails:  breakdown.Queued,
		SentEmails:    breakdown.Sent,
		FailedEmails:  breakdown.Failed,
		UniqueOpens:   breakdown.UniqueOpens,
		TotalOpens:    breakdown.TotalOpens,
		DeliveryRate:  rate(breakdown.Sent, total),
		FailureRate:   rate(breakdown.Failed, total),
		OpenRate:      rate(breakdown.UniqueOpens, breakdown.Sent),
		EstimatedTime: campaign.EstimatedTime,
		StartedAt:     campaign.StartedAt,
		CompletedAt:   campaign.CompletedAt,
		Timeline:      mapTimeline(buckets),
	}

	resp.TopRecipients = make([]dto.TopRecipientEntry, 0, len(top))
	for _, t := range top {
		resp.TopRecipients = append(resp.TopRecipients, dto.TopRecipientEntry{
			Email:        t.Email,
			OpenCount:    t.OpenCount,
			LastOpenedAt: t.LastOpen,
		})
	}

	resp.FailureReasons = make([]dto.FailureReasonEntry, 0, len(reasons))
	for _, r := range reasons {
		resp.FailureReasons = append(resp.FailureReasons, dto.FailureReasonEntry{
			Reason: r.Reason,
			Count:  r.Count,
		})
	}

	return resp, nil
}

func ListCampaignRecipients(ctx context.Context, campaignID uint, status string, offset, limit int) (dto.RecipientListResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "ListCampaignRecipients")
	defer span.End()

	userID := authctx.UserID(ctx)

	if _, err := repositary.GetCampaignByID(ctx, userID, campaignID); err != nil {
		return dto.RecipientListResponse{}, err
	}

	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	recipients, total, err := repositary.ListRecipients(ctx, userID, campaignID, status, offset, limit)
	if err != nil {
		return dto.RecipientListResponse{}, err
	}

	rows := make([]dto.RecipientEntry, 0, len(recipients))
	for _, r := range recipients {
		rows = append(rows, dto.RecipientEntry{
			Email:         r.Email,
			Status:        r.Status,
			ErrorMessage:  r.ErrorMessage,
			SentAt:        r.SentAt,
			FirstOpenedAt: r.FirstOpenedAt,
			LastOpenedAt:  r.LastOpenedAt,
			OpenCount:     r.OpenCount,
		})
	}

	return dto.RecipientListResponse{
		Recipients: rows,
		Total:      total,
		Offset:     offset,
		Limit:      limit,
	}, nil
}

func mapTimeline(buckets []repositary.TimeBucket) []dto.TimeSeriesPoint {
	out := make([]dto.TimeSeriesPoint, 0, len(buckets))
	for _, b := range buckets {
		out = append(out, dto.TimeSeriesPoint{
			Bucket:      b.Bucket,
			Opens:       b.Opens,
			UniqueOpens: b.Unique,
		})
	}
	return out
}

func normaliseWindow(days int, interval string) (int, string) {
	if days <= 0 {
		days = defaultTimelineDays
	}
	if days > maxTimelineDays {
		days = maxTimelineDays
	}

	switch interval {
	case "hour", "day":
	default:
		if days <= 2 {
			interval = "hour"
		} else {
			interval = "day"
		}
	}

	return days, interval
}

func rate(part, whole int64) float64 {
	if whole <= 0 {
		return 0
	}
	return math.Round((float64(part)/float64(whole))*10000) / 100
}
