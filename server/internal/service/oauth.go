package service

import (
	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/models"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/utils"
	"domain-connect-backend/internal/views"
	"net/url"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	authRequestTTL = 10 * time.Minute
	authCodeTTL    = 5 * time.Minute
)

type OAuthError struct {
	Code        string
	Description string
	Status      int
}

func (e *OAuthError) Error() string { return e.Code + ": " + e.Description }

func oauthErr(status int, code, desc string) *OAuthError {
	return &OAuthError{Code: code, Description: desc, Status: status}
}

type OAuthService interface {
	Metadata() views.OAuthASMetadata
	RegisterClient(req views.OAuthRegisterRequest) (*views.OAuthRegisterResponse, error)

	CreateAuthRequest(p views.OAuthAuthorizeParams) (*models.OAuthAuthRequest, error)
	GetAuthRequest(requestID string) (*models.OAuthAuthRequest, error)

	LoginPassword(requestID, email, password string) (*models.OAuthAuthRequest, error)
	CompleteSocialLogin(requestID, provider, code string, location views.GeoResponse) (*models.OAuthAuthRequest, error)
	GoogleAuthURL(requestID string) string

	IssueCode(requestID string) (redirectURL string, err error)
	DenyRedirect(requestID string) (redirectURL string, err error)

	ExchangeToken(req views.OAuthTokenRequest) (*views.OAuthTokenResponse, error)
}

type oauthService struct {
	repo     repository.OAuthRepository
	userRepo repository.UserRepository
	auth     AuthService
}

func NewOAuthService(repo repository.OAuthRepository, userRepo repository.UserRepository, auth AuthService) OAuthService {
	return &oauthService{repo: repo, userRepo: userRepo, auth: auth}
}

func issuer() string {
	return strings.TrimRight(config.AppConfig.OAuthIssuerURL, "/")
}

func (s *oauthService) Metadata() views.OAuthASMetadata {
	base := issuer()
	return views.OAuthASMetadata{
		Issuer:                            base,
		AuthorizationEndpoint:             base + "/oauth/authorize",
		TokenEndpoint:                     base + "/oauth/token",
		RegistrationEndpoint:              base + "/oauth/register",
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"none", "client_secret_post"},
	}
}

func (s *oauthService) RegisterClient(req views.OAuthRegisterRequest) (*views.OAuthRegisterResponse, error) {
	if len(req.RedirectURIs) == 0 {
		return nil, oauthErr(400, "invalid_redirect_uri", "at least one redirect_uri is required")
	}
	for _, u := range req.RedirectURIs {
		if _, err := url.ParseRequestURI(u); err != nil {
			return nil, oauthErr(400, "invalid_redirect_uri", "invalid redirect_uri: "+u)
		}
	}

	clientID, err := utils.RandomToken(24)
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to generate client id")
	}

	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}

	client := &models.OAuthClient{
		ClientID:                clientID,
		ClientName:              req.ClientName,
		GrantTypes:              strings.Join(grantTypes, " "),
		TokenEndpointAuthMethod: "none",
	}
	if err := client.SetRedirectURIs(req.RedirectURIs); err != nil {
		return nil, oauthErr(500, "server_error", "failed to store redirect uris")
	}
	if err := s.repo.CreateClient(client); err != nil {
		return nil, oauthErr(500, "server_error", "failed to register client")
	}

	return &views.OAuthRegisterResponse{
		ClientID:                client.ClientID,
		ClientName:              client.ClientName,
		RedirectURIs:            req.RedirectURIs,
		GrantTypes:              grantTypes,
		TokenEndpointAuthMethod: client.TokenEndpointAuthMethod,
	}, nil
}

