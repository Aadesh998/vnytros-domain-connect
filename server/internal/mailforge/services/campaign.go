package services

import (
	"context"
	"domain-connect-backend/internal/jobs"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/authctx"
	"domain-connect-backend/internal/mailforge/config"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/model"
	"domain-connect-backend/internal/mailforge/queue"
	"domain-connect-backend/internal/mailforge/repositary"
	"encoding/csv"
	"io"
	"log"
	"math"
	"net/mail"
	"strings"

	"go.opentelemetry.io/otel"
)

var tracer = otel.Tracer("services")

func CreateCampaign(ctx context.Context, req dto.CampaignRequest) (dto.CampaignResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "CreateCampaign")
	defer span.End()

	userID := authctx.UserID(ctx)

	if _, err := repositary.GetTemplateByID(ctx, userID, req.TemplateID); err != nil {
		return dto.CampaignResponse{}, err
	}

	campaign := model.Campaign{
		UserID:             userID,
		CampaignName:       req.CampaignName,
		Description:        req.Description,
		Status:             req.Status,
		TemplateID:         req.TemplateID,
		AudienceDataSource: req.AudienceDataSource,
	}
	if campaign.Status == "" {
		campaign.Status = model.CampaignStatusDraft
	}

	saved, err := repositary.SaveCampaign(ctx, campaign)
	if err != nil {
		return dto.CampaignResponse{}, err
	}

	return mapToCampaignResponse(saved), nil
}

func SendCampaign(ctx context.Context, id uint, smtpConfigID uint, csvFile io.Reader) (dto.SendCampaignResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "SendCampaign")
	defer span.End()

	userID := authctx.UserID(ctx)

	campaign, err := repositary.GetCampaignByID(ctx, userID, id)
	if err != nil {
		return dto.SendCampaignResponse{}, err
	}

	switch campaign.Status {
	case model.CampaignStatusQueued, model.CampaignStatusInProgress:
		return dto.SendCampaignResponse{}, &apperror.CampaignNotSendable
	}

	template, err := repositary.GetTemplateByID(ctx, userID, campaign.TemplateID)
	if err != nil {
		return dto.SendCampaignResponse{}, err
	}

	smtpConfig, err := resolveSmtpConfig(ctx, userID, smtpConfigID)
	if err != nil {
		return dto.SendCampaignResponse{}, err
	}

	recipients, skipped, err := parseRecipients(csvFile)
	if err != nil {
		return dto.SendCampaignResponse{}, err
	}
	if len(recipients) == 0 {
		return dto.SendCampaignResponse{}, &apperror.NoRecipients
	}

	watermark, err := ResolveWatermark(ctx)
	if err != nil {
		return dto.SendCampaignResponse{}, err
	}

	batchSize := config.AppConfig.SendBatchSize
	batches := chunk(recipients, batchSize)

	if err := repositary.ReplaceRecipients(ctx, campaign.ID, userID, recipients); err != nil {
		return dto.SendCampaignResponse{}, err
	}
	if err := repositary.MarkCampaignQueued(ctx, userID, campaign.ID, len(recipients), len(batches), smtpConfig.ID); err != nil {
		return dto.SendCampaignResponse{}, err
	}

	subject := template.Subject
	if subject == "" {
		subject = campaign.CampaignName
	}

	published := 0
	for i, batch := range batches {
		job := jobs.CampaignSendJob{
			CampaignID:   campaign.ID,
			UserID:       userID,
			TemplateID:   campaign.TemplateID,
			SmtpConfigID: smtpConfig.ID,
			Subject:      subject,
			Recipients:   batch,
			BatchIndex:   i,
			TotalBatches: len(batches),
			TrackBaseURL: config.AppConfig.BaseURL,
			Watermark:    watermark,
		}

		if pubErr := queue.PublishCampaignSend(ctx, job); pubErr != nil {
			log.Printf("SendCampaign: campaign %d batch %d publish failed: %v", campaign.ID, i, pubErr)
			break
		}
		published++
	}

	if published == 0 {
		if statusErr := repositary.UpdateCampaignStatus(ctx, userID, campaign.ID, model.CampaignStatusDraft); statusErr != nil {
			log.Printf("SendCampaign: campaign %d rollback failed: %v", campaign.ID, statusErr)
		}
		return dto.SendCampaignResponse{}, &apperror.QueueUnavailable
	}

	if published < len(batches) {
		log.Printf("SendCampaign: campaign %d published only %d/%d batches", campaign.ID, published, len(batches))
		if statusErr := repositary.UpdateCampaignStatus(ctx, userID, campaign.ID, model.CampaignStatusFailed); statusErr != nil {
			log.Printf("SendCampaign: campaign %d status update failed: %v", campaign.ID, statusErr)
		}
		return dto.SendCampaignResponse{}, apperror.QueueUnavailable.WithMessage(
			"Only part of this campaign could be queued. It has been marked failed; please retry.")
	}

	updated, err := repositary.GetCampaignByID(ctx, userID, campaign.ID)
	if err != nil {
		return dto.SendCampaignResponse{}, err
	}

	return dto.SendCampaignResponse{
		Campaign:       mapToCampaignResponse(updated),
		QueuedEmails:   len(recipients),
		Batches:        len(batches),
		SkippedInvalid: skipped,
		Message:        "Campaign queued for delivery",
	}, nil
}

