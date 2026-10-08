package service

import (
	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/errorz"
	"domain-connect-backend/internal/mail"
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/utils"
	"domain-connect-backend/internal/views"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthService interface {
	Signup(req views.SignupRequest, location views.GeoResponse) (*views.SignupResponse, error)
	Login(req views.LoginRequest) (*views.AuthResponse, error)
	Verify(token string) (*views.AuthResponse, error)
	Refresh(refreshToken string) (*views.AuthResponse, error)
	GoogleLogin(redirectURI string) (string, error)
	GoogleCallback(code, state string, location views.GeoResponse) (*views.AuthResponse, error)
	GithubLogin(redirectURI string) (string, error)
	GithubCallback(code, state string, location views.GeoResponse) (*views.AuthResponse, error)
	BuildGoogleAuthURL(state string) string
	BuildGithubAuthURL(state string) string
	BuildGoogleAuthURLForMCP(state string) string
	GoogleAuthenticateForMCP(code string, location views.GeoResponse) (*models.Users, error)
	GoogleAuthenticate(code, redirectURI string, location views.GeoResponse) (*models.Users, error)
	GithubAuthenticate(code, redirectURI string, location views.GeoResponse) (*models.Users, error)
	GeoLocation(ip string) (*views.GeoResponse, error)
	ForgetPassword(email string) error
	ResetPassword(token, newPassword string) error
	CreateAPIKey(user uint, webhook string) (string, error)
	UpdateAPIKey(userID, apikeyID uint, req views.UpdateAPIKeyRequest) (*models.ApiKeys, error)
	DeleteAPIKey(userID, apiKeyID uint) error
	GetAllAPIKey(userID uint, cursor uint, limit int) (*ApiKeyPage, error)
	Me(userID uint) (*views.MeResponse, error)
}

type authService struct {
	repo       repository.UserRepository
	apiKeyRepo repository.ApiKeyRepository
	domainRepo repository.DomainRepository
}

func NewAuthService(repo repository.UserRepository, apiKeyRepo repository.ApiKeyRepository, domainRepo repository.DomainRepository) AuthService {
	return &authService{
		repo:       repo,
		apiKeyRepo: apiKeyRepo,
		domainRepo: domainRepo,
	}
}

func (a *authService) Me(userID uint) (*views.MeResponse, error) {
	user, err := a.repo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errorz.ErrNotFound
	}

	resp := &views.MeResponse{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		UserType: user.UserType,
		Country:  user.Country,
		City:     user.City,
		Verified: user.Verified,
	}

	// domain_count is how many domains this user has connected.
	if a.domainRepo != nil {
		count, err := a.domainRepo.CountByUserID(userID)
		if err != nil {
			return nil, err
		}
		resp.DomainCount = count
	}

	return resp, nil
}

func (a *authService) Signup(req views.SignupRequest, location views.GeoResponse) (*views.SignupResponse, error) {
	existing, err := a.repo.GetByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errorz.ErrUserExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	token := uuid.New()
	verification := token.String()

	user := &models.Users{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hash),
		Country:  location.Country,
		City:     location.City,
		Token:    verification,
		UserType: "user",
	}

	if err := a.repo.Create(user); err != nil {
		return nil, err
	}

	verifyLink := fmt.Sprintf("%s/verify?token=%s", config.AppConfig.FrontendURL, verification)
	mail.Enqueue(user.Email, "Verify your email address", "verify_email.html", map[string]string{
		"name":        user.Name,
		"verify_link": verifyLink,
	})

	resp := views.SignupResponse{
		Message: "Successfully Send Verification mail to user.",
		Email:   user.Email,
	}

	return &resp, nil
}

func (a *authService) Login(req views.LoginRequest) (*views.AuthResponse, error) {
	user, err := a.repo.GetByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errorz.ErrInvalidCreds
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		return nil, errorz.ErrInvalidCreds
	}

	if !user.Verified {
		return nil, errorz.ErrEmailNotVerified
	}
	return buildAuthResponse(user)
}

