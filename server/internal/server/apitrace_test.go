package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"domain-connect-backend/internal/telemetry"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Guards the shape of the api's server span. Both assertions have failed in
// practice: the route pattern is only known after routing, so a middleware
// ordered wrongly yields a span named just "GET" (every endpoint collapsing
// into one) or loses http.route (which the collector's spanmetrics connector
// needs for its per-route dimension).
func TestServerSpanHasNameAndRoute(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))

	r := chi.NewRouter()
	r.Use(telemetry.HTTPMiddlewares()...)
	r.Get("/v1/plans/{id}", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/v1/plans/42")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	spans := rec.Ended()
	if len(spans) == 0 {
		t.Fatal("no spans produced — the api is not traced at all")
	}

	sp := spans[0]
	if want := "GET /v1/plans/{id}"; sp.Name() != want {
		t.Errorf("span name = %q, want %q", sp.Name(), want)
	}

	var route string
	for _, a := range sp.Attributes() {
		if string(a.Key) == "http.route" {
			route = a.Value.Emit()
		}
	}
	if want := "/v1/plans/{id}"; route != want {
		t.Errorf("http.route = %q, want %q", route, want)
	}

	// The id must not leak into either, or Tempo and the derived metrics get
	// one series per plan.
	if sp.Name() == "GET /v1/plans/42" || route == "/v1/plans/42" {
		t.Error("raw path leaked into the span — cardinality bug")
	}
}