func (s *oauthService) CreateAuthRequest(p views.OAuthAuthorizeParams) (*models.OAuthAuthRequest, error) {
	if p.ResponseType != "code" {
		return nil, oauthErr(400, "unsupported_response_type", "only response_type=code is supported")
	}
	if p.ClientID == "" {
		return nil, oauthErr(400, "invalid_request", "client_id is required")
	}

	client, err := s.repo.GetClient(p.ClientID)
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to load client")
	}
	if client == nil {
		return nil, oauthErr(400, "invalid_client", "unknown client_id")
	}
	if !redirectURIAllowed(client.RedirectURIList(), p.RedirectURI) {
		return nil, oauthErr(400, "invalid_request", "redirect_uri is not registered for this client")
	}

	// PKCE is mandatory for our public clients.
	if p.CodeChallenge == "" || p.CodeChallengeMethod != "S256" {
		return nil, oauthErr(400, "invalid_request", "code_challenge with code_challenge_method=S256 is required")
	}

	requestID, err := utils.RandomToken(24)
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to create request")
	}

	req := &models.OAuthAuthRequest{
		RequestID:           requestID,
		ClientID:            p.ClientID,
		RedirectURI:         p.RedirectURI,
		Scope:               p.Scope,
		State:               p.State,
		CodeChallenge:       p.CodeChallenge,
		CodeChallengeMethod: p.CodeChallengeMethod,
		ResponseType:        p.ResponseType,
		ExpiresAt:           time.Now().Add(authRequestTTL),
	}
	if err := s.repo.CreateAuthRequest(req); err != nil {
		return nil, oauthErr(500, "server_error", "failed to persist request")
	}
	return req, nil
}

func (s *oauthService) GetAuthRequest(requestID string) (*models.OAuthAuthRequest, error) {
	req, err := s.repo.GetAuthRequest(requestID)
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to load request")
	}
	if req == nil {
		return nil, oauthErr(400, "invalid_request", "authorization request not found")
	}
	if time.Now().After(req.ExpiresAt) {
		return nil, oauthErr(400, "invalid_request", "authorization request expired")
	}
	return req, nil
}

func (s *oauthService) LoginPassword(requestID, email, password string) (*models.OAuthAuthRequest, error) {
	req, err := s.GetAuthRequest(requestID)
	if err != nil {
		return nil, err
	}

	user, err := s.userRepo.GetByEmail(strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to load user")
	}
	if user == nil {
		return nil, oauthErr(401, "access_denied", "invalid email or password")
	}
	if bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)) != nil {
		return nil, oauthErr(401, "access_denied", "invalid email or password")
	}
	if !user.Verified {
		return nil, oauthErr(403, "access_denied", "email is not verified")
	}

	if err := s.repo.SetAuthRequestUser(requestID, user.ID); err != nil {
		return nil, oauthErr(500, "server_error", "failed to bind user")
	}
	req.UserID = &user.ID
	return req, nil
}

func (s *oauthService) GoogleAuthURL(requestID string) string {
	return s.auth.BuildGoogleAuthURLForMCP("asreq:" + requestID)
}

func (s *oauthService) CompleteSocialLogin(requestID, provider, code string, location views.GeoResponse) (*models.OAuthAuthRequest, error) {
	req, err := s.GetAuthRequest(requestID)
	if err != nil {
		return nil, err
	}

	var user *models.Users
	switch provider {
	case "google":
		user, err = s.auth.GoogleAuthenticateForMCP(code, location)
	default:
		return nil, oauthErr(400, "invalid_request", "unsupported provider")
	}
	if err != nil {
		return nil, oauthErr(401, "access_denied", "social login failed")
	}

	if err := s.repo.SetAuthRequestUser(requestID, user.ID); err != nil {
		return nil, oauthErr(500, "server_error", "failed to bind user")
	}
	req.UserID = &user.ID
	return req, nil
}

func (s *oauthService) IssueCode(requestID string) (string, error) {
	req, err := s.GetAuthRequest(requestID)
	if err != nil {
		return "", err
	}
	if req.UserID == nil {
		return "", oauthErr(401, "access_denied", "user is not authenticated")
	}

	code, err := utils.RandomToken(32)
	if err != nil {
		return "", oauthErr(500, "server_error", "failed to generate code")
	}

	authCode := &models.OAuthAuthCode{
		CodeHash:            utils.HashOAuthCode(code),
		ClientID:            req.ClientID,
		UserID:              *req.UserID,
		RedirectURI:         req.RedirectURI,
		Scope:               req.Scope,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(authCodeTTL),
	}
	if err := s.repo.CreateCode(authCode); err != nil {
		return "", oauthErr(500, "server_error", "failed to persist code")
	}

	_ = s.repo.DeleteAuthRequest(requestID)

	return appendQuery(req.RedirectURI, map[string]string{
		"code":  code,
		"state": req.State,
	}), nil
}

