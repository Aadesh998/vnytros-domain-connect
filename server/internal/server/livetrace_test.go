package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"domain-connect-backend/internal/telemetry"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/trace"
)

// Sends a real api request's span to a live collector and prints the trace id.
func TestLiveApiTrace(t *testing.T) {
	// Integration test, skipped by default. To watch a real span land in Tempo:
	//
	//   cd deploy && docker compose -f docker-compose.infra.yaml up -d otel-collector tempo
	//   LIVE_OTEL=1 go test ./internal/server/ -run TestLiveApiTrace -v
	//   curl -s localhost:3200/api/traces/<the printed TRACE_ID> | jq
	//
	// TestServerSpanHasNameAndRoute covers the same span shape offline, so this
	// exists for confirming the export path itself works.
	if os.Getenv("LIVE_OTEL") == "" {
		t.Skip("needs the collector running; see the comment above")
	}
	t.Setenv("OTEL_ENDPOINT", "localhost:4318")
	t.Setenv("ENV", "production")
	shutdown := telemetry.Init(context.Background(), "vnytros-server")

	r := chi.NewRouter()
	r.Use(telemetry.HTTPMiddlewares()...)
	var traceID string
	r.Get("/v1/plans/{id}", func(w http.ResponseWriter, req *http.Request) {
		traceID = trace.SpanContextFromContext(req.Context()).TraceID().String()
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(r)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/v1/plans/42")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	shutdown() // flush the batch
	fmt.Println("TRACE_ID=" + traceID)
}
