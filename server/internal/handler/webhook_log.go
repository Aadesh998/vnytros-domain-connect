package handler

import (
	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/middleware"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/utils"
	"errors"
	"log"
	"net/http"
	"strconv"
)

type WebhookLogHandler struct {
	service service.WebhookLogService
}

func NewWebhookLogHandler(s service.WebhookLogService) *WebhookLogHandler {
	return &WebhookLogHandler{service: s}
}

func parseUintQuery(r *http.Request, key string) uint {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		return 0
	}
	return uint(n)
}

func parseIntQuery(r *http.Request, key string) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

func (h *WebhookLogHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims == nil {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	cursor := parseUintQuery(r, "cursor")
	limit := parseIntQuery(r, "limit")

	page, err := h.service.GetAll(claims.User.ID, cursor, limit)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("GetAll webhook logs failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, page)
}

func (h *WebhookLogHandler) GetFiltered(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims == nil {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	q := r.URL.Query()
	filters := service.WebhookLogFilters{
		Status:         q.Get("status"),
		Event:          q.Get("event"),
		ApiKeyID:       parseUintQuery(r, "api_key_id"),
		ResponseStatus: parseIntQuery(r, "response_status"),
		From:           q.Get("from"),
		To:             q.Get("to"),
	}

	cursor := parseUintQuery(r, "cursor")
	limit := parseIntQuery(r, "limit")

	page, err := h.service.GetWithFilters(claims.User.ID, filters, cursor, limit)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("GetFiltered webhook logs failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, page)
}
