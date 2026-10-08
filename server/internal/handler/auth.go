package handler

import (
	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/middleware"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/utils"
	"domain-connect-backend/internal/views"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
)

type AuthHandler struct {
	service service.AuthService
	oauth   service.OAuthService
}

func NewAuthHandler(service service.AuthService, oauth service.OAuthService) *AuthHandler {
	return &AuthHandler{service: service, oauth: oauth}
}

func (h *AuthHandler) Signup(w http.ResponseWriter, r *http.Request) {
	var req views.SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode signup body: %v", err)
		errorz.ErrBadRequest.SendError(w)
		return
	}

	if err := req.Valid(); err != nil {
		errorz.ErrInvalidInput.WithMessage(err.Error()).SendError(w)
		return
	}

	location, err := h.service.GeoLocation(middleware.ClientIP(r))
	if err != nil {
		log.Printf("Geo lookup error: %v", err)
		location = &views.GeoResponse{Country: "Unknown", City: "Unknown"}
	}

	resp, err := h.service.Signup(req, *location)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("Signup failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req views.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode login body: %v", err)
		errorz.ErrBadRequest.SendError(w)
		return
	}

	if err := req.Valid(); err != nil {
		errorz.ErrInvalidInput.WithMessage(err.Error()).SendError(w)
		return
	}

	resp, err := h.service.Login(req)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("Login failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	utils.SendSuccess(w, resp)
}

func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		errorz.ErrInvalidInput.SendError(w)
		return
	}

	resp, err := h.service.Verify(token)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	utils.SendSuccess(w, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	utils.ClearAuthCookies(w)
	utils.SendSuccess(w, views.MessageResponse{Message: "Logged out"})
}

func (h *AuthHandler) ForgetPassword(w http.ResponseWriter, r *http.Request) {
	var req views.ForgetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	if err := req.Valid(); err != nil {
		errorz.ErrInvalidInput.WithMessage(err.Error()).SendError(w)
		return
	}

	if err := h.service.ForgetPassword(req.Email); err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("ForgetPassword failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, views.MessageResponse{
		Message: "If an account exists for that email, a password reset link has been sent.",
	})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req views.ResetPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorz.ErrBadRequest.SendError(w)
		return
	}

	if err := req.Valid(); err != nil {
		errorz.ErrInvalidInput.WithMessage(err.Error()).SendError(w)
		return
	}

	if err := h.service.ResetPassword(req.Token, req.NewPassword); err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("ResetPassword failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, views.MessageResponse{
		Message: "Password has been reset successfully.",
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req views.RefreshRequest
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if strings.TrimSpace(req.RefreshToken) == "" {
		if c, err := r.Cookie(utils.RefreshTokenCookie); err == nil {
			req.RefreshToken = c.Value
		}
	}

	if !req.Valid() {
		errorz.ErrInvalidInput.SendError(w)
		return
	}

	resp, err := h.service.Refresh(req.RefreshToken)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	utils.SendSuccess(w, resp)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := r.Context().Value(middleware.UserContextKey).(*utils.Claims)
	if !ok || claims == nil {
		errorz.ErrUnauthorized.SendError(w)
		return
	}

	resp, err := h.service.Me(claims.User.ID)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		log.Printf("Me lookup failed: %v", err)
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SendSuccess(w, resp)
}

func (h *AuthHandler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	url, err := h.service.GoogleLogin(r.URL.Query().Get("redirect_uri"))
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		errorz.ErrInternalServer.SendError(w)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	location, err := h.service.GeoLocation(middleware.ClientIP(r))
	if err != nil {
		log.Printf("Geo lookup error: %v", err)
		location = &views.GeoResponse{Country: "Unknown", City: "Unknown"}
	}

	resp, err := h.service.GoogleCallback(code, state, *location)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	utils.SendSuccess(w, resp)
}

func (h *AuthHandler) GithubLogin(w http.ResponseWriter, r *http.Request) {
	url, err := h.service.GithubLogin(r.URL.Query().Get("redirect_uri"))
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		errorz.ErrInternalServer.SendError(w)
		return
	}
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *AuthHandler) GithubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	location, err := h.service.GeoLocation(middleware.ClientIP(r))
	if err != nil {
		log.Printf("Geo lookup error: %v", err)
		location = &views.GeoResponse{Country: "Unknown", City: "Unknown"}
	}

	resp, err := h.service.GithubCallback(code, state, *location)
	if err != nil {
		var appErr *errorz.AppError
		if errors.As(err, &appErr) {
			appErr.SendError(w)
			return
		}
		errorz.ErrInternalServer.SendError(w)
		return
	}

	utils.SetAuthCookies(w, resp.AccessToken, resp.RefreshToken)
	utils.SendSuccess(w, resp)
}
