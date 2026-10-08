package repository

import (
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/utils"
	"errors"
	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ApiKeyRepository interface {
	GetByUserID(userID uint) (*models.ApiKeys, error)
	GetAllByUserID(userID uint, cursor uint, limit int) ([]*models.ApiKeys, error)
	GetByID(id uint) (*models.ApiKeys, error)
	GetByKey(apiKey string) (*models.ApiKeys, error)
	IncrementDomainCount(userID uint) error
	CreateAPIKey(models.ApiKeys) error
	UpdateAPIKey(api string, status string) error
	UpdateByID(id uint, updates map[string]any) error
	DeleteByID(id uint) error
}

type apiKeyRepository struct {
	db *gorm.DB
}

func NewApiKeyRepository(db *gorm.DB) ApiKeyRepository {
	return &apiKeyRepository{db: db}
}

func (r *apiKeyRepository) CreateAPIKey(apikey models.ApiKeys) error {
	log.Printf("apiKeyRepository.CreateAPIKey: inserting api key for user_id=%d webhook=%q", apikey.UserID, apikey.WebHook)
	if err := r.db.Omit(clause.Associations).Create(&apikey).Error; err != nil {
		log.Printf("apiKeyRepository.CreateAPIKey: insert failed for user_id=%d: %v", apikey.UserID, err)
		return err
	}
	log.Printf("apiKeyRepository.CreateAPIKey: inserted id=%d for user_id=%d", apikey.ID, apikey.UserID)
	return nil
}

func (r *apiKeyRepository) UpdateAPIKey(api string, status string) error {
	err := r.db.Model(&models.ApiKeys{}).Where("api_key = ?", api).Update("status = ?", status).Error
	return err
}

// IncrementDomainCount records one more connected domain against the user's
// API keys. It is a usage statistic, not a quota.
func (r *apiKeyRepository) IncrementDomainCount(userID uint) error {
	return r.db.Model(&models.ApiKeys{}).
		Where("user_id = ?", userID).
		Update("domain_count", gorm.Expr("domain_count + 1")).Error
}

func (r *apiKeyRepository) GetByUserID(userID uint) (*models.ApiKeys, error) {
	var k models.ApiKeys
	if err := r.db.Preload("User").Where("user_id = ?", userID).First(&k).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

func (r *apiKeyRepository) GetAllByUserID(userID uint, cursor uint, limit int) ([]*models.ApiKeys, error) {
	limit = utils.ClampPageSize(limit)

	var keys []*models.ApiKeys
	q := r.db.Where("user_id = ?", userID)
	if cursor > 0 {
		q = q.Where("id < ?", cursor)
	}
	if err := q.Order("id DESC").Limit(limit + 1).Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *apiKeyRepository) GetByID(id uint) (*models.ApiKeys, error) {
	var k models.ApiKeys
	if err := r.db.Where("id = ?", id).First(&k).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}

func (r *apiKeyRepository) UpdateByID(id uint, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&models.ApiKeys{}).Where("id = ?", id).Updates(updates).Error
}

func (r *apiKeyRepository) DeleteByID(id uint) error {
	return r.db.Where("id = ?", id).Delete(&models.ApiKeys{}).Error
}

func (r *apiKeyRepository) GetByKey(apiKey string) (*models.ApiKeys, error) {
	var k models.ApiKeys
	if err := r.db.Preload("User").Where("api_key = ?", apiKey).First(&k).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &k, nil
}
