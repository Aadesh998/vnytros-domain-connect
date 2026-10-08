package db_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/db"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// Verifies the real InitDB path: that queries on the shared pool produce spans,
// that they attach to the caller's span, and that bound values never reach the
// db.statement attribute.
func TestGormTracing(t *testing.T) {
	// Integration test: needs a real Postgres, so it is skipped by default.
	//
	//   docker run -d --rm --name pgtrace -p 127.0.0.1:55432:5432 \
	//     -e POSTGRES_PASSWORD=testpw -e POSTGRES_DB=tracetest postgres:16-alpine
	//   PG_TRACE_DSN_PORT=55432 go test ./internal/db/ -run TestGormTracing -v
	//
	// Worth keeping despite that: it guards the WithoutQueryVariables option,
	// without which every bound parameter — SMTP passwords, API keys, password
	// hashes — would be written into db.statement and shipped to Tempo.
	if os.Getenv("PG_TRACE_DSN_PORT") == "" {
		t.Skip("needs a throwaway postgres; see the comment above")
	}
	t.Setenv("DB_HOST", "127.0.0.1")
	t.Setenv("DB_PORT", os.Getenv("PG_TRACE_DSN_PORT"))
	t.Setenv("DB_USER", "postgres")
	t.Setenv("DB_PASSWORD", "testpw")
	t.Setenv("DB_NAME", "tracetest")

	rec := tracetest.NewSpanRecorder()
	otel.SetTracerProvider(sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec)))

	config.LoadConfig()
	db.InitDB()

	type Secret struct {
		ID       uint `gorm:"primaryKey"`
		Password string
	}
	if err := db.DB.AutoMigrate(&Secret{}); err != nil {
		t.Fatal(err)
	}

	// A parent span, exactly as an HTTP handler or worker job would have.
	ctx, parent := otel.Tracer("probe").Start(context.Background(), "parent-operation")
	const secret = "super-secret-smtp-password"
	if err := db.DB.WithContext(ctx).Create(&Secret{Password: secret}).Error; err != nil {
		t.Fatal(err)
	}
	var got Secret
	if err := db.DB.WithContext(ctx).Where("password = ?", secret).First(&got).Error; err != nil {
		t.Fatal(err)
	}
	parent.End()

	// AutoMigrate above ran without a context, so its spans are roots in their
	// own traces — the same thing that happens to every api repository call
	// today. Only spans in the caller's trace are the ones under test here.
	var dbSpans, orphans int
	for _, sp := range rec.Ended() {
		if sp.Name() == "parent-operation" {
			continue
		}
		if sp.SpanContext().TraceID() != parent.SpanContext().TraceID() {
			orphans++
			continue
		}
		dbSpans++

		if sp.Parent().SpanID() != parent.SpanContext().SpanID() {
			t.Errorf("span %q is in the trace but not a child of the caller's span", sp.Name())
		}
		for _, a := range sp.Attributes() {
			if strings.Contains(a.Value.Emit(), secret) {
				t.Errorf("SECRET LEAKED into attribute %s: %s", a.Key, a.Value.Emit())
			}
			if a.Key == "db.statement" {
				t.Logf("db.statement = %s", a.Value.Emit())
			}
		}
	}
	if dbSpans == 0 {
		t.Fatal("no database spans recorded — plugin not active")
	}
	t.Logf("%d context-carrying queries produced correctly parented spans, no values leaked", dbSpans)
	t.Logf("%d context-less queries (AutoMigrate) produced orphan root spans — expected", orphans)
}
