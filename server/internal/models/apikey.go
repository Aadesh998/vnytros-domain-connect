package models

type ApiKeys struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	ApiKey      string `json:"api_key" gorm:"type:text;not null"`
	WebHook     string `json:"web_hook" gorm:"type:varchar(255);not null"`
	UserID      uint   `json:"user_id"`
	User        Users  `gorm:"foreignKey:UserID;references:ID" json:"user"`
	Status      string `gorm:"default:active" json:"status"` // active, disable, cancelled
	DomainCount int64  `gorm:"default:0" json:"domain_count"`
}

func (ApiKeys) TableName() string {
	return "api_keys"
}
