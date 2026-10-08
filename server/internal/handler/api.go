package handler

import (
	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/middleware"
	"domain-connect-backend/internal/utils"
	"domain-connect-backend/internal/views"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

func parseIDParam(r *http.Request) (uint, error) {
	idStr := strings.TrimSpace(r.URL.Query().Get("id"))
	if idStr == "" {
		return 0, errors.New("id query param is required")
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("invalid id")
	}
	return uint(id), nil
}

func (a *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims == nil {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	var req views.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode create api key body: %v", err)
		errorz.ErrBadRequest.SendError(w)
		return
	}

	req.WebHook = strings.TrimSpace(req.WebHook)
	if req.WebHook == "" {
		errorz.ErrInvalidInput.WithMessage("webhook is required").SendError(w)
		return
	}

	apikey, err := a.service.CreateAPIKey(claims.User.ID, req.WebHook)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("CreateApiKey failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, views.MessageResponse{Message: apikey})
}

func (a *AuthHandler) GetAllAPIKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims == nil {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	cursor := parseUintQuery(r, "cursor")
	limit := parseIntQuery(r, "limit")

	keys, err := a.service.GetAllAPIKey(claims.User.ID, cursor, limit)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("GetAllAPIKey failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, keys)
}

func (a *AuthHandler) UpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims == nil {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		errorz.ErrInvalidInput.WithMessage(err.Error()).SendError(w)
		return
	}

	var req views.UpdateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode update api key body: %v", err)
		errorz.ErrBadRequest.SendError(w)
		return
	}

	if req.WebHook == nil && req.Status == nil {
		errorz.ErrInvalidInput.WithMessage("at least one of web_hook or status is required").SendError(w)
		return
	}

	apikey, err := a.service.UpdateAPIKey(claims.User.ID, id, req)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("UpdateAPIKey failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, apikey)
}

func (a *AuthHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims == nil {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	id, err := parseIDParam(r)
	if err != nil {
		errorz.ErrInvalidInput.WithMessage(err.Error()).SendError(w)
		return
	}

	if err := a.service.DeleteAPIKey(claims.User.ID, id); err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("DeleteAPIKey failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, views.MessageResponse{Message: "API key deleted successfully"})
}
