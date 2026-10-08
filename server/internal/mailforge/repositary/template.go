package repositary

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/db"
	"domain-connect-backend/internal/mailforge/model"
	"errors"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

var templateTracer = otel.Tracer("template_repositary")

func SaveTemplate(ctx context.Context, template *model.Template) *apperror.AppError {
	_, span := templateTracer.Start(ctx, "SaveTemplate")
	defer span.End()

	if err := db.DB.WithContext(ctx).Create(template).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}

func GetTemplates(ctx context.Context, userID uint, lastID uint, limit int) ([]model.Template, *apperror.AppError) {
	_, span := templateTracer.Start(ctx, "GetTemplates")
	defer span.End()

	return findTemplates(ctx, userID, lastID, limit, "published")
}

func GetDraftTemplates(ctx context.Context, userID uint, lastID uint, limit int) ([]model.Template, *apperror.AppError) {
	_, span := templateTracer.Start(ctx, "GetDraftTemplates")
	defer span.End()

	return findTemplates(ctx, userID, lastID, limit, "draft")
}

func findTemplates(ctx context.Context, userID, lastID uint, limit int, status string) ([]model.Template, *apperror.AppError) {
	var templates []model.Template
	query := db.DB.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, status).
		Order("id desc").Limit(limit)
	if lastID > 0 {
		query = query.Where("id < ?", lastID)
	}
	if err := query.Find(&templates).Error; err != nil {
		return nil, &apperror.InternalServerError
	}
	return templates, nil
}

func GetTemplateByID(ctx context.Context, userID, id uint) (model.Template, *apperror.AppError) {
	_, span := templateTracer.Start(ctx, "GetTemplateByID")
	defer span.End()

	var template model.Template
	if err := db.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&template).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Template{}, &apperror.NotFound
		}
		return model.Template{}, &apperror.InternalServerError
	}
	return template, nil
}

func UpdateTemplate(ctx context.Context, template model.Template) *apperror.AppError {
	_, span := templateTracer.Start(ctx, "UpdateTemplate")
	defer span.End()

	if err := db.DB.WithContext(ctx).Save(&template).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}

func DeleteTemplate(ctx context.Context, userID, id uint) *apperror.AppError {
	_, span := templateTracer.Start(ctx, "DeleteTemplate")
	defer span.End()

	if err := db.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.Template{}).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}
