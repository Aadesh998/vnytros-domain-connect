package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"slices"
	"strings"

	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/utils"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserContextKey contextKey = "user"

func AuthorizeRoles(allowedRoles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			tokenString := r.Header.Get("Authorization")

			if tokenString == "" {
				queryToken := r.URL.Query().Get("token")
				if queryToken != "" {
					tokenString = "Bearer " + queryToken
					r.Header.Set("Authorization", tokenString)
				}
			}

			if tokenString == "" {
				if c, err := r.Cookie(utils.AccessTokenCookie); err == nil && c.Value != "" {
					tokenString = "Bearer " + c.Value
					r.Header.Set("Authorization", tokenString)
				}
			}

			if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
				w.Header().Set("WWW-Authenticate", `Bearer realm="api"`)
				errorz.ErrUnauthorizedToken.SendError(w)
				return
			}

			tokenString = strings.TrimPrefix(tokenString, "Bearer ")
			claims, err := utils.ParseAndCacheJWT(tokenString)
			if err != nil {
				log.Printf("ERROR: failed to get claims: %s", err)
				if errors.Is(err, jwt.ErrTokenExpired) {
					errorz.ErrTokenExpired.SendError(w)
					return
				}
				errorz.ErrUnauthorizedToken.SendError(w)
				return
			}

			userType := claims.User.UserType
			log.Printf("INFO: User Type: %s", userType)

			authorized := slices.Contains(allowedRoles, userType)

			if !authorized {
				errorz.ErrForbidden.SendError(w)
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next(w, r.WithContext(ctx))
		}
	}
}
