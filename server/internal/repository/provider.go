package repository

import (
	"domain-connect-backend/internal/models"
	"gorm.io/gorm"
)

type ProviderRepository interface {
	GetAllProviders() ([]*models.Providers, error)
}

type providerRepository struct {
	db *gorm.DB
}

func NewProviderRepository(db *gorm.DB) ProviderRepository {
	return &providerRepository{db: db}
}

func (r *providerRepository) GetAllProviders() ([]*models.Providers, error) {
	var providers []*models.Providers
	if err := r.db.Find(&providers).Error; err != nil {
		return nil, err
	}
	return providers, nil
}
