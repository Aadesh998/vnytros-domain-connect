package dto

import "time"

type AccountAnalyticsResponse struct {
	TotalCampaigns  int64             `json:"total_campaigns"`
	ActiveCampaigns int64             `json:"active_campaigns"`
	TotalEmails     int64             `json:"total_emails"`
	SentEmails      int64             `json:"sent_emails"`
	FailedEmails    int64             `json:"failed_emails"`
	UniqueOpens     int64             `json:"unique_opens"`
	DeliveryRate    float64           `json:"delivery_rate"`
	FailureRate     float64           `json:"failure_rate"`
	OpenRate        float64           `json:"open_rate"`
	Timeline        []TimeSeriesPoint `json:"timeline"`
}

type CampaignAnalyticsResponse struct {
	CampaignID     uint                 `json:"campaign_id"`
	CampaignName   string               `json:"campaign_name"`
	Status         string               `json:"status"`
	TotalEmails    int64                `json:"total_emails"`
	QueuedEmails   int64                `json:"queued_emails"`
	SentEmails     int64                `json:"sent_emails"`
	FailedEmails   int64                `json:"failed_emails"`
	UniqueOpens    int64                `json:"unique_opens"`
	TotalOpens     int64                `json:"total_opens"`
	DeliveryRate   float64              `json:"delivery_rate"`
	FailureRate    float64              `json:"failure_rate"`
	OpenRate       float64              `json:"open_rate"`
	EstimatedTime  string               `json:"estimated_time"`
	StartedAt      *time.Time           `json:"started_at,omitempty"`
	CompletedAt    *time.Time           `json:"completed_at,omitempty"`
	Timeline       []TimeSeriesPoint    `json:"timeline"`
	TopRecipients  []TopRecipientEntry  `json:"top_recipients"`
	FailureReasons []FailureReasonEntry `json:"failure_reasons"`
}

type TimeSeriesPoint struct {
	Bucket      time.Time `json:"bucket"`
	Opens       int64     `json:"opens"`
	UniqueOpens int64     `json:"unique_opens"`
}

type TopRecipientEntry struct {
	Email        string     `json:"email"`
	OpenCount    int        `json:"open_count"`
	LastOpenedAt *time.Time `json:"last_opened_at,omitempty"`
}

type FailureReasonEntry struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type RecipientEntry struct {
	Email         string     `json:"email"`
	Status        string     `json:"status"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	FirstOpenedAt *time.Time `json:"first_opened_at,omitempty"`
	LastOpenedAt  *time.Time `json:"last_opened_at,omitempty"`
	OpenCount     int        `json:"open_count"`
}

type RecipientListResponse struct {
	Recipients []RecipientEntry `json:"recipients"`
	Total      int64            `json:"total"`
	Offset     int              `json:"offset"`
	Limit      int              `json:"limit"`
}
