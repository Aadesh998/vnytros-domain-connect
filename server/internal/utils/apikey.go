package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"domain-connect-backend/internal/config"
)

type APIKeyPayload struct {
	ID       uint   `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	UserType string `json:"user_type"`
	Verified bool   `json:"verified"`
	Nonce    string `json:"nonce"`
}

func GenerateAPIKey(p APIKeyPayload) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	p.Nonce = base64.RawURLEncoding.EncodeToString(nonce)

	raw, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	payloadEnc := base64.RawURLEncoding.EncodeToString(raw)
	return payloadEnc + "." + sign(payloadEnc), nil
}

func DecodeAPIKey(apikey string) (*APIKeyPayload, error) {
	parts := strings.SplitN(apikey, ".", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, errors.New("invalid api key format")
	}

	expected := sign(parts[0])
	if !hmac.Equal([]byte(expected), []byte(parts[1])) {
		return nil, errors.New("invalid api key signature")
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("invalid api key payload")
	}

	var p APIKeyPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return nil, errors.New("invalid api key payload")
	}
	if p.ID == 0 {
		return nil, errors.New("invalid api key user")
	}
	return &p, nil
}

func sign(data string) string {
	h := hmac.New(sha256.New, []byte(config.AppConfig.ApiKeySigning))
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}
