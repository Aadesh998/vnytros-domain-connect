package repositary

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/db"
	"domain-connect-backend/internal/mailforge/model"
	"errors"
	"strings"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

var settingTracer = otel.Tracer("smtp_config_repositary")

func ListSmtpConfigs(ctx context.Context, userID uint) ([]model.SmtpConfig, *apperror.AppError) {
	_, span := settingTracer.Start(ctx, "ListSmtpConfigs")
	defer span.End()

	var configs []model.SmtpConfig
	if err := db.DB.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default desc, id asc").
		Find(&configs).Error; err != nil {
		return nil, &apperror.InternalServerError
	}
	return configs, nil
}

func GetSmtpConfigByID(ctx context.Context, userID, id uint) (model.SmtpConfig, *apperror.AppError) {
	_, span := settingTracer.Start(ctx, "GetSmtpConfigByID")
	defer span.End()

	var cfg model.SmtpConfig
	if err := db.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&cfg).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SmtpConfig{}, &apperror.NotFound
		}
		return model.SmtpConfig{}, &apperror.InternalServerError
	}
	return cfg, nil
}

func GetDefaultSmtpConfig(ctx context.Context, userID uint) (model.SmtpConfig, *apperror.AppError) {
	_, span := settingTracer.Start(ctx, "GetDefaultSmtpConfig")
	defer span.End()

	var cfg model.SmtpConfig
	err := db.DB.WithContext(ctx).
		Where("user_id = ? AND active = ?", userID, true).
		Order("is_default desc, id asc").
		First(&cfg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.SmtpConfig{}, &apperror.NotFound
		}
		return model.SmtpConfig{}, &apperror.InternalServerError
	}
	return cfg, nil
}

func CreateSmtpConfig(ctx context.Context, cfg *model.SmtpConfig) *apperror.AppError {
	_, span := settingTracer.Start(ctx, "CreateSmtpConfig")
	defer span.End()

	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// The first sender a user saves becomes their default automatically.
		var existing int64
		if err := tx.Model(&model.SmtpConfig{}).Where("user_id = ?", cfg.UserID).Count(&existing).Error; err != nil {
			return err
		}
		if existing == 0 {
			cfg.IsDefault = true
		}

		if err := tx.Create(cfg).Error; err != nil {
			return err
		}
		if cfg.IsDefault {
			return clearOtherDefaults(tx, cfg.UserID, cfg.ID)
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return &apperror.Conflict
		}
		return &apperror.InternalServerError
	}
	return nil
}

func UpdateSmtpConfig(ctx context.Context, cfg model.SmtpConfig) *apperror.AppError {
	_, span := settingTracer.Start(ctx, "UpdateSmtpConfig")
	defer span.End()

	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&cfg).Error; err != nil {
			return err
		}
		if cfg.IsDefault {
			return clearOtherDefaults(tx, cfg.UserID, cfg.ID)
		}
		return nil
	})
	if err != nil {
		if isUniqueViolation(err) {
			return &apperror.Conflict
		}
		return &apperror.InternalServerError
	}
	return nil
}

func SetDefaultSmtpConfig(ctx context.Context, userID, id uint) *apperror.AppError {
	_, span := settingTracer.Start(ctx, "SetDefaultSmtpConfig")
	defer span.End()

	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Model(&model.SmtpConfig{}).
			Where("id = ? AND user_id = ?", id, userID).
			Update("is_default", true)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return clearOtherDefaults(tx, userID, id)
	})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &apperror.NotFound
		}
		return &apperror.InternalServerError
	}
	return nil
}

func DeleteSmtpConfig(ctx context.Context, userID, id uint) *apperror.AppError {
	_, span := settingTracer.Start(ctx, "DeleteSmtpConfig")
	defer span.End()

	if err := db.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.SmtpConfig{}).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}

func clearOtherDefaults(tx *gorm.DB, userID, keepID uint) error {
	return tx.Model(&model.SmtpConfig{}).
		Where("user_id = ? AND id <> ?", userID, keepID).
		Update("is_default", false).Error
}

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	msg := err.Error()
	for _, marker := range []string{"duplicate key", "SQLSTATE 23505", "unique constraint"} {
		if strings.Contains(msg, marker) {
			return true
		}
	}
	return false
}
