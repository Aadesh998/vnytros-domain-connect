package dto

import "time"

type TemplateRequest struct {
	Name    string `json:"name" binding:"required"`
	Subject string `json:"subject" binding:"required"`
	Body    string `json:"body" binding:"required"`
	Status  string `json:"status" binding:"required"`
}

type TemplateResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Subject   string    `json:"subject"`
	Status    string    `json:"status"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type TemplatePaginationResponse struct {
	Templates []TemplateResponse `json:"templates"`
	NextID    uint               `json:"next_id,omitempty"`
}
