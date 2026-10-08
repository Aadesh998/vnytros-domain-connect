package telemetry

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// HTTPMiddlewares returns the instrumentation chain for a chi router, in the
// order it must be applied:
//
//	r.Use(telemetry.HTTPMiddlewares()...)
//
// The order is not cosmetic, and it was arrived at by measuring what the span
// actually came out looking like (internal/server/apitrace_test.go):
//
//  1. HTTPMiddleware — Prometheus counters/histograms. Outermost so it times
//     the whole chain, including the tracing overhead.
//  2. serverSpan — otelhttp, which opens the server span. Everything inside it
//     is therefore a child.
//  3. routeAttribute — INNERMOST, and it has to be. chi only fills in
//     RoutePattern() while routing, which happens after the middleware chain,
//     so http.route can only be read on the way back out. Being inside the
//     otelhttp middleware is what keeps the span still open at that point;
//     from anywhere further out the span has already ended and setting the
//     attribute silently does nothing.
func HTTPMiddlewares() []func(http.Handler) http.Handler {
	return []func(http.Handler) http.Handler{
		HTTPMiddleware,
		serverSpan(),
		routeAttribute,
	}
}

// serverSpan opens one span per request, named "METHOD /route/{pattern}".
func serverSpan() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return otelhttp.NewHandler(next, "http.server",
			otelhttp.WithSpanNameFormatter(func(_ string, r *http.Request) string {
				if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
					return r.Method + " " + rc.RoutePattern()
				}
				return r.Method
			}),
		)
	}
}

// routeAttribute records http.route on the open server span.
//
// otelhttp cannot set this itself — it has no idea what a chi pattern is — and
// without it the collector's spanmetrics connector has nothing to put in its
// http.route dimension, so the span-derived RED metrics lose their per-endpoint
// breakdown.
func routeAttribute(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)

		rc := chi.RouteContext(r.Context())
		if rc == nil || rc.RoutePattern() == "" {
			return
		}
		trace.SpanFromContext(r.Context()).SetAttributes(
			attribute.String("http.route", rc.RoutePattern()),
		)
	})
}
