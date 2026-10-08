package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/utils"

	"github.com/modelcontextprotocol/go-sdk/auth"
)

var neverExpires = time.Date(9999, time.December, 31, 23, 59, 59, 0, time.UTC)

func MCPTokenVerifier(ctx context.Context, token string, r *http.Request) (*auth.TokenInfo, error) {
	if token == "" {
		return nil, fmt.Errorf("%w: bearer token is missing", auth.ErrInvalidToken)
	}

	claims, err := utils.ParseAndCacheJWT(token)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", auth.ErrInvalidToken, err)
	}

	exp := neverExpires
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	return &auth.TokenInfo{
		UserID:     fmt.Sprint(claims.User.ID),
		Expiration: exp,
		Extra: map[string]any{
			"id":        claims.User.ID,
			"email":     claims.User.Email,
			"user_type": claims.User.UserType,
		},
	}, nil
}

func UserFromContext(ctx context.Context) (*models.ApiKeys, bool) {
	info := auth.TokenInfoFromContext(ctx)
	if info == nil || info.Extra == nil {
		return nil, false
	}

	id, ok := toUint(info.Extra["id"])
	if !ok || id == 0 {
		return nil, false
	}

	apiKey := &models.ApiKeys{
		ApiKey: asString(info.Extra["api_key"]),
		UserID: id,
		User: models.Users{
			ID:       id,
			Name:     asString(info.Extra["name"]),
			Email:    asString(info.Extra["email"]),
			UserType: asString(info.Extra["user_type"]),
			Verified: asBool(info.Extra["verified"]),
		},
	}
	return apiKey, true
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}

func toUint(v any) (uint, bool) {
	switch n := v.(type) {
	case uint:
		return n, true
	case int:
		return uint(n), true
	case int64:
		return uint(n), true
	case float64:
		return uint(n), true
	default:
		return 0, false
	}
}