// resolveSmtpConfig picks the sender for a campaign: the one the caller
// chose, or the account's default.
func resolveSmtpConfig(ctx context.Context, userID, smtpConfigID uint) (model.SmtpConfig, *apperror.AppError) {
	if smtpConfigID > 0 {
		cfg, err := repositary.GetSmtpConfigByID(ctx, userID, smtpConfigID)
		if err != nil {
			if err.HTTPStatus == 404 {
				return model.SmtpConfig{}, &apperror.SmtpNotConfigured
			}
			return model.SmtpConfig{}, err
		}
		if !cfg.Active {
			return model.SmtpConfig{}, apperror.SmtpNotConfigured.WithMessage(
				"The selected sender is disabled. Enable it or pick another.")
		}
		return cfg, nil
	}

	cfg, err := repositary.GetDefaultSmtpConfig(ctx, userID)
	if err != nil {
		if err.HTTPStatus == 404 {
			return model.SmtpConfig{}, &apperror.SmtpNotConfigured
		}
		return model.SmtpConfig{}, err
	}
	return cfg, nil
}

func parseRecipients(r io.Reader) ([]string, int, *apperror.AppError) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true

	records, err := reader.ReadAll()
	if err != nil {
		return nil, 0, apperror.BadRequest.WithMessage("The audience file could not be parsed as CSV")
	}

	seen := make(map[string]struct{}, len(records))
	out := make([]string, 0, len(records))
	skipped := 0

	for i, record := range records {
		if len(record) == 0 {
			continue
		}

		candidate := strings.TrimSpace(record[0])
		if candidate == "" {
			continue
		}
		if i == 0 && strings.EqualFold(candidate, "email") {
			continue
		}

		address, ok := extractAddress(candidate)
		if !ok {
			skipped++
			continue
		}

		normalised := strings.ToLower(address)
		if _, dup := seen[normalised]; dup {
			continue
		}
		seen[normalised] = struct{}{}
		out = append(out, normalised)
	}

	return out, skipped, nil
}

func extractAddress(candidate string) (string, bool) {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return "", false
	}

	if addr, err := mail.ParseAddress(candidate); err == nil {
		return addr.Address, true
	}

	cleaned := strings.TrimSpace(strings.ReplaceAll(candidate, `"`, ""))
	if cleaned == "" {
		return "", false
	}
	if addr, err := mail.ParseAddress(cleaned); err == nil {
		return addr.Address, true
	}

	if open := strings.LastIndex(cleaned, "<"); open >= 0 {
		if close := strings.Index(cleaned[open:], ">"); close > 0 {
			inner := strings.TrimSpace(cleaned[open+1 : open+close])
			if addr, err := mail.ParseAddress(inner); err == nil {
				return addr.Address, true
			}
		}
	}

	return "", false
}

func chunk(items []string, size int) [][]string {
	if size <= 0 {
		size = 50
	}

	out := make([][]string, 0, (len(items)+size-1)/size)
	for start := 0; start < len(items); start += size {
		end := min(start+size, len(items))
		out = append(out, items[start:end])
	}
	return out
}

