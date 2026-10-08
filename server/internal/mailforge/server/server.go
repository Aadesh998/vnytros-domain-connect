package server

import (
	"context"
	"log"
	"net/http"

	"domain-connect-backend/internal/mailforge/config"
	mailforgedb "domain-connect-backend/internal/mailforge/db"
	"domain-connect-backend/internal/mailforge/middleware"
	"domain-connect-backend/internal/mailforge/queue"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/gorm"
)

// Init prepares the mailforge half of the api process: its own config (BASE_URL,
// send limits, watermark defaults) and its RabbitMQ publisher.
//
// It no longer sets up a tracer. cmd/server calls internal/telemetry.Init for
// the whole process, and mailforge's otelgin middleware picks that up from the
// global provider — two providers in one process would mean spans landing on
// whichever registered last.
//
// It deliberately does NOT open a database of its own. mailforge used to run as
// a separate service with a separate pool against the very same Postgres; now
// that both halves share a process it reuses the api's *gorm.DB, so there is one
// pool and one place that owns connection tuning.
//
// The returned func releases what Init acquired and is safe to call once.
func Init(_ context.Context, db *gorm.DB) func() {
	config.LoadConfig()
	mailforgedb.DB = db

	if err := queue.Connect(); err != nil {
		log.Printf("WARN: mailforge queue unavailable at startup: %v", err)
	}

	return func() {
		queue.Close()
	}
}

// NewHandler builds the mailforge router. Every route it owns lives under /api,
// which the api's own router leaves free (it serves /v1, /oauth and
// /.well-known), so the two can be mounted side by side without rewriting a
// single path or handler.
func NewHandler() *gin.Engine {
	if config.AppConfig.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(
		gin.Recovery(),
		otelgin.Middleware("vnytros-mail"),
		middleware.IPLoggingMiddleware(),
		middleware.ErrorHandlingMiddleware(),
		// mailforge keeps its own CORS against the explicit ALLOWED_ORIGINS
		// list. The api's CORS middleware steps aside for /api/* so this one
		// stays authoritative for these routes.
		middleware.CORSMiddleware(),
		middleware.RateLimiterMiddleware(),
	)

	publicRoutes(r)

	api := r.Group("/api", middleware.AuthMiddleware())
	routesTemplate(api.Group("/template"))
	routesCampaign(api.Group("/campaign"))
	routesSettings(api.Group("/settings"))
	routesAnalytics(api.Group("/analytics"))

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    "NOT_FOUND",
			"error":   "API route not found",
			"message": "API route not found",
		})
	})

	return r
}