func (a *authService) Verify(token string) (*views.AuthResponse, error) {
	if token == "" {
		return nil, errorz.ErrInvalidToken
	}

	user, err := a.repo.GetByToken(token)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errorz.ErrInvalidToken
	}
	if user.Verified {
		return nil, errorz.ErrAlreadyVerified
	}

	user.Verified = true
	user.Token = ""

	if err := a.repo.Update(user); err != nil {
		return nil, err
	}

	mail.Enqueue(user.Email, "Welcome to "+config.AppConfig.ProductName+"!", "welcome_email.html", map[string]string{
		"name": user.Name,
	})

	return buildAuthResponse(user)
}

func (a *authService) Refresh(refreshToken string) (*views.AuthResponse, error) {
	if refreshToken == "" {
		return nil, errorz.ErrInvalidRefresh
	}

	pair, err := utils.RefreshAccessToken(refreshToken)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, errorz.ErrTokenExpired
		}
		return nil, errorz.ErrInvalidRefresh
	}

	claims, err := utils.ParseAndCacheJWT(pair.AccessToken)
	if err != nil {
		return nil, err
	}

	return &views.AuthResponse{
		ID:           claims.User.ID,
		Email:        claims.User.Email,
		UserType:     claims.User.UserType,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

func buildAuthResponse(user *models.Users) (*views.AuthResponse, error) {
	pair, err := utils.GenerateTokenPair(utils.UserClaims{
		ID:       user.ID,
		Email:    user.Email,
		UserType: user.UserType,
	})
	if err != nil {
		return nil, err
	}

	return &views.AuthResponse{
		ID:           user.ID,
		Name:         user.Name,
		Email:        user.Email,
		UserType:     user.UserType,
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
	}, nil
}

func (a *authService) buildGoogleAuthURL(state, redirectURI string) string {
	oauth2Endpoint := "https://accounts.google.com/o/oauth2/v2/auth"
	params := url.Values{}
	params.Add("client_id", config.AppConfig.GoogleClientID)
	params.Add("redirect_uri", redirectURI)
	params.Add("response_type", "code")
	params.Add("scope", "https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile")
	params.Add("include_granted_scopes", "true")
	params.Add("state", state)

	return oauth2Endpoint + "?" + params.Encode()
}

func (a *authService) BuildGoogleAuthURL(state string) string {
	return a.buildGoogleAuthURL(state, config.AppConfig.GoogleCallbackURL)
}

func (a *authService) BuildGoogleAuthURLForMCP(state string) string {
	return a.buildGoogleAuthURL(state, config.AppConfig.GoogleMCPCallbackURL)
}

func (a *authService) GoogleLogin(redirectURI string) (string, error) {
	target, err := resolveOAuthRedirectURI(redirectURI, config.AppConfig.GoogleCallbackURL)
	if err != nil {
		return "", err
	}
	return a.buildGoogleAuthURL(encodeOAuthState("google-auth", target), target), nil
}

func (a *authService) GoogleAuthenticate(code, redirectURI string, location views.GeoResponse) (*models.Users, error) {
	target, err := resolveOAuthRedirectURI(redirectURI, config.AppConfig.GoogleCallbackURL)
	if err != nil {
		return nil, err
	}
	return a.googleAuthenticate(code, target, location)
}

func (a *authService) GoogleAuthenticateForMCP(code string, location views.GeoResponse) (*models.Users, error) {
	return a.googleAuthenticate(code, config.AppConfig.GoogleMCPCallbackURL, location)
}

func (a *authService) googleAuthenticate(code, redirectURI string, location views.GeoResponse) (*models.Users, error) {
	if code == "" {
		return nil, errorz.ErrInvalidInput
	}

	token, err := a.getGoogleAccessToken(code, redirectURI)
	if err != nil {
		log.Printf("ERROR: Failed to exchange Google token: %v", err)
		return nil, errorz.ErrInternalServer
	}

	userInfo, err := a.getGoogleUserInfo(token)
	if err != nil {
		log.Printf("ERROR: Failed to fetch Google user info: %v", err)
		return nil, errorz.ErrUnauthorized
	}

	user, err := a.repo.GetByEmail(userInfo.Email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user = &models.Users{
			Name:     userInfo.Name,
			Email:    userInfo.Email,
			Password: uuid.New().String(),
			Country:  location.Country,
			City:     location.City,
			Verified: true,
			UserType: "user",
		}
		if err := a.repo.Create(user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (a *authService) GoogleCallback(code, state string, location views.GeoResponse) (*views.AuthResponse, error) {
	user, err := a.GoogleAuthenticate(code, decodeOAuthState(state), location)
	if err != nil {
		return nil, err
	}
	return buildAuthResponse(user)
}

func (a *authService) getGoogleAccessToken(code, redirectURI string) (string, error) {
	tokenURL := "https://oauth2.googleapis.com/token"

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", config.AppConfig.GoogleClientID)
	data.Set("client_secret", config.AppConfig.GoogleSecret)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	resp, err := http.PostForm(tokenURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token exchange failed: %s", string(body))
	}

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

type googleUserInfo struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

func (a *authService) getGoogleUserInfo(accessToken string) (*googleUserInfo, error) {
	req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to fetch user info: %s", string(body))
	}

	var userInfo googleUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

func (a *authService) BuildGithubAuthURL(state string) string {
	return a.buildGithubAuthURL(state, config.AppConfig.GithubCallbackURL)
}

func (a *authService) buildGithubAuthURL(state, redirectURI string) string {
	githubURL := "https://github.com/login/oauth/authorize"
	params := url.Values{}
	params.Add("client_id", config.AppConfig.GithubClientID)
	params.Add("redirect_uri", redirectURI)
	params.Add("response_type", "code")
	params.Add("state", state)
	params.Add("scope", "read:user, user:email")
	params.Add("allow_signup", "true")

	return githubURL + "?" + params.Encode()
}

func (a *authService) GithubLogin(redirectURI string) (string, error) {
	target, err := resolveOAuthRedirectURI(redirectURI, config.AppConfig.GithubCallbackURL)
	if err != nil {
		return "", err
	}
	return a.buildGithubAuthURL(encodeOAuthState("github-auth", target), target), nil
}

func (a *authService) GithubAuthenticate(code, redirectURI string, location views.GeoResponse) (*models.Users, error) {
	if code == "" {
		return nil, errorz.ErrInvalidInput
	}

	target, err := resolveOAuthRedirectURI(redirectURI, config.AppConfig.GithubCallbackURL)
	if err != nil {
		return nil, err
	}

	token, err := a.getGithubAccessToken(code, target)
	if err != nil {
		log.Printf("ERROR: Failed to get GitHub access token: %v", err)
		return nil, errorz.ErrInternalServer
	}

	userInfo, err := a.getGithubUserProfile(token)
	if err != nil {
		log.Printf("ERROR: Failed to get GitHub user profile: %v", err)
		return nil, errorz.ErrUnauthorized
	}

	user, err := a.repo.GetByEmail(userInfo.Email)
	if err != nil {
		return nil, err
	}

	if user == nil {
		user = &models.Users{
			Name:     userInfo.Name,
			Email:    userInfo.Email,
			Password: uuid.New().String(),
			Country:  location.Country,
			City:     location.City,
			Verified: true,
			UserType: "user",
		}
		if err := a.repo.Create(user); err != nil {
			return nil, err
		}
	}

	return user, nil
}

func (a *authService) GithubCallback(code, state string, location views.GeoResponse) (*views.AuthResponse, error) {
	user, err := a.GithubAuthenticate(code, decodeOAuthState(state), location)
	if err != nil {
		return nil, err
	}
	return buildAuthResponse(user)
}

func (a *authService) getGithubAccessToken(code, redirectURI string) (string, error) {
	tokenURL := "https://github.com/login/oauth/access_token"

	data := url.Values{}
	data.Set("client_id", config.AppConfig.GithubClientID)
	data.Set("client_secret", config.AppConfig.GithubSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if result.AccessToken == "" {
		return "", errors.New("failed to get access token from response")
	}
	return result.AccessToken, nil
}

func (a *authService) getGithubUserProfile(accessToken string) (*views.GithubUserInfo, error) {
	client := &http.Client{}

	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	var user views.GithubUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, err
	}

	if user.Name == "" {
		user.Name = user.Login
	}

	emailReq, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	emailReq.Header.Set("Authorization", "Bearer "+accessToken)
	emailResp, err := client.Do(emailReq)
	if err != nil {
		return nil, err
	}
	defer emailResp.Body.Close()

	if emailResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub Email API returned status: %d", emailResp.StatusCode)
	}

	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
		return nil, err
	}

	for _, e := range emails {
		if e.Primary && e.Verified {
			user.Email = e.Email
			break
		}
	}

	if user.Email == "" && len(emails) > 0 {
		user.Email = emails[0].Email
	}

	return &user, nil
}

func (a *authService) ForgetPassword(email string) error {
	user, err := a.repo.GetByEmail(email)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}

	resetToken := uuid.New().String()
	user.Token = resetToken

	if err := a.repo.Update(user); err != nil {
		return err
	}

	resetLink := fmt.Sprintf("%s/reset-password?token=%s", config.AppConfig.FrontendURL, resetToken)
	mail.Enqueue(user.Email, "Reset your password", "reset_password.html", map[string]string{
		"name":       user.Name,
		"reset_link": resetLink,
	})

	return nil
}

func (a *authService) ResetPassword(token, newPassword string) error {
	if token == "" {
		return errorz.ErrInvalidToken
	}

	user, err := a.repo.GetByToken(token)
	if err != nil {
		return err
	}
	if user == nil {
		return errorz.ErrInvalidToken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hash)
	user.Token = ""

	return a.repo.Update(user)
}

func (a *authService) GeoLocation(ip string) (*views.GeoResponse, error) {
	unknown := &views.GeoResponse{Country: "Unknown", City: "Unknown"}

	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}

	parsed := net.ParseIP(ip)
	if ip == "" || parsed == nil || parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsUnspecified() {
		return unknown, nil
	}

	resp, err := http.Get("http://ip-api.com/json/" + ip)
	if err != nil {
		log.Printf("ERROR: Failed to get geolocation for IP %s: %v", ip, err)
		return unknown, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("WARN: Geolocation lookup HTTP %d for IP %s", resp.StatusCode, ip)
		return unknown, nil
	}

	var geolocation views.GeoResponse
	if err := json.NewDecoder(resp.Body).Decode(&geolocation); err != nil {
		log.Printf("ERROR: Failed to decode geolocation data: %v", err)
		return unknown, nil
	}

	if geolocation.Status != "success" {
		log.Printf("WARN: Geolocation lookup failed for IP %s: %s", ip, geolocation.Message)
		return unknown, nil
	}

	return &geolocation, nil
}

func (a *authService) CreateAPIKey(userID uint, webhook string) (string, error) {
	log.Printf("CreateAPIKey: start user_id=%d webhook=%q", userID, webhook)

	webhook = strings.TrimSpace(webhook)
	if webhook == "" {
		log.Printf("CreateAPIKey: rejected empty webhook for user_id=%d", userID)
		return "", errorz.ErrInvalidInput.WithMessage("webhook is required")
	}

	user, err := a.repo.GetByID(userID)
	if err != nil {
		log.Printf("CreateAPIKey: repo.GetByID(%d) failed: %v", userID, err)
		return "", err
	}
	if user == nil {
		log.Printf("CreateAPIKey: user not found id=%d", userID)
		return "", errorz.ErrUnauthorized
	}
	log.Printf("CreateAPIKey: loaded user id=%d email=%s user_type=%s verified=%v",
		user.ID, user.Email, user.UserType, user.Verified)

	api, err := utils.GenerateAPIKey(utils.APIKeyPayload{
		ID:       user.ID,
		Name:     user.Name,
		Email:    user.Email,
		UserType: user.UserType,
		Verified: user.Verified,
	})
	if err != nil {
		log.Printf("CreateAPIKey: GenerateAPIKey failed for user_id=%d: %v", user.ID, err)
		return "", err
	}
	log.Printf("CreateAPIKey: generated api key for user_id=%d (len=%d)", user.ID, len(api))

	if err := a.apiKeyRepo.CreateAPIKey(models.ApiKeys{
		ApiKey:  api,
		WebHook: webhook,
		UserID:  user.ID,
	}); err != nil {
		log.Printf("CreateAPIKey: apiKeyRepo.CreateAPIKey failed for user_id=%d: %v", user.ID, err)
		return "", err
	}

	log.Printf("CreateAPIKey: success user_id=%d", user.ID)
	return api, nil
}

type ApiKeyPage struct {
	Items      []*models.ApiKeys `json:"items"`
	NextCursor uint              `json:"next_cursor,omitempty"`
	Limit      int               `json:"limit"`
	HasMore    bool              `json:"has_more"`
}

func (a *authService) GetAllAPIKey(userID uint, cursor uint, limit int) (*ApiKeyPage, error) {
	limit = utils.ClampPageSize(limit)
	items, err := a.apiKeyRepo.GetAllByUserID(userID, cursor, limit)
	if err != nil {
		return nil, err
	}
	data, next, hasMore := utils.BuildPage(items, limit, func(k *models.ApiKeys) uint { return k.ID })
	return &ApiKeyPage{Items: data, NextCursor: next, Limit: limit, HasMore: hasMore}, nil
}

func (a *authService) UpdateAPIKey(userID, apikeyID uint, req views.UpdateAPIKeyRequest) (*models.ApiKeys, error) {
	existing, err := a.apiKeyRepo.GetByID(apikeyID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, errorz.ErrNotFound
	}
	if existing.UserID != userID {
		return nil, errorz.ErrForbidden
	}

	updates := map[string]any{}
	if req.WebHook != nil {
		wh := strings.TrimSpace(*req.WebHook)
		if wh == "" {
			return nil, errorz.ErrInvalidInput.WithMessage("webhook cannot be empty")
		}
		updates["web_hook"] = wh
	}
	if req.Status != nil {
		s := strings.ToLower(strings.TrimSpace(*req.Status))
		switch s {
		case "active", "disable", "cancelled":
		default:
			return nil, errorz.ErrInvalidInput.WithMessage("status must be one of active, disable, cancelled")
		}
		updates["status"] = s
	}

	if len(updates) == 0 {
		return existing, nil
	}

	if err := a.apiKeyRepo.UpdateByID(apikeyID, updates); err != nil {
		return nil, err
	}
	return a.apiKeyRepo.GetByID(apikeyID)
}

func (a *authService) DeleteAPIKey(userID, apikeyID uint) error {
	existing, err := a.apiKeyRepo.GetByID(apikeyID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errorz.ErrNotFound
	}
	if existing.UserID != userID {
		return errorz.ErrForbidden
	}
	return a.apiKeyRepo.DeleteByID(apikeyID)
}

const oauthStateSep = "|"

func resolveOAuthRedirectURI(requested, fallback string) (string, error) {
	requested = strings.TrimSpace(requested)
	if requested == "" {
		return fallback, nil
	}
	for _, allowed := range config.AppConfig.OAuthAllowedCallbackURLs {
		if allowed == requested {
			return requested, nil
		}
	}
	return "", errorz.ErrInvalidInput
}

func encodeOAuthState(prefix, redirectURI string) string {
	if redirectURI == "" {
		return prefix
	}
	return prefix + oauthStateSep + base64.RawURLEncoding.EncodeToString([]byte(redirectURI))
}

func decodeOAuthState(state string) string {
	_, encoded, found := strings.Cut(state, oauthStateSep)
	if !found {
		return ""
	}
	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return ""
	}
	// Re-validated by resolveOAuthRedirectURI in the caller.
	return string(raw)
}
