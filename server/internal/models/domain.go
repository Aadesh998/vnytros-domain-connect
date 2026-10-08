package models

import "time"

const (
	DomainStatusPending   = "pending"
	DomainStatusIPApplied = "ip_applied"
	DomainStatusCompleted = "completed"
)

type Domains struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`
	DomainName string    `gorm:"type:varchar(255);uniqueIndex" json:"domain_name"`
	Status     string    `gorm:"type:varchar(32);default:'pending'" json:"status"`
	IP         string    `gorm:"type:varchar(255)" json:"ip"`
	Target     string    `gorm:"type:varchar(255)" json:"target"`
	TextRecord string    `gorm:"type:varchar(255)" json:"text_record"`
	Session    string    `gorm:"type:varchar(64);index" json:"session"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Domains) TableName() string {
	return "domains"
}
