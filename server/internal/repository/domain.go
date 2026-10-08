package repository

import (
	"domain-connect-backend/internal/models"
	"errors"

	"gorm.io/gorm"
)

type DomainRepository interface {
	Create(domain *models.Domains) error
	Update(domain *models.Domains) error
	UpsertByDomain(domain *models.Domains) error
	GetByDomain(name string) (*models.Domains, error)
	GetByID(id uint) (*models.Domains, error)
	GetByUserID(userID uint) ([]*models.Domains, error)
	CountByUserID(userID uint) (int64, error)
	UpdateStatus(id uint, status string) error
}

type domainRepository struct {
	db *gorm.DB
}

func NewDomainRepository(db *gorm.DB) DomainRepository {
	return &domainRepository{db: db}
}

func (r *domainRepository) Create(domain *models.Domains) error {
	return r.db.Create(domain).Error
}

func (r *domainRepository) Update(domain *models.Domains) error {
	return r.db.Save(domain).Error
}

func (r *domainRepository) UpsertByDomain(domain *models.Domains) error {
	existing, err := r.GetByDomain(domain.DomainName)
	if err != nil {
		return err
	}
	if existing == nil {
		return r.Create(domain)
	}
	existing.UserID = domain.UserID
	existing.IP = domain.IP
	existing.Target = domain.Target
	existing.TextRecord = domain.TextRecord
	existing.Session = domain.Session
	existing.Status = models.DomainStatusPending
	if err := r.db.Save(existing).Error; err != nil {
		return err
	}
	*domain = *existing
	return nil
}

func (r *domainRepository) GetByDomain(name string) (*models.Domains, error) {
	var d models.Domains
	if err := r.db.Where("domain_name = ?", name).First(&d).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *domainRepository) GetByID(id uint) (*models.Domains, error) {
	var d models.Domains
	if err := r.db.First(&d, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &d, nil
}

func (r *domainRepository) GetByUserID(userID uint) ([]*models.Domains, error) {
	var list []*models.Domains
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *domainRepository) CountByUserID(userID uint) (int64, error) {
	var n int64
	if err := r.db.Model(&models.Domains{}).Where("user_id = ?", userID).Count(&n).Error; err != nil {
		return 0, err
	}
	return n, nil
}

func (r *domainRepository) UpdateStatus(id uint, status string) error {
	return r.db.Model(&models.Domains{}).Where("id = ?", id).Update("status", status).Error
}
