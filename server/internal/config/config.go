package config

import (
	"domain-connect-backend/internal/views"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var AppConfig *views.Config

func LoadConfig() {
	// ENV_FILE picks the dotenv file to load; it defaults to .env.production.
	// Variables already set in the process environment win over the file.
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env.production"
	}
	err := godotenv.Load(envFile)
	if err != nil {
		log.Printf("Failed to load %s file.", envFile)
	}

	accessTTL, _ := strconv.Atoi(os.Getenv("JWT_ACCESS_TTL_MIN"))
	if accessTTL == 0 {
		accessTTL = 15
	}
	refreshTTL, _ := strconv.Atoi(os.Getenv("JWT_REFRESH_TTL_HOURS"))
	if refreshTTL == 0 {
		refreshTTL = 24 * 7
	}

	mailConcurrency, _ := strconv.Atoi(os.Getenv("MAIL_SEND_CONCURRENCY"))
	if mailConcurrency <= 0 {
		mailConcurrency = 8
	}

	AppConfig = &views.Config{
		DBUser:               os.Getenv("DB_USER"),
		DBPassword:           os.Getenv("DB_PASSWORD"),
		DBHost:               os.Getenv("DB_HOST"),
		DBName:               os.Getenv("DB_NAME"),
		DBPort:               os.Getenv("DB_PORT"),
		JWTSecret:            os.Getenv("JWT_SECRET"),
		JWTAccessTTL:         accessTTL,
		JWTRefreshTTL:        refreshTTL,
		GithubClientID:       os.Getenv("GITHUB_CLIENT_ID"),
		GithubSecret:         os.Getenv("GITHUB_CLIENT_SECRET"),
		GithubCallbackURL:    os.Getenv("GITHUB_CLIENT_CALLBACK_URL"),
		GoogleClientID:       os.Getenv("GOOGLE_CLIENT_ID"),
		GoogleSecret:         os.Getenv("GOOGLE_CLIENT_SECRET"),
		GoogleCallbackURL:    os.Getenv("GOOGLE_CLIENT_CALLBACK_URL"),
		SMTPFromEmail:        os.Getenv("SMTP_FROM_EMAIL"),
		SMTPPassword:         os.Getenv("SMTP_PASSWORD"),
		SMTPHost:             os.Getenv("SMTP_HOST"),
		SMTPPort:             os.Getenv("SMTP_PORT"),
		FrontendURL:          os.Getenv("FRONTEND_URL"),
		ApiKeySigning:        os.Getenv("APIKEY_SIGNING_SECRET"),
		RabitMQ:              os.Getenv("RABITMQ"),
		MCPBaseURL:           os.Getenv("MCP_SERVER_BASE_URL"),
		OAuthIssuerURL:       os.Getenv("OAUTH_ISSUER_URL"),
		GoogleMCPCallbackURL: os.Getenv("GOOGLE_MCP_CALLBACK_URL"),
		AuthCookieDomain:     os.Getenv("AUTH_COOKIE_DOMAIN"),
		AuthCookieSecure:     strings.EqualFold(os.Getenv("AUTH_COOKIE_SECURE"), "true"),
		WebhookSigningSecret: os.Getenv("WEBHOOK_SIGNING_SECRET"),
		MailSendConcurrency:  mailConcurrency,

		BaseURL:          strings.TrimRight(os.Getenv("BASE_URL"), "/"),
		DashboardURL:     strings.TrimRight(os.Getenv("DASHBOARD_URL"), "/"),
		AllowedOrigins:   splitList(os.Getenv("ALLOWED_ORIGINS")),
		ProductName:      firstNonEmpty(os.Getenv("PRODUCT_NAME"), "Vnytros"),
		SupportEmail:     firstNonEmpty(os.Getenv("SUPPORT_EMAIL"), os.Getenv("SMTP_FROM_EMAIL")),
		EmailLogoURL:     os.Getenv("EMAIL_LOGO_URL"),
		DCProviderDomain: strings.TrimSpace(os.Getenv("DC_PROVIDER_DOMAIN")),
	}

	AppConfig.FrontendURL = strings.TrimRight(AppConfig.FrontendURL, "/")
	if AppConfig.FrontendURL == "" {
		AppConfig.FrontendURL = "http://localhost:5173" // dashboard dev server
	}
	if AppConfig.DashboardURL == "" {
		AppConfig.DashboardURL = AppConfig.FrontendURL
	}
	if AppConfig.BaseURL == "" {
		// The api listens on :8000 (see internal/server).
		AppConfig.BaseURL = "http://localhost:8000"
	}

	AppConfig.OAuthAllowedCallbackURLs = allowedCallbackURLs(
		os.Getenv("OAUTH_ALLOWED_CALLBACK_URLS"),
		AppConfig.GoogleCallbackURL,
		AppConfig.GithubCallbackURL,
	)

	// Self-hosted defaults: everything resolves to this machine until the
	// operator points it at their own domain.
	if AppConfig.OAuthIssuerURL == "" {
		AppConfig.OAuthIssuerURL = AppConfig.BaseURL
	}
	if AppConfig.MCPBaseURL == "" {
		AppConfig.MCPBaseURL = "http://localhost:5000" // cmd/mcp listens on :5000
	}

	if AppConfig.GoogleMCPCallbackURL == "" {
		AppConfig.GoogleMCPCallbackURL = strings.TrimRight(AppConfig.OAuthIssuerURL, "/") + "/oauth/google/callback"
	}

	if AppConfig.JWTSecret == "" {
		log.Printf("WARN: JWT_SECRET is empty; tokens will not be secure")
	}
	if len(AppConfig.AllowedOrigins) == 0 {
		log.Printf("WARN: ALLOWED_ORIGINS is empty; browsers on other origins will be blocked by CORS")
	}
	if AppConfig.DCProviderDomain == "" {
		log.Printf("INFO: DC_PROVIDER_DOMAIN is empty; the Domain Connect template route is disabled")
	}
}

// RequireProduction fails fast when the values a public deployment cannot run
// without are missing. It is called by the server entrypoint only when
// ENV=production, so local development keeps working with an empty env.
func RequireProduction() {
	if !strings.EqualFold(os.Getenv("ENV"), "production") {
		return
	}
	var missing []string
	check := func(name, value string) {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	check("JWT_SECRET", AppConfig.JWTSecret)
	check("APIKEY_SIGNING_SECRET", AppConfig.ApiKeySigning)
	check("BASE_URL", os.Getenv("BASE_URL"))
	check("FRONTEND_URL", os.Getenv("FRONTEND_URL"))
	check("ALLOWED_ORIGINS", os.Getenv("ALLOWED_ORIGINS"))
	if len(missing) > 0 {
		log.Fatalf("config: ENV=production but required variables are unset: %s (see .env.example)",
			strings.Join(missing, ", "))
	}
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func splitList(raw string) []string {
	var out []string
	for _, p := range strings.Split(raw, ",") {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func allowedCallbackURLs(csv string, defaults ...string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(raw string) {
		u := strings.TrimSpace(raw)
		if u == "" || seen[u] {
			return
		}
		seen[u] = true
		out = append(out, u)
	}
	for _, d := range defaults {
		add(d)
	}
	for _, u := range strings.Split(csv, ",") {
		add(u)
	}
	return out
}