func (s *oauthService) DenyRedirect(requestID string) (string, error) {
	req, err := s.GetAuthRequest(requestID)
	if err != nil {
		return "", err
	}
	_ = s.repo.DeleteAuthRequest(requestID)
	return appendQuery(req.RedirectURI, map[string]string{
		"error":             "access_denied",
		"error_description": "the user denied the request",
		"state":             req.State,
	}), nil
}

func (s *oauthService) ExchangeToken(req views.OAuthTokenRequest) (*views.OAuthTokenResponse, error) {
	switch req.GrantType {
	case "authorization_code":
		return s.exchangeAuthorizationCode(req)
	case "refresh_token":
		return s.exchangeRefreshToken(req)
	default:
		return nil, oauthErr(400, "unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func (s *oauthService) exchangeAuthorizationCode(req views.OAuthTokenRequest) (*views.OAuthTokenResponse, error) {
	if req.Code == "" {
		return nil, oauthErr(400, "invalid_request", "code is required")
	}

	code, err := s.repo.GetCode(utils.HashOAuthCode(req.Code))
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to load code")
	}
	if code == nil || code.Used || time.Now().After(code.ExpiresAt) {
		return nil, oauthErr(400, "invalid_grant", "authorization code is invalid or expired")
	}
	if code.ClientID != req.ClientID {
		return nil, oauthErr(400, "invalid_grant", "client_id mismatch")
	}
	if code.RedirectURI != req.RedirectURI {
		return nil, oauthErr(400, "invalid_grant", "redirect_uri mismatch")
	}
	if !utils.VerifyPKCE(req.CodeVerifier, code.CodeChallenge, code.CodeChallengeMethod) {
		return nil, oauthErr(400, "invalid_grant", "PKCE verification failed")
	}

	if err := s.repo.MarkCodeUsed(code.ID); err != nil {
		return nil, oauthErr(500, "server_error", "failed to consume code")
	}

	user, err := s.userRepo.GetByID(code.UserID)
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to load user")
	}
	if user == nil {
		return nil, oauthErr(400, "invalid_grant", "user no longer exists")
	}

	pair, err := utils.GenerateTokenPair(utils.UserClaims{
		ID:       user.ID,
		Email:    user.Email,
		UserType: user.UserType,
	})
	if err != nil {
		return nil, oauthErr(500, "server_error", "failed to issue token")
	}

	return &views.OAuthTokenResponse{
		AccessToken:  pair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
		RefreshToken: pair.RefreshToken,
		Scope:        code.Scope,
	}, nil
}

func (s *oauthService) exchangeRefreshToken(req views.OAuthTokenRequest) (*views.OAuthTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, oauthErr(400, "invalid_request", "refresh_token is required")
	}
	pair, err := utils.RefreshAccessToken(req.RefreshToken)
	if err != nil {
		return nil, oauthErr(400, "invalid_grant", "refresh token is invalid or expired")
	}
	return &views.OAuthTokenResponse{
		AccessToken:  pair.AccessToken,
		TokenType:    "Bearer",
		ExpiresIn:    pair.ExpiresIn,
		RefreshToken: pair.RefreshToken,
	}, nil
}

func redirectURIAllowed(registered []string, candidate string) bool {
	if candidate == "" {
		return false
	}
	cand, err := url.Parse(candidate)
	if err != nil {
		return false
	}
	for _, r := range registered {
		if r == candidate {
			return true
		}
		reg, err := url.Parse(r)
		if err != nil {
			continue
		}
		if isLoopback(reg.Hostname()) && isLoopback(cand.Hostname()) &&
			reg.Scheme == cand.Scheme && reg.Path == cand.Path &&
			reg.Hostname() == cand.Hostname() {
			return true
		}
	}
	return false
}

func isLoopback(host string) bool {
	return host == "127.0.0.1" || host == "::1" || host == "localhost"
}

func appendQuery(base string, params map[string]string) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}
