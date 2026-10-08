package models

type Users struct {
	ID       uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Name     string `gorm:"type:varchar(100);not null" json:"name"`
	Email    string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Password string `gorm:"type:text;not null" json:"-"`
	Country  string `gorm:"type:varchar(100)" json:"country"`
	City     string `gorm:"type:varchar(100)" json:"city"`
	Token    string `gorm:"type:varchar(255);index" json:"token"`
	UserType string `gorm:"type:varchar(20);default:'user'" json:"user_type"`
	Verified bool   `gorm:"default:false" json:"verified"`
}

func (Users) TableName() string {
	return "users"
}
