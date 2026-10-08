package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mfconfig "domain-connect-backend/internal/mailforge/config"
	mailforge "domain-connect-backend/internal/mailforge/server"
	appmiddleware "domain-connect-backend/internal/middleware"

	"github.com/go-chi/chi/v5"
)

// mountMailforge mirrors how setupRoutes wires the two routers together.
func mountMailforge(t *testing.T) *httptest.Server {
	t.Helper()
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:5173")
	mfconfig.LoadConfig()

	r := chi.NewRouter()
	r.Use(appmiddleware.CORSMiddleware)
	r.Get("/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("api"))
	})
	h := mailforge.NewHandler()
	r.Handle("/api", h)
	r.Handle("/api/*", h)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// mailforge registers its routes with the /api prefix baked in, so the mount
// has to preserve the incoming path. chi's Mount would strip it; Handle does
// not. This is the regression guard for that distinction.
func TestMailforgeMountPreservesPath(t *testing.T) {
	srv := mountMailforge(t)

	cases := []struct {
		path string
		code int
		want string
	}{
		{"/api/health", http.StatusOK, `"status":"ok"`},
		{"/v1/ping", http.StatusOK, "api"},
		{"/api/nope", http.StatusNotFound, "NOT_FOUND"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tc.code {
				t.Errorf("status = %d, want %d (body %s)", resp.StatusCode, tc.code, body)
			}
			if !strings.Contains(string(body), tc.want) {
				t.Errorf("body = %q, want substring %q", body, tc.want)
			}
		})
	}
}

// mailforge runs its own CORS against the explicit ALLOWED_ORIGINS list. The
// api middleware must step aside for /api/* rather than answering its
// preflights with its own policy.
func TestMailforgeOwnsCORSForItsRoutes(t *testing.T) {
	srv := mountMailforge(t)

	req, err := http.NewRequest(http.MethodOptions, srv.URL+"/api/campaign", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q, want http://localhost:5173", got)
	}
}
