package utils

import (
	"domain-connect-backend/internal/config"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	ID       uint   `json:"id"`
	Email    string `json:"email"`
	UserType string `json:"user_type"`
}

type Claims struct {
	User    UserClaims `json:"user"`
	Refresh bool       `json:"refresh,omitempty"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type cachedClaims struct {
	claims    *Claims
	expiresAt time.Time
}

var (
	jwtCache   = make(map[string]cachedClaims)
	jwtCacheMu sync.RWMutex
)

func secret() []byte {
	return []byte(config.AppConfig.JWTSecret)
}

func GenerateTokenPair(user UserClaims) (*TokenPair, error) {
	accessTTL := time.Duration(config.AppConfig.JWTAccessTTL) * time.Minute
	refreshTTL := time.Duration(config.AppConfig.JWTRefreshTTL) * time.Hour

	access, err := signToken(user, accessTTL, false)
	if err != nil {
		return nil, err
	}
	refresh, err := signToken(user, refreshTTL, true)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    int64(accessTTL.Seconds()),
	}, nil
}

func signToken(user UserClaims, ttl time.Duration, isRefresh bool) (string, error) {
	now := time.Now()
	claims := Claims{
		User:    user,
		Refresh: isRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			Subject:   fmt.Sprintf("%d", user.ID),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(secret())
}

func ParseAndCacheJWT(tokenString string) (*Claims, error) {
	jwtCacheMu.RLock()
	if entry, ok := jwtCache[tokenString]; ok && time.Now().Before(entry.expiresAt) {
		jwtCacheMu.RUnlock()
		return entry.claims, nil
	}
	jwtCacheMu.RUnlock()

	claims, err := parseToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.Refresh {
		return nil, errors.New("refresh token cannot be used as access token")
	}

	exp := time.Now().Add(5 * time.Minute)
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(exp) {
		exp = claims.ExpiresAt.Time
	}

	jwtCacheMu.Lock()
	jwtCache[tokenString] = cachedClaims{claims: claims, expiresAt: exp}
	if len(jwtCache) > 1024 {
		evictExpiredLocked()
	}
	jwtCacheMu.Unlock()

	return claims, nil
}

func parseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret(), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func RefreshAccessToken(refreshToken string) (*TokenPair, error) {
	claims, err := parseToken(refreshToken)
	if err != nil {
		return nil, err
	}
	if !claims.Refresh {
		return nil, errors.New("not a refresh token")
	}

	jwtCacheMu.Lock()
	delete(jwtCache, refreshToken)
	jwtCacheMu.Unlock()

	return GenerateTokenPair(claims.User)
}

func evictExpiredLocked() {
	now := time.Now()
	for k, v := range jwtCache {
		if now.After(v.expiresAt) {
			delete(jwtCache, k)
		}
	}
}
