package middleware

import (
	"domain-connect-backend/internal/mailforge/apperror"
	"domain-connect-backend/internal/mailforge/authctx"
	"domain-connect-backend/internal/mailforge/config"
	"errors"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const accessTokenCookie = "access_token"

type userClaims struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	UserType string `json:"user_type"`
}

type claims struct {
	User    userClaims `json:"user"`
	Refresh bool       `json:"refresh,omitempty"`
	jwt.RegisteredClaims
}

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractToken(c)
		if raw == "" {
			apperror.ErrUnauthorized.SendError(c)
			c.Abort()
			return
		}

		parsed, err := parseToken(raw)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				apperror.TokenExpired.SendError(c)
			} else {
				log.Printf("auth: token rejected: %v", err)
				apperror.ErrUnauthorized.SendError(c)
			}
			c.Abort()
			return
		}

		if parsed.User.ID == 0 {
			apperror.ErrUnauthorized.SendError(c)
			c.Abort()
			return
		}

		principal := &authctx.Principal{
			UserID:   parsed.User.ID,
			Email:    parsed.User.Email,
			UserType: parsed.User.UserType,
		}

		c.Request = c.Request.WithContext(authctx.WithPrincipal(c.Request.Context(), principal))
		c.Next()
	}
}

// extractToken looks for the access token in the Authorization header first,
// then the cookie set by the /v1 auth handlers on the shared parent domain.
func extractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		if tok := strings.TrimSpace(strings.TrimPrefix(h, "Bearer ")); tok != "" {
			return tok
		}
	}

	if cookie, err := c.Request.Cookie(accessTokenCookie); err == nil && cookie.Value != "" {
		return cookie.Value
	}

	return ""
}

func parseToken(tokenString string) (*claims, error) {
	if config.AppConfig.JWTSecret == "" {
		return nil, errors.New("JWT secret not configured")
	}

	out := &claims{}
	token, err := jwt.ParseWithClaims(tokenString, out, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(config.AppConfig.JWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	if out.Refresh {
		return nil, errors.New("refresh token cannot be used as an access token")
	}

	return out, nil
}
