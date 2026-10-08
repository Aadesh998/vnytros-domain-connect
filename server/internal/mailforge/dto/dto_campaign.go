package dto

import "time"

type CampaignRequest struct {
	CampaignName       string `json:"campaign_name" binding:"required"`
	Description        string `json:"description"`
	Status             string `json:"status"`
	TemplateID         uint   `json:"template_id" binding:"required"`
	AudienceDataSource string `json:"audience_data_source" binding:"required"`
}

type CampaignResponse struct {
	ID                 uint       `json:"id"`
	CampaignName       string     `json:"campaign_name"`
	Description        string     `json:"description"`
	Status             string     `json:"status"`
	TotalEmails        int        `json:"total_emails"`
	SentEmails         int        `json:"sent_emails"`
	FailedEmails       int        `json:"failed_emails"`
	OpenedEmails       int        `json:"opened_emails"`
	EstimatedTime      string     `json:"estimated_time"`
	ProgressPercentage float64    `json:"progress_percentage"`
	FailurePercentage  float64    `json:"failure_percentage"`
	OpenRate           float64    `json:"open_rate"`
	TemplateID         uint       `json:"template_id"`
	SmtpConfigID       uint       `json:"smtp_config_id"`
	AudienceDataSource string     `json:"audience_data_source"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type CampaignPaginationResponse struct {
	Campaigns []CampaignResponse `json:"campaigns"`
	NextID    uint               `json:"next_id,omitempty"`
}

type SendCampaignResponse struct {
	Campaign       CampaignResponse `json:"campaign"`
	QueuedEmails   int              `json:"queued_emails"`
	Batches        int              `json:"batches"`
	SkippedInvalid int              `json:"skipped_invalid"`
	Message        string           `json:"message"`
}
