package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	CampaignStatusDraft      = "draft"
	CampaignStatusQueued     = "queued"
	CampaignStatusInProgress = "in_progress"
	CampaignStatusCompleted  = "completed"
	CampaignStatusFailed     = "failed"
)

const (
	RecipientStatusQueued = "queued"
	RecipientStatusSent   = "sent"
	RecipientStatusFailed = "failed"
)

const TrackEventOpen = "open"

type MailTemplate struct {
	ID        uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint           `gorm:"index;not null" json:"user_id"`
	Name      string         `gorm:"type:varchar(255);not null" json:"name"`
	Subject   string         `gorm:"type:varchar(255)" json:"subject"`
	Status    string         `gorm:"type:varchar(32);default:'draft'" json:"status"`
	Body      string         `gorm:"type:text" json:"body"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MailTemplate) TableName() string { return "mail_templates" }

type MailSmtpConfig struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint           `gorm:"index:idx_mail_smtp_user_label,unique;not null" json:"user_id"`
	Label          string         `gorm:"index:idx_mail_smtp_user_label,unique;type:varchar(120);not null" json:"label"`
	FromEmail      string         `gorm:"type:varchar(255);not null" json:"from_email"`
	FromName       string         `gorm:"type:varchar(255)" json:"from_name"`
	Username       string         `gorm:"type:varchar(255);not null" json:"username"`
	Password       string         `gorm:"type:text;not null" json:"-"`
	Host           string         `gorm:"type:varchar(255);not null" json:"host"`
	Port           int            `gorm:"not null;default:587" json:"port"`
	Encryption     string         `gorm:"type:varchar(16);default:'starttls'" json:"encryption"`
	IsDefault      bool           `gorm:"default:false" json:"is_default"`
	Active         bool           `gorm:"default:true" json:"active"`
	LastVerifiedAt *time.Time     `json:"last_verified_at,omitempty"`
	LastError      string         `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MailSmtpConfig) TableName() string { return "mail_smtp_configs" }

type MailCampaign struct {
	ID                 uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID             uint           `gorm:"index;not null" json:"user_id"`
	CampaignName       string         `gorm:"type:varchar(255);not null" json:"campaign_name"`
	Description        string         `gorm:"type:text" json:"description"`
	Status             string         `gorm:"type:varchar(32);not null;default:'draft'" json:"status"`
	TotalEmails        int            `gorm:"default:0" json:"total_emails"`
	SentEmails         int            `gorm:"default:0" json:"sent_emails"`
	FailedEmails       int            `gorm:"default:0" json:"failed_emails"`
	OpenedEmails       int            `gorm:"default:0" json:"opened_emails"`
	BatchesTotal       int            `gorm:"default:0" json:"batches_total"`
	BatchesDone        int            `gorm:"default:0" json:"batches_done"`
	EstimatedTime      string         `gorm:"type:varchar(64)" json:"estimated_time"`
	TemplateID         uint           `gorm:"index;not null" json:"template_id"`
	SmtpConfigID       uint           `gorm:"index" json:"smtp_config_id"`
	AudienceDataSource string         `gorm:"type:varchar(32)" json:"audience_data_source"`
	StartedAt          *time.Time     `json:"started_at,omitempty"`
	CompletedAt        *time.Time     `json:"completed_at,omitempty"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MailCampaign) TableName() string { return "mail_campaigns" }

type MailRecipient struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	CampaignID    uint       `gorm:"index:idx_mail_recipient_campaign_email,unique;not null" json:"campaign_id"`
	Email         string     `gorm:"index:idx_mail_recipient_campaign_email,unique;type:varchar(255);not null" json:"email"`
	UserID        uint       `gorm:"index;not null" json:"user_id"`
	Status        string     `gorm:"type:varchar(20);index;default:'queued'" json:"status"`
	ErrorMessage  string     `gorm:"type:text" json:"error_message,omitempty"`
	SentAt        *time.Time `json:"sent_at,omitempty"`
	FirstOpenedAt *time.Time `json:"first_opened_at,omitempty"`
	LastOpenedAt  *time.Time `json:"last_opened_at,omitempty"`
	OpenCount     int        `gorm:"default:0" json:"open_count"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (MailRecipient) TableName() string { return "mail_recipients" }

type MailTrack struct {
	ID         uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	CampaignID uint      `gorm:"index;not null" json:"campaign_id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	TemplateID uint      `gorm:"index" json:"template_id"`
	Recipient  string    `gorm:"type:varchar(255);index" json:"recipient"`
	EventType  string    `gorm:"type:varchar(20);index;default:'open'" json:"event_type"`
	UserAgent  string    `gorm:"type:text" json:"user_agent,omitempty"`
	IPAddress  string    `gorm:"type:varchar(64)" json:"ip_address,omitempty"`
	OccurredAt time.Time `gorm:"index" json:"occurred_at"`
	CreatedAt  time.Time `json:"created_at"`
}

func (MailTrack) TableName() string { return "mail_tracks" }

type MailBranding struct {
	ID                uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID            uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	ShowWatermark     bool      `gorm:"default:true" json:"show_watermark"`
	WatermarkImageURL string    `gorm:"type:text" json:"watermark_image_url"`
	WatermarkLinkURL  string    `gorm:"type:text" json:"watermark_link_url"`
	WatermarkLabel    string    `gorm:"type:varchar(120)" json:"watermark_label"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (MailBranding) TableName() string { return "mail_branding" }
