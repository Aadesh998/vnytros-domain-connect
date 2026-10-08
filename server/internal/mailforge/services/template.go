package services

import (
	"context"
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/authctx"
	"domain-connect-backend/internal/mailforge/dto"
	"domain-connect-backend/internal/mailforge/model"
	"domain-connect-backend/internal/mailforge/repositary"
)

func CreateTemplate(ctx context.Context, req dto.TemplateRequest) (dto.TemplateResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "CreateTemplate")
	defer span.End()

	temp := model.Template{
		UserID:  authctx.UserID(ctx),
		Name:    req.Name,
		Subject: req.Subject,
		Body:    req.Body,
		Status:  req.Status,
	}
	if temp.Status == "" {
		temp.Status = "draft"
	}

	if err := repositary.SaveTemplate(ctx, &temp); err != nil {
		return dto.TemplateResponse{}, err
	}
	return mapTemplate(temp), nil
}

func GetTemplates(ctx context.Context, lastID uint, limit int) (dto.TemplatePaginationResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetTemplates")
	defer span.End()

	templates, err := repositary.GetTemplates(ctx, authctx.UserID(ctx), lastID, limit)
	if err != nil {
		return dto.TemplatePaginationResponse{}, err
	}
	return paginateTemplates(templates), nil
}

func GetDraftTemplates(ctx context.Context, lastID uint, limit int) (dto.TemplatePaginationResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetDraftTemplates")
	defer span.End()

	templates, err := repositary.GetDraftTemplates(ctx, authctx.UserID(ctx), lastID, limit)
	if err != nil {
		return dto.TemplatePaginationResponse{}, err
	}
	return paginateTemplates(templates), nil
}

func GetTemplate(ctx context.Context, id uint) (dto.TemplateResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "GetTemplate")
	defer span.End()

	temp, err := repositary.GetTemplateByID(ctx, authctx.UserID(ctx), id)
	if err != nil {
		return dto.TemplateResponse{}, err
	}
	return mapTemplate(temp), nil
}

func UpdateTemplate(ctx context.Context, id uint, req dto.TemplateRequest) (dto.TemplateResponse, *apperror.AppError) {
	ctx, span := tracer.Start(ctx, "UpdateTemplate")
	defer span.End()

	temp, err := repositary.GetTemplateByID(ctx, authctx.UserID(ctx), id)
	if err != nil {
		return dto.TemplateResponse{}, err
	}

	temp.Name = req.Name
	temp.Subject = req.Subject
	temp.Body = req.Body
	if req.Status != "" {
		temp.Status = req.Status
	}

	if err := repositary.UpdateTemplate(ctx, temp); err != nil {
		return dto.TemplateResponse{}, err
	}
	return mapTemplate(temp), nil
}

func DeleteTemplate(ctx context.Context, id uint) *apperror.AppError {
	ctx, span := tracer.Start(ctx, "DeleteTemplate")
	defer span.End()

	userID := authctx.UserID(ctx)
	if _, err := repositary.GetTemplateByID(ctx, userID, id); err != nil {
		return err
	}
	return repositary.DeleteTemplate(ctx, userID, id)
}

func mapTemplate(t model.Template) dto.TemplateResponse {
	return dto.TemplateResponse{
		ID:        t.ID,
		Name:      t.Name,
		Subject:   t.Subject,
		Status:    t.Status,
		Body:      t.Body,
		CreatedAt: t.CreatedAt,
	}
}

func paginateTemplates(templates []model.Template) dto.TemplatePaginationResponse {
	out := make([]dto.TemplateResponse, 0, len(templates))
	for _, t := range templates {
		out = append(out, mapTemplate(t))
	}

	var nextID uint
	if len(templates) > 0 {
		nextID = templates[len(templates)-1].ID
	}

	return dto.TemplatePaginationResponse{Templates: out, NextID: nextID}
}
