package errorz

import (
	"encoding/json"
	"log"
	"net/http"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}

var (
	ErrInternalServer    = &AppError{Code: "INTERNAL_SERVER_ERROR", Message: "An internal server error occurred", StatusCode: http.StatusInternalServerError}
	ErrInvalidInput      = &AppError{Code: "INVALID_INPUT", Message: "Invalid input provided", StatusCode: http.StatusBadRequest}
	ErrUnauthorized      = &AppError{Code: "UNAUTHORIZED", Message: "Unauthorized access", StatusCode: http.StatusUnauthorized}
	ErrForbidden         = &AppError{Code: "FORBIDDEN", Message: "Access forbidden", StatusCode: http.StatusForbidden}
	ErrNotFound          = &AppError{Code: "NOT_FOUND", Message: "Resource not found", StatusCode: http.StatusNotFound}
	ErrBadRequest        = &AppError{Code: "BAD_REQUEST", Message: "A Bad Request Body", StatusCode: http.StatusBadRequest}
	ErrDomainNotFound    = &AppError{Code: "DOMAIN_NOT_FOUND", Message: "Domain not found or no nameservers found", StatusCode: http.StatusNotFound}
	ErrUserExists        = &AppError{Code: "USER_EXISTS", Message: "User with this email already exists", StatusCode: http.StatusConflict}
	ErrInvalidCreds      = &AppError{Code: "INVALID_CREDENTIALS", Message: "Invalid email or password", StatusCode: http.StatusUnauthorized}
	ErrInvalidToken      = &AppError{Code: "INVALID_TOKEN", Message: "Invalid or expired verification token", StatusCode: http.StatusBadRequest}
	ErrTokenExpired      = &AppError{Code: "TOKEN_EXPIRED", Message: "Token has expired", StatusCode: http.StatusUnauthorized}
	ErrUnauthorizedToken = &AppError{Code: "UNAUTHORIZED_TOKEN", Message: "Invalid authentication token", StatusCode: http.StatusUnauthorized}
	ErrInvalidRefresh    = &AppError{Code: "INVALID_REFRESH", Message: "Invalid or expired refresh token", StatusCode: http.StatusUnauthorized}
	ErrInvalidAPIKey     = &AppError{Code: "INVALID_API_KEY", Message: "API key not recognized", StatusCode: http.StatusBadRequest}
	ErrAlreadyVerified   = &AppError{Code: "ALREADY_VERIFIED", Message: "Account is already verified", StatusCode: http.StatusConflict}
	ErrEmailNotVerified  = &AppError{Code: "EMAIL_UNVERIFIED", Message: "Email is not verified", StatusCode: http.StatusForbidden}
	ErrTooManyRequests   = &AppError{Code: "RATE_LIMITED", Message: "Too many requests, please slow down", StatusCode: http.StatusTooManyRequests}
)

func (a *AppError) Error() string {
	return a.Message
}

func (a *AppError) SendError(w http.ResponseWriter) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(a.StatusCode)
	if err := json.NewEncoder(w).Encode(a); err != nil {
		log.Printf("Failed to encode error response: %v", err)
	}
}

// WithMessage returns a copy carrying a custom message. It must not mutate the
// receiver: the exported errors above are package-level singletons, so mutating
// one would permanently change the message for every later caller.
func (a *AppError) WithMessage(message string) *AppError {
	clone := *a
	clone.Message = message
	return &clone
}

// wireError is the JSON body sent to clients. It carries each field twice: the
// lowercase keys used by the JS SDK and any new client, plus the capitalised
// keys the dashboard (dashboard/ in this monorepo) still reads. Drop the
// legacy trio once that dashboard has been migrated to the lowercase keys.
type wireError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`

	LegacyCode       string `json:"Code"`
	LegacyMessage    string `json:"Message"`
	LegacyStatusCode int    `json:"StatusCode"`
}

func (a *AppError) MarshalJSON() ([]byte, error) {
	return json.Marshal(wireError{
		Code:             a.Code,
		Message:          a.Message,
		StatusCode:       a.StatusCode,
		LegacyCode:       a.Code,
		LegacyMessage:    a.Message,
		LegacyStatusCode: a.StatusCode,
	})
}
