package repository

import (
	"domain-connect-backend/internal/models"
	"errors"

	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *models.Users) error
	Update(user *models.Users) error
	GetByEmail(email string) (*models.Users, error)
	GetByID(id uint) (*models.Users, error)
	GetByToken(token string) (*models.Users, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *models.Users) error {
	return r.db.Create(user).Error
}

func (r *userRepository) Update(user *models.Users) error {
	return r.db.Save(user).Error
}

func (r *userRepository) GetByToken(token string) (*models.Users, error) {
	var user models.Users
	if err := r.db.Where("token = ?", token).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(email string) (*models.Users, error) {
	var user models.Users
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByID(id uint) (*models.Users, error) {
	var user models.Users
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}
