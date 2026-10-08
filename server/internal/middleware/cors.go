package middleware

import (
	"domain-connect-backend/internal/config"
	"net/http"
	"net/url"
	"strings"
)

// isAllowedOrigin reports whether origin may make credentialed requests. The
// list comes from ALLOWED_ORIGINS (shared with mailforge) plus FRONTEND_URL and
// DASHBOARD_URL. An entry is either an exact origin ("https://app.example.com")
// or a subdomain wildcard ("*.example.com", which also matches example.com).
func isAllowedOrigin(origin string) bool {
	if origin == "" || config.AppConfig == nil {
		return false
	}
	cfg := config.AppConfig
	allowed := append([]string{cfg.FrontendURL, cfg.DashboardURL}, cfg.AllowedOrigins...)
	return originMatches(origin, allowed)
}

func originMatches(origin string, allowed []string) bool {
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	host := u.Hostname()
	for _, a := range allowed {
		a = strings.TrimRight(strings.TrimSpace(a), "/")
		if a == "" {
			continue
		}
		if suffix, ok := strings.CutPrefix(a, "*."); ok {
			if host == suffix || strings.HasSuffix(host, "."+suffix) {
				return true
			}
			continue
		}
		if strings.EqualFold(a, origin) {
			return true
		}
	}
	return false
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// /api/* belongs to the mounted mailforge router, which runs its own
		// CORS against the explicit ALLOWED_ORIGINS list (localhost dev ports
		// included). Handling it here too would answer its preflights with the
		// api's policy (exact origins plus "*." wildcards) and could diverge.
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/api" {
			next.ServeHTTP(w, r)
			return
		}

		origin := r.Header.Get("Origin")
		if isAllowedOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
