package models

import (
	"time"

	"gorm.io/datatypes"
)

type Providers struct {
	ID                   uint           `gorm:"primaryKey;index" json:"id"`
	Name                 string         `gorm:"type:varchar(100);not null;index" json:"name"`
	NSPatterns           datatypes.JSON `gorm:"type:jsonb;index" json:"ns_patterns"`
	Logo                 string         `json:"logo"`
	DomainConnectSupport bool           `json:"domain_connect_support"`
	CreatedAt            time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Providers) TableName() string {
	return "providers"
}
