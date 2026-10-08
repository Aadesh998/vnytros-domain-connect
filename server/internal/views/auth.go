package views

import (
	"errors"
	"strings"
)

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *SignupRequest) Valid() error {
	s.Name = strings.TrimSpace(s.Name)
	s.Email = strings.ToLower(strings.TrimSpace(s.Email))

	if s.Name == "" {
		return errors.New("name is required")
	}
	if s.Email == "" {
		return errors.New("email is required")
	}
	if len(s.Password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	return nil
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (l *LoginRequest) Valid() error {
	l.Email = strings.ToLower(strings.TrimSpace(l.Email))

	if l.Email == "" {
		return errors.New("email is required")
	}
	if l.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

type AuthResponse struct {
	ID           uint   `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	UserType     string `json:"user_type,omitempty"`
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int64  `json:"expires_in,omitempty"`
}

type SignupResponse struct {
	Message string `json:"message"`
	Email   string `json:"email"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (r *RefreshRequest) Valid() bool {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	return r.RefreshToken != ""
}

type ForgetPasswordRequest struct {
	Email string `json:"email"`
}

func (f *ForgetPasswordRequest) Valid() error {
	f.Email = strings.ToLower(strings.TrimSpace(f.Email))
	if f.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

type ResetPasswordRequest struct {
	Token           string `json:"token"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (f *ResetPasswordRequest) Valid() error {
	f.Token = strings.TrimSpace(f.Token)
	if f.Token == "" {
		return errors.New("reset token is required")
	}
	if f.NewPassword == "" {
		return errors.New("new password is required")
	}
	if f.ConfirmPassword == "" {
		return errors.New("confirm password is required")
	}
	if len(f.NewPassword) < 8 {
		return errors.New("password must be at least 8 characters long")
	}
	if f.NewPassword != f.ConfirmPassword {
		return errors.New("passwords do not match")
	}
	return nil
}

type MessageResponse struct {
	Message string `json:"message"`
}

type MeResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
	UserType    string `json:"user_type"`
	Country     string `json:"country,omitempty"`
	City        string `json:"city,omitempty"`
	Verified    bool   `json:"verified"`
	DomainCount int64  `json:"domain_count"`
}

type CreateAPIKeyRequest struct {
	WebHook string `json:"web_hook"`
}

type UpdateAPIKeyRequest struct {
	WebHook *string `json:"web_hook,omitempty"`
	Status  *string `json:"status,omitempty"`
}

type GeoResponse struct {
	Status  string  `json:"status"`
	Message string  `json:"message,omitempty"`
	Country string  `json:"country"`
	City    string  `json:"city"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
}

type GithubUserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
	Login string `json:"login"`
}
