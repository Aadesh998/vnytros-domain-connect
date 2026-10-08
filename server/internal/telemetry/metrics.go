package telemetry

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	tracerShutdownTimeout  = 5 * time.Second
	metricsShutdownTimeout = 5 * time.Second
)

// Metrics are registered on the default registry, which already carries the Go
// runtime (go_*) and process (process_*) collectors — so memory, goroutines, GC
// and fd counts come for free alongside these.
var (
	// HTTP server metrics. Labelled by chi route pattern rather than raw path:
	// "/v1/items/{id}" is one series, whereas the raw path would be one series
	// per item id and would eventually take Prometheus down.
	HTTPRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vnytros_http_requests_total",
		Help: "HTTP requests by route, method and status class.",
	}, []string{"route", "method", "status"})

	HTTPDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "vnytros_http_request_duration_seconds",
		Help:    "HTTP request latency by route and method.",
		Buckets: []float64{0.005, 0.025, 0.1, 0.25, 1, 5, 10},
	}, []string{"route", "method"})

	// Worker metrics. outcome is one of: ack, dlq, requeue, panic — the four
	// branches of the worker's dispatch switch, so a rise in "requeue" is
	// visible as retry churn rather than as silence.
	JobsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "vnytros_jobs_total",
		Help: "Queue jobs processed by queue and outcome.",
	}, []string{"queue", "outcome"})

	JobDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "vnytros_job_duration_seconds",
		Help:    "Queue job handler latency by queue.",
		Buckets: []float64{0.01, 0.1, 0.5, 1, 5, 15, 60, 300},
	}, []string{"queue"})
)

// Serve exposes /metrics (and /healthz) on its own listener and returns a
// shutdown func.
//
// This is deliberately a separate port from the application's: /metrics leaks
// route names, queue names and latency profiles, and mounting it on the public
// router would put that behind nothing but obscurity. It is never proxied by
// nginx, so it is only reachable from the box (and from containers on it).
//
// Bind address is METRICS_HOST + ":" + defaultPort. Host and port are split
// rather than taking one METRICS_ADDR because all three services read the SAME
// EnvironmentFile — a single full address there would make them fight over one
// port. So the shared file sets the host once and each binary keeps its own
// port (server 9101, mcp 9102, worker 9103).
//
//	METRICS_HOST  127.0.0.1 by default, which is right for local development.
//	              In production it must be 0.0.0.0, because Prometheus runs in
//	              a container and arrives via the docker gateway address — a
//	              listener bound to loopback refuses that connection. Ports
//	              9101-9103 must therefore stay closed in the security group;
//	              this is the same posture the api's own :8000 already relies on.
//	              Set to "off" to disable the endpoint entirely.
func Serve(defaultPort string) func() {
	host := os.Getenv("METRICS_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	if host == "off" {
		log.Printf("telemetry: METRICS_HOST=off, /metrics not served")
		return func() {}
	}
	addr := net.JoinHostPort(host, defaultPort)

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("telemetry: metrics on http://%s/metrics", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// A metrics port that fails to bind must not take the service down.
			log.Printf("telemetry: WARN metrics server stopped: %v", err)
		}
	}()

	return func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), metricsShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("telemetry: WARN metrics shutdown: %v", err)
		}
	}
}

// statusRecorder captures the status code, which net/http does not expose after
// the fact.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	return r.ResponseWriter.Write(b)
}

// HTTPMiddleware records request count and latency. Mount it on a chi router;
// the route pattern is only known after routing, so the label is read once the
// inner handler has returned.
func HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(rec, r)

		route := "unmatched"
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			route = rc.RoutePattern()
		}
		if rec.status == 0 {
			rec.status = http.StatusOK
		}

		HTTPRequests.WithLabelValues(route, r.Method, strconv.Itoa(rec.status)).Inc()
		HTTPDuration.WithLabelValues(route, r.Method).Observe(time.Since(start).Seconds())
	})
}
