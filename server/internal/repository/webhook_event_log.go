package repository

import (
	"errors"

	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/utils"

	"gorm.io/gorm"
)

type WebhookEventLogRepository interface {
	Create(log *models.WebHookEventLogs) error
	GetByID(id uint) (*models.WebHookEventLogs, error)
	Update(id uint, data map[string]any) error
	GetAllRecords(userID uint, cursor uint, limit int) ([]*models.WebHookEventLogs, error)
	GetRecordWithFilters(userID uint, filters map[string]interface{}, cursor uint, limit int) ([]*models.WebHookEventLogs, error)
}

type webhookEventLogRepository struct {
	db *gorm.DB
}

func NewWebhookEventLogRepository(db *gorm.DB) WebhookEventLogRepository {
	return &webhookEventLogRepository{db: db}
}

func (r *webhookEventLogRepository) Create(log *models.WebHookEventLogs) error {
	return r.db.Create(log).Error
}

func (r *webhookEventLogRepository) GetByID(id uint) (*models.WebHookEventLogs, error) {
	var log models.WebHookEventLogs
	if err := r.db.First(&log, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

func (r *webhookEventLogRepository) Update(id uint, data map[string]any) error {
	return r.db.Model(&models.WebHookEventLogs{}).Where("id = ?", id).Updates(data).Error
}

func (r *webhookEventLogRepository) GetAllRecords(userID uint, cursor uint, limit int) ([]*models.WebHookEventLogs, error) {
	limit = utils.ClampPageSize(limit)

	var logs []*models.WebHookEventLogs
	q := r.db.Where("user_id = ?", userID)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	if err := q.Order("id DESC").Limit(limit + 1).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *webhookEventLogRepository) GetRecordWithFilters(userID uint, filters map[string]interface{}, cursor uint, limit int) ([]*models.WebHookEventLogs, error) {
	limit = utils.ClampPageSize(limit)

	q := r.db.Model(&models.WebHookEventLogs{}).Where("user_id = ?", userID)

	if v, ok := filters["status"]; ok {
		q = q.Where("status = ?", v)
	}
	if v, ok := filters["event"]; ok {
		q = q.Where("event = ?", v)
	}
	if v, ok := filters["api_key_id"]; ok {
		q = q.Where("api_key_id = ?", v)
	}
	if v, ok := filters["response_status"]; ok {
		q = q.Where("response_status = ?", v)
	}
	if v, ok := filters["from"]; ok {
		q = q.Where("created_at >= ?", v)
	}
	if v, ok := filters["to"]; ok {
		q = q.Where("created_at <= ?", v)
	}

	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}

	var logs []*models.WebHookEventLogs
	if err := q.Order("id DESC").Limit(limit + 1).Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