func mapToCampaignResponse(c model.Campaign) dto.CampaignResponse {
	var progress, failure, openRate float64
	if c.TotalEmails > 0 {
		progress = percent(c.SentEmails, c.TotalEmails)
		failure = percent(c.FailedEmails, c.TotalEmails)
	}
	if c.SentEmails > 0 {
		openRate = percent(c.OpenedEmails, c.SentEmails)
	}

	return dto.CampaignResponse{
		ID:                 c.ID,
		CampaignName:       c.CampaignName,
		Description:        c.Description,
		Status:             c.Status,
		TotalEmails:        c.TotalEmails,
		SentEmails:         c.SentEmails,
		FailedEmails:       c.FailedEmails,
		OpenedEmails:       c.OpenedEmails,
		EstimatedTime:      c.EstimatedTime,
		ProgressPercentage: progress,
		FailurePercentage:  failure,
		OpenRate:           openRate,
		TemplateID:         c.TemplateID,
		SmtpConfigID:       c.SmtpConfigID,
		AudienceDataSource: c.AudienceDataSource,
		StartedAt:          c.StartedAt,
		CompletedAt:        c.CompletedAt,
		CreatedAt:          c.CreatedAt,
	}
}

func percent(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return math.Round((float64(part)/float64(whole))*10000) / 100
}

func GetCampaigns(ctx context.Context, lastID uint, limit int) (dto.CampaignPaginationResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetCampaigns")
	defer span.End()

	campaigns, err := repositary.GetCampaigns(ctx, authctx.UserID(ctx), lastID, limit)
	if err != nil {
		return dto.CampaignPaginationResponse{}, err
	}
	return paginateCampaigns(campaigns), nil
}

func GetDraftCampaigns(ctx context.Context, lastID uint, limit int) (dto.CampaignPaginationResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetDraftCampaigns")
	defer span.End()

	campaigns, err := repositary.GetDraftCampaigns(ctx, authctx.UserID(ctx), lastID, limit)
	if err != nil {
		return dto.CampaignPaginationResponse{}, err
	}
	return paginateCampaigns(campaigns), nil
}

func paginateCampaigns(campaigns []model.Campaign) dto.CampaignPaginationResponse {
	response := make([]dto.CampaignResponse, 0, len(campaigns))
	for _, c := range campaigns {
		response = append(response, mapToCampaignResponse(c))
	}

	var nextID uint
	if len(campaigns) > 0 {
		nextID = campaigns[len(campaigns)-1].ID
	}

	return dto.CampaignPaginationResponse{Campaigns: response, NextID: nextID}
}

func GetCampaign(ctx context.Context, id uint) (dto.CampaignResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetCampaign")
	defer span.End()

	c, err := repositary.GetCampaignByID(ctx, authctx.UserID(ctx), id)
	if err != nil {
		return dto.CampaignResponse{}, err
	}
	return mapToCampaignResponse(c), nil
}

func UpdateCampaign(ctx context.Context, id uint, req dto.CampaignRequest) (dto.CampaignResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "UpdateCampaign")
	defer span.End()

	userID := authctx.UserID(ctx)

	c, err := repositary.GetCampaignByID(ctx, userID, id)
	if err != nil {
		return dto.CampaignResponse{}, err
	}

	if c.Status == model.CampaignStatusQueued || c.Status == model.CampaignStatusInProgress {
		return dto.CampaignResponse{}, apperror.CampaignNotSendable.WithMessage(
			"This campaign is being sent and cannot be edited")
	}

	if req.TemplateID != c.TemplateID {
		if _, tErr := repositary.GetTemplateByID(ctx, userID, req.TemplateID); tErr != nil {
			return dto.CampaignResponse{}, tErr
		}
	}

	c.CampaignName = req.CampaignName
	c.Description = req.Description
	c.TemplateID = req.TemplateID
	c.AudienceDataSource = req.AudienceDataSource
	if req.Status != "" {
		c.Status = req.Status
	}

	if err := repositary.UpdateCampaign(ctx, c); err != nil {
		return dto.CampaignResponse{}, err
	}
	return mapToCampaignResponse(c), nil
}

func DeleteCampaign(ctx context.Context, id uint) *apperror.AppError {
	ctx, span := tracer.Start(ctx, "DeleteCampaign")
	defer span.End()

	userID := authctx.UserID(ctx)
	if _, err := repositary.GetCampaignByID(ctx, userID, id); err != nil {
		return err
	}
	return repositary.DeleteCampaign(ctx, userID, id)
}
