package dto

import "time"

type SmtpConfigRequest struct {
	Label      string `json:"label" binding:"required"`
	FromEmail  string `json:"from_email" binding:"required,email"`
	FromName   string `json:"from_name"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	Host       string `json:"host" binding:"required"`
	Port       int    `json:"port" binding:"required"`
	Encryption string `json:"encryption"`
	IsDefault  bool   `json:"is_default"`
	Active     *bool  `json:"active"`
}

type SmtpConfigResponse struct {
	ID             uint       `json:"id"`
	Label          string     `json:"label"`
	FromEmail      string     `json:"from_email"`
	FromName       string     `json:"from_name,omitempty"`
	Username       string     `json:"username"`
	Host           string     `json:"host"`
	Port           int        `json:"port"`
	Encryption     string     `json:"encryption"`
	IsDefault      bool       `json:"is_default"`
	Active         bool       `json:"active"`
	HasPassword    bool       `json:"has_password"`
	LastVerifiedAt *time.Time `json:"last_verified_at,omitempty"`
	LastError      string     `json:"last_error,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type SmtpConfigListResponse struct {
	Configs   []SmtpConfigResponse `json:"configs"`
	DefaultID uint                 `json:"default_id,omitempty"`
}
