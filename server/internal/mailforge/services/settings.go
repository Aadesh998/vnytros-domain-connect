package services

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/authctx"
	"domain-connect-backend/internal/mailforge/config"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/model"
	"domain-connect-backend/internal/mailforge/repositary"
	"strconv"
	"strings"
)

func ListSmtpConfigs(ctx context.Context) (dto.SmtpConfigListResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "ListSmtpConfigs")
	defer span.End()

	configs, err := repositary.ListSmtpConfigs(ctx, authctx.UserID(ctx))
	if err != nil {
		return dto.SmtpConfigListResponse{}, err
	}

	out := make([]dto.SmtpConfigResponse, 0, len(configs))
	var defaultID uint
	for _, cfg := range configs {
		if cfg.IsDefault {
			defaultID = cfg.ID
		}
		out = append(out, mapSmtpConfig(cfg))
	}

	return dto.SmtpConfigListResponse{Configs: out, DefaultID: defaultID}, nil
}

func GetSmtpConfig(ctx context.Context, id uint) (dto.SmtpConfigResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetSmtpConfig")
	defer span.End()

	cfg, err := repositary.GetSmtpConfigByID(ctx, authctx.UserID(ctx), id)
	if err != nil {
		return dto.SmtpConfigResponse{}, err
	}
	return mapSmtpConfig(cfg), nil
}

func CreateSmtpConfig(ctx context.Context, req dto.SmtpConfigRequest) (dto.SmtpConfigResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "CreateSmtpConfig")
	defer span.End()

	if strings.TrimSpace(req.Password) == "" {
		return dto.SmtpConfigResponse{}, apperror.BadRequest.WithMessage("A password is required for a new sender")
	}

	cfg := model.SmtpConfig{
		UserID:     authctx.UserID(ctx),
		Label:      strings.TrimSpace(req.Label),
		FromEmail:  strings.TrimSpace(req.FromEmail),
		FromName:   strings.TrimSpace(req.FromName),
		Username:   resolveUsername(req),
		Password:   req.Password,
		Host:       strings.TrimSpace(req.Host),
		Port:       req.Port,
		Encryption: normaliseEncryption(req.Encryption),
		IsDefault:  req.IsDefault,
		Active:     true,
	}
	if req.Active != nil {
		cfg.Active = *req.Active
	}

	if err := repositary.CreateSmtpConfig(ctx, &cfg); err != nil {
		if err.HTTPStatus == 409 {
			return dto.SmtpConfigResponse{}, apperror.Conflict.WithMessage(
				"You already have a sender with this label")
		}
		return dto.SmtpConfigResponse{}, err
	}

	return mapSmtpConfig(cfg), nil
}

func UpdateSmtpConfig(ctx context.Context, id uint, req dto.SmtpConfigRequest) (dto.SmtpConfigResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "UpdateSmtpConfig")
	defer span.End()

	userID := authctx.UserID(ctx)

	cfg, err := repositary.GetSmtpConfigByID(ctx, userID, id)
	if err != nil {
		return dto.SmtpConfigResponse{}, err
	}

	cfg.Label = strings.TrimSpace(req.Label)
	cfg.FromEmail = strings.TrimSpace(req.FromEmail)
	cfg.FromName = strings.TrimSpace(req.FromName)
	cfg.Username = resolveUsername(req)
	cfg.Host = strings.TrimSpace(req.Host)
	cfg.Port = req.Port
	cfg.Encryption = normaliseEncryption(req.Encryption)
	cfg.IsDefault = req.IsDefault
	if req.Active != nil {
		cfg.Active = *req.Active
	}

	if strings.TrimSpace(req.Password) != "" {
		cfg.Password = req.Password
		cfg.LastError = ""
	}

	if err := repositary.UpdateSmtpConfig(ctx, cfg); err != nil {
		if err.HTTPStatus == 409 {
			return dto.SmtpConfigResponse{}, apperror.Conflict.WithMessage(
				"You already have a sender with this label")
		}
		return dto.SmtpConfigResponse{}, err
	}

	return mapSmtpConfig(cfg), nil
}

func SetDefaultSmtpConfig(ctx context.Context, id uint) (dto.SmtpConfigListResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "SetDefaultSmtpConfig")
	defer span.End()

	if err := repositary.SetDefaultSmtpConfig(ctx, authctx.UserID(ctx), id); err != nil {
		return dto.SmtpConfigListResponse{}, err
	}
	return ListSmtpConfigs(ctx)
}

func DeleteSmtpConfig(ctx context.Context, id uint) *apperror.AppError {
	ctx, span := tracer.Start(ctx, "DeleteSmtpConfig")
	defer span.End()

	userID := authctx.UserID(ctx)
	if _, err := repositary.GetSmtpConfigByID(ctx, userID, id); err != nil {
		return err
	}
	return repositary.DeleteSmtpConfig(ctx, userID, id)
}

func mapSmtpConfig(cfg model.SmtpConfig) dto.SmtpConfigResponse {
	return dto.SmtpConfigResponse{
		ID:             cfg.ID,
		Label:          cfg.Label,
		FromEmail:      cfg.FromEmail,
		FromName:       cfg.FromName,
		Username:       cfg.Username,
		Host:           cfg.Host,
		Port:           cfg.Port,
		Encryption:     cfg.Encryption,
		IsDefault:      cfg.IsDefault,
		Active:         cfg.Active,
		HasPassword:    cfg.Password != "",
		LastVerifiedAt: cfg.LastVerifiedAt,
		LastError:      cfg.LastError,
		CreatedAt:      cfg.CreatedAt,
		UpdatedAt:      cfg.UpdatedAt,
	}
}

func resolveUsername(req dto.SmtpConfigRequest) string {
	if u := strings.TrimSpace(req.Username); u != "" {
		return u
	}
	return strings.TrimSpace(req.FromEmail)
}

func normaliseEncryption(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case model.EncryptionSSL:
		return model.EncryptionSSL
	case model.EncryptionNone:
		return model.EncryptionNone
	default:
		return model.EncryptionStartTLS
	}
}

func SeedFallbackSmtpConfig(ctx context.Context) (dto.SmtpConfigResponse, *apperror.AppError) {
	userID := authctx.UserID(ctx)

	existing, err := repositary.ListSmtpConfigs(ctx, userID)
	if err != nil {
		return dto.SmtpConfigResponse{}, err
	}
	if len(existing) > 0 {
		return mapSmtpConfig(existing[0]), nil
	}

	cfg := config.AppConfig
	if cfg.EmailFrom == "" || cfg.EmailHost == "" || cfg.EmailPass == "" {
		return dto.SmtpConfigResponse{}, &apperror.SmtpNotConfigured
	}

	port, convErr := strconv.Atoi(cfg.EmailPort)
	if convErr != nil || port <= 0 {
		port = 587
	}

	seeded := model.SmtpConfig{
		UserID:     userID,
		Label:      "Default",
		FromEmail:  cfg.EmailFrom,
		Username:   cfg.EmailFrom,
		Password:   cfg.EmailPass,
		Host:       cfg.EmailHost,
		Port:       port,
		Encryption: model.EncryptionStartTLS,
		IsDefault:  true,
		Active:     true,
	}
	if err := repositary.CreateSmtpConfig(ctx, &seeded); err != nil {
		return dto.SmtpConfigResponse{}, err
	}

	return mapSmtpConfig(seeded), nil
}
