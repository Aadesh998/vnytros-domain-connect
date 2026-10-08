// Package telemetry is the one place this module configures tracing and
// metrics. Every binary calls Init once at startup and Serve to expose
// /metrics; everything else uses the global tracer provider and the metric
// vectors declared in metrics.go.
package telemetry

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// Tracer returns a named tracer from the global provider. Safe to call before
// Init — it just yields no-op spans until Init installs a real provider.
func Tracer(name string) trace.Tracer { return otel.Tracer(name) }

// Init installs a global tracer provider that exports over OTLP/HTTP to the
// collector, and returns a shutdown func that flushes pending spans.
//
// serviceName becomes service.name on every span, which is what separates the
// three binaries in Tempo and Grafana. It used to be hardcoded "go-backend" for
// everything.
//
// Config:
//
//	OTEL_ENDPOINT            host:port of the collector (default localhost:4318)
//	                         set to "off" to disable tracing entirely
//	OTEL_TRACES_SAMPLE_RATIO 0.0-1.0, default 1.0 (sample everything)
//	ENV                      recorded as deployment.environment
//
// Unlike the previous implementation this never calls log.Fatal: a missing or
// unreachable collector must not stop the service from booting. Tracing is
// diagnostics, not a dependency.
func Init(ctx context.Context, serviceName string) func() {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_ENDPOINT"))
	if endpoint == "" {
		endpoint = "localhost:4318"
	}
	if strings.EqualFold(endpoint, "off") {
		log.Printf("telemetry: OTEL_ENDPOINT=off, tracing disabled for %s", serviceName)
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func() {}
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		log.Printf("telemetry: WARN tracing disabled, exporter setup failed: %v", err)
		otel.SetTracerProvider(noop.NewTracerProvider())
		return func() {}
	}

	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}

	res, err := resource.Merge(resource.Default(), resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
		attribute.String("deployment.environment", env),
	))
	if err != nil {
		// Schema mismatch between resource.Default() and ours; fall back to
		// ours rather than losing the service name.
		res = resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			attribute.String("deployment.environment", env),
		)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRatio()))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{},
	))

	log.Printf("telemetry: tracing %s -> %s", serviceName, endpoint)

	return func() {
		// Fresh context: ctx is usually already cancelled by the time shutdown
		// runs, and a cancelled context would drop the final batch of spans.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), tracerShutdownTimeout)
		defer cancel()
		if err := tp.Shutdown(shutdownCtx); err != nil {
			log.Printf("telemetry: WARN tracer shutdown: %v", err)
		}
	}
}

func sampleRatio() float64 {
	raw := os.Getenv("OTEL_TRACES_SAMPLE_RATIO")
	if raw == "" {
		return 1.0
	}
	ratio, err := strconv.ParseFloat(raw, 64)
	if err != nil || ratio < 0 || ratio > 1 {
		log.Printf("telemetry: WARN bad OTEL_TRACES_SAMPLE_RATIO %q, using 1.0", raw)
		return 1.0
	}
	return ratio
}
