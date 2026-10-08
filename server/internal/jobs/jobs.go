package jobs

type WebhookJob struct {
	EventLogID uint   `json:"event_log_id"`
	URL        string `json:"url"`
	Event      string `json:"event"`
	Payload    string `json:"payload"`
}

type EmailJob struct {
	To       string            `json:"to"`
	Subject  string            `json:"subject"`
	Template string            `json:"template"`
	Vars     map[string]string `json:"vars"`
}

type VerifyDomainJob struct {
	DomainID uint `json:"domain_id"`
	Attempt  int  `json:"attempt"`
}

type Watermark struct {
	Show     bool   `json:"show"`
	ImageURL string `json:"image_url"`
	LinkURL  string `json:"link_url"`
	Label    string `json:"label"`
}

type CampaignSendJob struct {
	CampaignID   uint      `json:"campaign_id"`
	UserID       uint      `json:"user_id"`
	TemplateID   uint      `json:"template_id"`
	SmtpConfigID uint      `json:"smtp_config_id"`
	Subject      string    `json:"subject"`
	Recipients   []string  `json:"recipients"`
	BatchIndex   int       `json:"batch_index"`
	TotalBatches int       `json:"total_batches"`
	TrackBaseURL string    `json:"track_base_url"`
	Watermark    Watermark `json:"watermark"`
}

type CampaignTrackJob struct {
	CampaignID uint   `json:"campaign_id"`
	UserID     uint   `json:"user_id"`
	TemplateID uint   `json:"template_id"`
	Recipient  string `json:"recipient"`
	EventType  string `json:"event_type"`
	UserAgent  string `json:"user_agent"`
	IPAddress  string `json:"ip_address"`
	OccurredAt int64  `json:"occurred_at"`
}
