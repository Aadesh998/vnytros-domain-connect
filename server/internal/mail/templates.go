package mail

import (
	"domain-connect-backend/internal/config"
	"embed"
	"html"
	"log"
	"strings"
)

//go:embed html/*.html
var templateFS embed.FS

func loadTemplate(name string) string {
	data, err := templateFS.ReadFile("html/" + name)
	if err != nil {
		log.Printf("failed to load email template %s: %v", name, err)
		return ""
	}
	return string(data)
}

func RenderTemplate(name string, vars map[string]string) string {
	return renderTemplate(name, vars)
}

func renderTemplate(name string, vars map[string]string) string {
	body := loadTemplate(name)
	if body == "" {
		return ""
	}
	// Deployment-wide values first, so a caller value that happens to contain
	// "{{product_name}}" is not expanded a second time.
	for k, v := range globalVars() {
		body = strings.ReplaceAll(body, "{{"+k+"}}", v)
	}
	for k, v := range vars {
		body = strings.ReplaceAll(body, "{{"+k+"}}", v)
	}
	return body
}

// globalVars are the per-deployment values every transactional email can use.
// They come from configuration (PRODUCT_NAME, FRONTEND_URL, DASHBOARD_URL,
// SUPPORT_EMAIL, EMAIL_LOGO_URL); nothing points at a hosted instance.
func globalVars() map[string]string {
	productName, frontendURL, dashboardURL, supportEmail, logoURL := "Vnytros", "", "", "", ""
	if c := config.AppConfig; c != nil {
		if c.ProductName != "" {
			productName = c.ProductName
		}
		frontendURL, dashboardURL = c.FrontendURL, c.DashboardURL
		supportEmail, logoURL = c.SupportEmail, c.EmailLogoURL
	}
	if dashboardURL == "" {
		dashboardURL = frontendURL
	}
	logoCell := ""
	if logoURL != "" {
		logoCell = `<td style="padding-right:10px; vertical-align:middle;"><img src="` +
			html.EscapeString(logoURL) + `" width="28" height="28" alt="` + html.EscapeString(productName) +
			`" style="display:block; border:0; outline:none; text-decoration:none;" /></td>`
	}
	return map[string]string{
		"product_name":  html.EscapeString(productName),
		"frontend_url":  html.EscapeString(frontendURL),
		"dashboard_url": html.EscapeString(dashboardURL),
		"support_email": html.EscapeString(supportEmail),
		"logo_cell":     logoCell,
	}
}
