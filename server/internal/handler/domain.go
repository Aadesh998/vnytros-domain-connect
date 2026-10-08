package handler

import (
	"domain-connect-backend/internal/errorz"
	appmiddleware "domain-connect-backend/internal/middleware"
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/utils"
	"domain-connect-backend/internal/views"
	"encoding/json"
	"errors"
	"log"
	"net/http"
)

func userIDFromContext(r *http.Request) (uint, bool) {
	if apiKey, ok := r.Context().Value(appmiddleware.ApiKeyContextKey).(*models.ApiKeys); ok && apiKey != nil {
		return apiKey.UserID, true
	}
	if claims, ok := r.Context().Value(appmiddleware.UserContextKey).(*utils.Claims); ok && claims != nil {
		return claims.User.ID, true
	}
	return 0, false
}

type DomainHandler struct {
	service service.DomainService
}

func NewDomainHandler(service service.DomainService) *DomainHandler {
	return &DomainHandler{service: service}
}

type DetectResponse struct {
	Domain      string            `json:"domain"`
	Nameservers []string          `json:"nameservers"`
	Provider    *models.Providers `json:"provider"`
}

func (h *DomainHandler) DetectProvider(w http.ResponseWriter, r *http.Request) {
	domainName := r.URL.Query().Get("domain")

	if domainName == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	domain, nameservers, provider, err := h.service.DetectProvider(domainName)
	if err != nil {
		if errors.Is(err, errorz.ErrDomainNotFound) {
			errorz.ErrDomainNotFound.SendError(w)
			return
		}
		log.Printf("Failed to detect provider for %s: %v", domainName, err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	resp := DetectResponse{
		Domain:      domain,
		Nameservers: nameservers,
		Provider:    provider,
	}

	utils.SendSuccess(w, resp)
}

func (h *DomainHandler) DNSLookup(w http.ResponseWriter, r *http.Request) {
	domainName := r.URL.Query().Get("domain")

	if domainName == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	resp, err := h.service.LookupDNS(domainName)
	if err != nil {
		if errors.Is(err, errorz.ErrDomainNotFound) {
			errorz.ErrDomainNotFound.SendError(w)
			return
		}
		if errors.Is(err, errorz.ErrInvalidInput) {
			errorz.ErrBadRequest.SendError(w)
			return
		}
		log.Printf("Failed to run DNS lookup for %s: %v", domainName, err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, resp)
}

func (h *DomainHandler) GetDomainStatus(w http.ResponseWriter, r *http.Request) {
	var req views.DomainStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		errorz.ErrBadRequest.SendError(w)
		return
	}

	if !req.Valid() {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	status, err := h.service.CheckDomainStatus(req)
	if err != nil {
		log.Printf("Failed to check domain status for %s: %v", req.Domain, err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, status)
}

func (h *DomainHandler) GetPublicKey(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetPublicKey()
	if err != nil {
		log.Printf("Failed to read public key: %v", err)
		errorz.ErrNotFound.SendError(w)
		return
	}

	w.Header().Set("Content-Type", "application/x-pem-file")
	w.Write(data)
}

func (h *DomainHandler) GetTemplate(w http.ResponseWriter, r *http.Request) {
	template, err := h.service.GetTemplate()
	if err != nil {
		log.Printf("Failed to get template: %v", err)
		errorz.ErrNotFound.SendError(w)
		return
	}

	utils.SendSuccess(w, template)
}

func (h *DomainHandler) HandleDCDiscovery(w http.ResponseWriter, r *http.Request) {
	discovery, err := h.service.GetDiscovery()
	if err != nil {
		log.Printf("Failed to get discovery info: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, discovery)
}

func (h *DomainHandler) VerifyDomain(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	var req views.DomainVerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorz.ErrBadRequest.SendError(w)
		return
	}
	if !req.Valid() {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	resp, err := h.service.VerifyDomain(userID, req.Domain)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("Failed to verify domain %s: %v", req.Domain, err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, resp)
}

func (h *DomainHandler) ListUserDomains(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	domains, err := h.service.ListUserDomains(userID)
	if err != nil {
		log.Printf("Failed to list user domains: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, domains)
}

func (h *DomainHandler) GetUserDomain(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	domainName := r.URL.Query().Get("domain")
	if domainName == "" {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	d, err := h.service.GetUserDomain(userID, domainName)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("Failed to get user domain: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, d)
}

func (h *DomainHandler) DirectProviderConnect(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromContext(r)
	if !ok {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	var req views.DirectConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		errorz.ErrBadRequest.SendError(w)
		return
	}

	if !req.Valid() {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	err := h.service.DirectProviderConnect(r.Context(), userID, req)
	if err != nil {
		log.Printf("Failed direct provider connect for %s: %v", req.Domain, err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, map[string]string{
		"message": "Records applied successfully and verification queued",
		"domain":  req.Domain,
	})
}
