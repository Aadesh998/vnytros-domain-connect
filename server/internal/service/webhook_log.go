package service

import (
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/utils"
	"strings"
	"time"
)

type WebhookLogFilters struct {
	Status         string
	Event          string
	ApiKeyID       uint
	ResponseStatus int
	From           string
	To             string
}

type WebhookLogPage struct {
	Items      []*models.WebHookEventLogs `json:"items"`
	NextCursor uint                       `json:"next_cursor,omitempty"`
	Limit      int                        `json:"limit"`
	HasMore    bool                       `json:"has_more"`
}

type WebhookLogService interface {
	GetAll(userID uint, cursor uint, limit int) (*WebhookLogPage, error)
	GetWithFilters(userID uint, filters WebhookLogFilters, cursor uint, limit int) (*WebhookLogPage, error)
}

type webhookLogService struct {
	repo repository.WebhookEventLogRepository
}

func NewWebhookLogService(repo repository.WebhookEventLogRepository) WebhookLogService {
	return &webhookLogService{repo: repo}
}

func (s *webhookLogService) GetAll(userID uint, cursor uint, limit int) (*WebhookLogPage, error) {
	limit = utils.ClampPageSize(limit)
	items, err := s.repo.GetAllRecords(userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	return buildPage(items, limit), nil
}

func (s *webhookLogService) GetWithFilters(userID uint, f WebhookLogFilters, cursor uint, limit int) (*WebhookLogPage, error) {
	m := map[string]interface{}{}
	if s := strings.TrimSpace(f.Status); s != "" {
		m["status"] = s
	}
	if s := strings.TrimSpace(f.Event); s != "" {
		m["event"] = s
	}
	if f.ApiKeyID > 0 {
		m["api_key_id"] = f.ApiKeyID
	}
	if f.ResponseStatus > 0 {
		m["response_status"] = f.ResponseStatus
	}
	if t, err := parseTime(f.From); err == nil && !t.IsZero() {
		m["from"] = t
	}
	if t, err := parseTime(f.To); err == nil && !t.IsZero() {
		m["to"] = t
	}

	limit = utils.ClampPageSize(limit)
	items, err := s.repo.GetRecordWithFilters(userID, m, cursor, limit)
	if err != nil {
		return nil, err
	}
	return buildPage(items, limit), nil
}

func buildPage(items []*models.WebHookEventLogs, limit int) *WebhookLogPage {
	data, next, hasMore := utils.BuildPage(items, limit, func(l *models.WebHookEventLogs) uint { return l.ID })
	return &WebhookLogPage{Items: data, NextCursor: next, Limit: limit, HasMore: hasMore}
}

func parseTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	return time.Parse("2006-01-02", s)
}
