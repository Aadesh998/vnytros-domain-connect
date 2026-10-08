package views

type Config struct {
	DBUser               string
	DBPassword           string
	DBHost               string
	DBName               string
	DBPort               string
	JWTSecret            string
	JWTAccessTTL         int
	JWTRefreshTTL        int
	GithubCallbackURL    string
	GithubSecret         string
	GithubClientID       string
	GoogleClientID       string
	GoogleSecret         string
	GoogleCallbackURL    string
	SMTPFromEmail        string
	SMTPPassword         string
	SMTPHost             string
	SMTPPort             string
	FrontendURL          string
	ApiKeySigning        string
	RabitMQ              string
	MCPBaseURL           string
	OAuthIssuerURL       string
	GoogleMCPCallbackURL string
	AuthCookieDomain     string
	AuthCookieSecure     bool
	WebhookSigningSecret string

	OAuthAllowedCallbackURLs []string

	MailSendConcurrency int

	// Public URLs and identity of this deployment. Nothing here defaults to a
	// hosted instance: a self-hoster sets them for their own domain.
	BaseURL          string   // BASE_URL: public URL of this API
	DashboardURL     string   // DASHBOARD_URL: dashboard web app (defaults to FRONTEND_URL)
	AllowedOrigins   []string // ALLOWED_ORIGINS: CORS origins; "*.example.com" matches subdomains
	ProductName      string   // PRODUCT_NAME: name shown in emails and HTML pages
	SupportEmail     string   // SUPPORT_EMAIL: contact address in emails (defaults to SMTP_FROM_EMAIL)
	EmailLogoURL     string   // EMAIL_LOGO_URL: logo in transactional emails (omitted when empty)
	DCProviderDomain string   // DC_PROVIDER_DOMAIN: Domain Connect providerId / template domain
}
