package telemetry_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"domain-connect-backend/internal/telemetry"

	"github.com/go-chi/chi/v5"
)

// Guards two things that are easy to regress: that /metrics actually serves,
// and that HTTP metrics are labelled by chi ROUTE PATTERN rather than raw path.
// The second matters more than it looks — labelling by path would create one
// time series per item id and eventually fall over.
func TestMetricsEndToEnd(t *testing.T) {
	t.Setenv("OTEL_ENDPOINT", "off") // no collector in a test
	shutdown := telemetry.Init(context.Background(), "probe")
	defer shutdown()

	r := chi.NewRouter()
	r.Use(telemetry.HTTPMiddleware)
	r.Get("/v1/items/{id}", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	// two different ids must collapse into ONE series
	for _, id := range []string{"1", "2", "999"} {
		resp, err := http.Get(srv.URL + "/v1/items/" + id)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	telemetry.JobsTotal.WithLabelValues("campaign.send", "ack").Inc()
	telemetry.JobDuration.WithLabelValues("campaign.send").Observe(1.5)

	stop := telemetry.Serve("19199")
	defer stop()

	var body string
	for i := 0; i < 20; i++ {
		resp, err := http.Get("http://127.0.0.1:19199/metrics")
		if err == nil {
			b, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			body = string(b)
			break
		}
	}
	if body == "" {
		t.Fatal("metrics endpoint never came up")
	}

	for _, want := range []string{
		`vnytros_http_requests_total{method="GET",route="/v1/items/{id}",status="418"} 3`,
		`vnytros_jobs_total{outcome="ack",queue="campaign.send"} 1`,
		"vnytros_http_request_duration_seconds_bucket",
		"vnytros_job_duration_seconds_sum",
		"go_goroutines",    // free runtime metrics
		"process_open_fds", // free process metrics
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing from /metrics: %s", want)
		}
	}
	if strings.Contains(body, "/v1/items/1") {
		t.Error("raw path leaked into a label — cardinality bug")
	}
}
