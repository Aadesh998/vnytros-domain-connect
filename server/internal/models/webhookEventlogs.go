package models

import "time"

type WebhookStatus string

const (
	WebhookPendingStatus   WebhookStatus = "pending"
	WebhookDeliveredStatus WebhookStatus = "delivered"
	WebhookFailedStatus    WebhookStatus = "failed"
)

type WebHookEventLogs struct {
	ID             uint          `json:"id" gorm:"primaryKey"`
	WebHook        string        `json:"web_hook" gorm:"type:varchar(255)"`
	Event          string        `json:"event" gorm:"type:varchar(100)"`
	Status         WebhookStatus `json:"status" gorm:"type:varchar(20);default:pending;index"`
	Payload        string        `json:"payload" gorm:"type:text"`
	ResponseStatus int           `json:"response_status" gorm:"default:0"`
	ResponseBody   string        `json:"response_body" gorm:"type:text"`
	Attempts       int           `json:"attempts" gorm:"default:0"`
	LastError      string        `json:"last_error" gorm:"type:text"`
	ApiKeyID       uint          `json:"api_key_id" gorm:"not null;index"`
	UserID         uint          `json:"user_id" gorm:"not null;index"`
	User           Users         `json:"user" gorm:"foreignKey:UserID;references:ID"`
	Api            ApiKeys       `json:"api" gorm:"foreignKey:ApiKeyID;references:ID"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
}

func (WebHookEventLogs) TableName() string {
	return "webhook_event_logs"
}
