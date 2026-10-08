package repositary

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/db"
	"domain-connect-backend/internal/mailforge/model"

	"go.opentelemetry.io/otel"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var recipientTracer = otel.Tracer("recipient_repositary")

func ReplaceRecipients(ctx context.Context, campaignID, userID uint, emails []string) *apperror.AppError {
	_, span := recipientTracer.Start(ctx, "ReplaceRecipients")
	defer span.End()

	rows := make([]model.Recipient, 0, len(emails))
	for _, email := range emails {
		rows = append(rows, model.Recipient{
			CampaignID: campaignID,
			UserID:     userID,
			Email:      email,
			Status:     model.RecipientStatusQueued,
		})
	}

	err := db.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("campaign_id = ?", campaignID).
			Delete(&model.Recipient{}).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "campaign_id"}, {Name: "email"}},
			DoNothing: true,
		}).CreateInBatches(rows, 500).Error
	})
	if err != nil {
		return &apperror.InternalServerError
	}
	return nil
}

func ListRecipients(ctx context.Context, userID, campaignID uint, status string, offset, limit int) ([]model.Recipient, int64, *apperror.AppError) {
	_, span := recipientTracer.Start(ctx, "ListRecipients")
	defer span.End()

	base := db.DB.WithContext(ctx).
		Model(&model.Recipient{}).
		Where("campaign_id = ? AND user_id = ?", campaignID, userID)
	if status != "" {
		base = base.Where("status = ?", status)
	}

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, &apperror.InternalServerError
	}

	var recipients []model.Recipient
	if err := base.Order("id asc").Offset(offset).Limit(limit).
		Find(&recipients).Error; err != nil {
		return nil, 0, &apperror.InternalServerError
	}

	return recipients, total, nil
}
