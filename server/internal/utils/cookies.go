package utils

import (
	"net/http"
	"time"

	"domain-connect-backend/internal/config"
)

const (
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
)

func SetAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	cfg := config.AppConfig
	domain := ""
	secure := false
	accessTTL := 15
	refreshTTL := 24 * 7
	if cfg != nil {
		domain = cfg.AuthCookieDomain
		secure = cfg.AuthCookieSecure
		if cfg.JWTAccessTTL > 0 {
			accessTTL = cfg.JWTAccessTTL
		}
		if cfg.JWTRefreshTTL > 0 {
			refreshTTL = cfg.JWTRefreshTTL
		}
	}

	if accessToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     AccessTokenCookie,
			Value:    accessToken,
			Path:     "/",
			Domain:   domain,
			Expires:  time.Now().Add(time.Duration(accessTTL) * time.Minute),
			MaxAge:   accessTTL * 60,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}

	if refreshToken != "" {
		http.SetCookie(w, &http.Cookie{
			Name:     RefreshTokenCookie,
			Value:    refreshToken,
			Path:     "/",
			Domain:   domain,
			Expires:  time.Now().Add(time.Duration(refreshTTL) * time.Hour),
			MaxAge:   refreshTTL * 3600,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

func ClearAuthCookies(w http.ResponseWriter) {
	cfg := config.AppConfig
	domain := ""
	secure := false
	if cfg != nil {
		domain = cfg.AuthCookieDomain
		secure = cfg.AuthCookieSecure
	}

	for _, name := range []string{AccessTokenCookie, RefreshTokenCookie} {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			Domain:   domain,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
			HttpOnly: true,
			Secure:   secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}
