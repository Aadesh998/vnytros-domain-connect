package middleware

import (
	"context"
	"net/http"
	"strings"

	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/utils"
)

const ApiKeyContextKey contextKey = "api_key"

func APIKeyAuth() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
				errorz.ErrUnauthorizedToken.SendError(w)
				return
			}

			token := header
			if strings.HasPrefix(strings.ToLower(header), "bearer ") {
				token = strings.TrimSpace(header[7:])
			}
			if token == "" {
				errorz.ErrInvalidAPIKey.SendError(w)
				return
			}

			payload, err := utils.DecodeAPIKey(token)
			if err != nil {
				errorz.ErrInvalidAPIKey.SendError(w)
				return
			}

			apiKey := &models.ApiKeys{
				ApiKey: token,
				UserID: payload.ID,
				User: models.Users{
					ID:       payload.ID,
					Name:     payload.Name,
					Email:    payload.Email,
					UserType: payload.UserType,
					Verified: payload.Verified,
				},
			}

			ctx := context.WithValue(r.Context(), ApiKeyContextKey, apiKey)
			next(w, r.WithContext(ctx))
		}
	}
}
