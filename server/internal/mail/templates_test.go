package mail

import (
	"strings"
	"testing"

	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/views"
)

// Every embedded template must render with no placeholder left over and no
// link to a hosted instance: the URLs come from the deployment's config.
func TestTemplatesUseConfiguredValues(t *testing.T) {
	prev := config.AppConfig
	t.Cleanup(func() { config.AppConfig = prev })
	config.AppConfig = &views.Config{
		ProductName:  "Acme Mail",
		FrontendURL:  "https://acme.example",
		DashboardURL: "https://app.acme.example",
		SupportEmail: "help@acme.example",
		EmailLogoURL: "https://acme.example/logo.svg",
	}

	vars := map[string]string{"name": "Ada", "domain": "ada.example", "verify_link": "x", "reset_link": "y"}
	for _, name := range []string{"welcome_email.html", "verify_email.html", "reset_password.html", "domain.html"} {
		body := RenderTemplate(name, vars)
		if body == "" {
			t.Fatalf("%s rendered empty", name)
		}
		if strings.Contains(body, "{{") {
			i := strings.Index(body, "{{")
			t.Errorf("%s has an unreplaced placeholder near %q", name, body[i:min(i+40, len(body))])
		}
		if strings.Contains(body, "vnytros.dev") {
			t.Errorf("%s still links to vnytros.dev", name)
		}
		if !strings.Contains(body, "Acme Mail") || !strings.Contains(body, "https://acme.example/logo.svg") {
			t.Errorf("%s does not carry the configured product name and logo", name)
		}
	}
}
