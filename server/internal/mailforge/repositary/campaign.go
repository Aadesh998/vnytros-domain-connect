package repositary

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/db"
	"domain-connect-backend/internal/mailforge/model"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
)

var tracer = otel.Tracer("repositary")

func SaveCampaign(ctx context.Context, campaign model.Campaign) (model.Campaign, *apperror.AppError) {
	_, span := tracer.Start(ctx, "SaveCampaign")
	defer span.End()

	if err := db.DB.WithContext(ctx).Create(&campaign).Error; err != nil {
		return model.Campaign{}, &apperror.InternalServerError
	}
	return campaign, nil
}

func GetCampaigns(ctx context.Context, userID uint, lastID uint, limit int) ([]model.Campaign, *apperror.AppError) {
	_, span := tracer.Start(ctx, "GetCampaigns")
	defer span.End()

	var campaigns []model.Campaign
	query := db.DB.WithContext(ctx).
		Where("user_id = ? AND status != ?", userID, model.CampaignStatusDraft).
		Order("id desc").Limit(limit)
	if lastID > 0 {
		query = query.Where("id < ?", lastID)
	}
	if err := query.Find(&campaigns).Error; err != nil {
		return nil, &apperror.InternalServerError
	}
	return campaigns, nil
}

func GetDraftCampaigns(ctx context.Context, userID uint, lastID uint, limit int) ([]model.Campaign, *apperror.AppError) {
	_, span := tracer.Start(ctx, "GetDraftCampaigns")
	defer span.End()

	var campaigns []model.Campaign
	query := db.DB.WithContext(ctx).
		Where("user_id = ? AND status = ?", userID, model.CampaignStatusDraft).
		Order("id desc").Limit(limit)
	if lastID > 0 {
		query = query.Where("id < ?", lastID)
	}
	if err := query.Find(&campaigns).Error; err != nil {
		return nil, &apperror.InternalServerError
	}
	return campaigns, nil
}

func GetCampaignByID(ctx context.Context, userID, id uint) (model.Campaign, *apperror.AppError) {
	_, span := tracer.Start(ctx, "GetCampaignByID")
	defer span.End()

	var campaign model.Campaign
	if err := db.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&campaign).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Campaign{}, &apperror.NotFound
		}
		return model.Campaign{}, &apperror.InternalServerError
	}
	return campaign, nil
}

func UpdateCampaign(ctx context.Context, campaign model.Campaign) *apperror.AppError {
	_, span := tracer.Start(ctx, "UpdateCampaign")
	defer span.End()

	if err := db.DB.WithContext(ctx).Save(&campaign).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}

func MarkCampaignQueued(ctx context.Context, userID, id uint, total, batches int, smtpConfigID uint) *apperror.AppError {
	_, span := tracer.Start(ctx, "MarkCampaignQueued")
	defer span.End()

	now := time.Now()
	err := db.DB.WithContext(ctx).
		Model(&model.Campaign{}).
		Where("id = ? AND user_id = ?", id, userID).
		Updates(map[string]any{
			"status":         model.CampaignStatusQueued,
			"total_emails":   total,
			"sent_emails":    0,
			"failed_emails":  0,
			"opened_emails":  0,
			"batches_total":  batches,
			"batches_done":   0,
			"smtp_config_id": smtpConfigID,
			"estimated_time": "calculating...",
			"started_at":     &now,
			"completed_at":   nil,
		}).Error
	if err != nil {
		return &apperror.InternalServerError
	}
	return nil
}

func UpdateCampaignStatus(ctx context.Context, userID, id uint, status string) *apperror.AppError {
	_, span := tracer.Start(ctx, "UpdateCampaignStatus")
	defer span.End()

	if err := db.DB.WithContext(ctx).
		Model(&model.Campaign{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("status", status).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}

func DeleteCampaign(ctx context.Context, userID, id uint) *apperror.AppError {
	_, span := tracer.Start(ctx, "DeleteCampaign")
	defer span.End()

	if err := db.DB.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&model.Campaign{}).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}
