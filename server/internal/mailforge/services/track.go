package services

import (
	"context"
	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/mailforge/queue"
	"time"
)

func RecordOpen(ctx context.Context, job jobs.CampaignTrackJob) error {
	if job.EventType == "" {
		job.EventType = "open"
	}
	if job.OccurredAt == 0 {
		job.OccurredAt = time.Now().Unix()
	}
	return queue.PublishCampaignTrack(ctx, job)
}
