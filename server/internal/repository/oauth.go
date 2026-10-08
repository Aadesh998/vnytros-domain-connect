package repository

import (
	"domain-connect-backend/internal/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type OAuthRepository interface {
	CreateClient(c *models.OAuthClient) error
	GetClient(clientID string) (*models.OAuthClient, error)

	CreateAuthRequest(r *models.OAuthAuthRequest) error
	GetAuthRequest(requestID string) (*models.OAuthAuthRequest, error)
	SetAuthRequestUser(requestID string, userID uint) error
	DeleteAuthRequest(requestID string) error

	CreateCode(c *models.OAuthAuthCode) error
	GetCode(codeHash string) (*models.OAuthAuthCode, error)
	MarkCodeUsed(id uint) error

	DeleteExpired(now time.Time) error
}

type oauthRepository struct {
	db *gorm.DB
}

func NewOAuthRepository(db *gorm.DB) OAuthRepository {
	return &oauthRepository{db: db}
}

func (r *oauthRepository) CreateClient(c *models.OAuthClient) error {
	return r.db.Create(c).Error
}

func (r *oauthRepository) GetClient(clientID string) (*models.OAuthClient, error) {
	var c models.OAuthClient
	if err := r.db.Where("client_id = ?", clientID).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *oauthRepository) CreateAuthRequest(req *models.OAuthAuthRequest) error {
	return r.db.Create(req).Error
}

func (r *oauthRepository) GetAuthRequest(requestID string) (*models.OAuthAuthRequest, error) {
	var req models.OAuthAuthRequest
	if err := r.db.Where("request_id = ?", requestID).First(&req).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &req, nil
}

func (r *oauthRepository) SetAuthRequestUser(requestID string, userID uint) error {
	res := r.db.Model(&models.OAuthAuthRequest{}).
		Where("request_id = ?", requestID).
		Update("user_id", userID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("auth request not found")
	}
	return nil
}

func (r *oauthRepository) DeleteAuthRequest(requestID string) error {
	return r.db.Where("request_id = ?", requestID).Delete(&models.OAuthAuthRequest{}).Error
}

func (r *oauthRepository) CreateCode(c *models.OAuthAuthCode) error {
	return r.db.Create(c).Error
}

func (r *oauthRepository) GetCode(codeHash string) (*models.OAuthAuthCode, error) {
	var c models.OAuthAuthCode
	if err := r.db.Where("code_hash = ?", codeHash).First(&c).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

func (r *oauthRepository) MarkCodeUsed(id uint) error {
	return r.db.Model(&models.OAuthAuthCode{}).Where("id = ?", id).Update("used", true).Error
}

func (r *oauthRepository) DeleteExpired(now time.Time) error {
	if err := r.db.Where("expires_at < ?", now).Delete(&models.OAuthAuthRequest{}).Error; err != nil {
		return err
	}
	return r.db.Where("expires_at < ?", now).Delete(&models.OAuthAuthCode{}).Error
}
