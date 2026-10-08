package models

import (
	"encoding/json"
	"time"
)

type OAuthClient struct {
	ID                      uint      `gorm:"primaryKey" json:"-"`
	ClientID                string    `gorm:"uniqueIndex;type:varchar(64);not null" json:"client_id"`
	ClientName              string    `gorm:"type:varchar(255)" json:"client_name,omitempty"`
	RedirectURIs            string    `gorm:"type:text;not null" json:"-"`
	GrantTypes              string    `gorm:"type:varchar(255)" json:"-"`
	TokenEndpointAuthMethod string    `gorm:"type:varchar(40);default:'none'" json:"token_endpoint_auth_method"`
	CreatedAt               time.Time `json:"created_at"`
}

func (OAuthClient) TableName() string { return "oauth_clients" }

func (c *OAuthClient) RedirectURIList() []string {
	var uris []string
	if c.RedirectURIs == "" {
		return uris
	}
	_ = json.Unmarshal([]byte(c.RedirectURIs), &uris)
	return uris
}

func (c *OAuthClient) SetRedirectURIs(uris []string) error {
	b, err := json.Marshal(uris)
	if err != nil {
		return err
	}
	c.RedirectURIs = string(b)
	return nil
}

type OAuthAuthRequest struct {
	ID                  uint      `gorm:"primaryKey"`
	RequestID           string    `gorm:"uniqueIndex;type:varchar(64);not null"`
	ClientID            string    `gorm:"type:varchar(64);not null"`
	RedirectURI         string    `gorm:"type:text;not null"`
	Scope               string    `gorm:"type:text"`
	State               string    `gorm:"type:text"`
	CodeChallenge       string    `gorm:"type:varchar(255)"`
	CodeChallengeMethod string    `gorm:"type:varchar(10)"`
	ResponseType        string    `gorm:"type:varchar(20)"`
	UserID              *uint     `gorm:"index"`
	ExpiresAt           time.Time `gorm:"index"`
	CreatedAt           time.Time
}

func (OAuthAuthRequest) TableName() string { return "oauth_auth_requests" }

type OAuthAuthCode struct {
	ID                  uint      `gorm:"primaryKey"`
	CodeHash            string    `gorm:"uniqueIndex;type:varchar(128);not null"`
	ClientID            string    `gorm:"type:varchar(64);not null"`
	UserID              uint      `gorm:"not null"`
	RedirectURI         string    `gorm:"type:text;not null"`
	Scope               string    `gorm:"type:text"`
	CodeChallenge       string    `gorm:"type:varchar(255)"`
	CodeChallengeMethod string    `gorm:"type:varchar(10)"`
	Used                bool      `gorm:"default:false"`
	ExpiresAt           time.Time `gorm:"index"`
	CreatedAt           time.Time
}

func (OAuthAuthCode) TableName() string { return "oauth_auth_codes" }
