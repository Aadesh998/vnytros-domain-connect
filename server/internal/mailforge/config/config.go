package config

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// The watermark image and link have no default: there is no hosted asset to
// point at. With MAIL_WATERMARK_IMAGE_URL unset the footer is simply omitted.
const DefaultWatermarkLabel = "Sent with Vnytros"

type Config struct {
	JWTSecret         string
	RabbitMQURL       string
	BaseURL           string
	AllowedOrigins    []string
	Port              string
	Env               string
	SendBatchSize     int
	WatermarkImageURL string
	WatermarkLinkURL  string
	WatermarkLabel    string
	// WatermarkAllowedImageURLs are the extra images a user may pick for their
	// footer (MAIL_WATERMARK_ALLOWED_IMAGE_URLS, comma separated).
	WatermarkAllowedImageURLs []string
	EmailFrom                 string
	EmailPass                 string
	EmailHost                 string
	EmailPort                 string
}

var AppConfig *Config

func LoadConfig() {
	env := os.Getenv("ENV")
	if env == "" {
		env = "production"
	}

	envfile := os.Getenv("ENV_FILE")
	if envfile == "" {
		envfile = ".env." + env
	}
	if err := godotenv.Load(envfile); err != nil {
		log.Printf("config: %s not loaded (%v); relying on process environment", envfile, err)
	}

	batchSize, _ := strconv.Atoi(os.Getenv("SEND_BATCH_SIZE"))
	if batchSize <= 0 {
		batchSize = 50
	}

	AppConfig = &Config{
		JWTSecret:   os.Getenv("JWT_SECRET"),
		RabbitMQURL: os.Getenv("RABITMQ"),

		BaseURL:        os.Getenv("BASE_URL"),
		AllowedOrigins: splitList(os.Getenv("ALLOWED_ORIGINS")),

		// mailforge is mounted into cmd/server, which listens on :8000; PORT
		// only feeds the localhost BASE_URL fallback below.
		Port:          firstNonEmpty(os.Getenv("PORT"), "8000"),
		Env:           os.Getenv("ENV"),
		SendBatchSize: batchSize,

		WatermarkImageURL:         strings.TrimSpace(os.Getenv("MAIL_WATERMARK_IMAGE_URL")),
		WatermarkLinkURL:          strings.TrimSpace(os.Getenv("MAIL_WATERMARK_LINK_URL")),
		WatermarkLabel:            firstNonEmpty(os.Getenv("MAIL_WATERMARK_LABEL"), DefaultWatermarkLabel),
		WatermarkAllowedImageURLs: splitList(os.Getenv("MAIL_WATERMARK_ALLOWED_IMAGE_URLS")),

		// The api half of this repo already defines SMTP_* for the same
		// Mailgun account, so prefer those and keep EMAIL_* working as a
		// fallback for envs written before the two services merged.
		EmailFrom: firstNonEmpty(os.Getenv("SMTP_FROM_EMAIL"), os.Getenv("EMAIL_FROM")),
		EmailPass: firstNonEmpty(os.Getenv("SMTP_PASSWORD"), os.Getenv("EMAIL_PASSWORD")),
		EmailHost: firstNonEmpty(os.Getenv("SMTP_HOST"), os.Getenv("EMAIL_HOST")),
		EmailPort: firstNonEmpty(os.Getenv("SMTP_PORT"), os.Getenv("EMAIL_PORT")),
	}

	if AppConfig.BaseURL == "" {
		AppConfig.BaseURL = "http://localhost:" + AppConfig.Port
	}

	warnMissing()
}

func warnMissing() {
	if AppConfig.JWTSecret == "" {
		log.Printf("WARN: JWT_SECRET is empty; every authenticated request will be rejected")
	}
	if AppConfig.RabbitMQURL == "" {
		log.Printf("WARN: RABITMQ is empty; campaigns cannot be dispatched")
	}
	if len(AppConfig.AllowedOrigins) == 0 {
		log.Printf("WARN: ALLOWED_ORIGINS is empty; the dashboard will be blocked by CORS")
	}
	if AppConfig.Env == "production" && strings.HasPrefix(AppConfig.BaseURL, "http://localhost") {
		log.Printf("WARN: BASE_URL is still localhost in production; open tracking will not work")
	}
}

func firstNonEmpty(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func splitList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
