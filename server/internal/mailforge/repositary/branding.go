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

var brandingTracer = otel.Tracer("branding_repositary")

func GetBranding(ctx context.Context, userID uint) (model.Branding, bool, *apperror.AppError) {
	_, span := brandingTracer.Start(ctx, "GetBranding")
	defer span.End()

	var branding model.Branding
	err := db.DB.WithContext(ctx).Where("user_id = ?", userID).First(&branding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Branding{}, false, nil
	}
	if err != nil {
		return model.Branding{}, false, &apperror.InternalServerError
	}
	return branding, true, nil
}

func SaveBranding(ctx context.Context, branding *model.Branding) *apperror.AppError {
	_, span := brandingTracer.Start(ctx, "SaveBranding")
	defer span.End()

	if err := db.DB.WithContext(ctx).Save(branding).Error; err != nil {
		return &apperror.InternalServerError
	}
	return nil
}
