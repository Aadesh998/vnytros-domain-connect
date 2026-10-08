package server

import (
	"context"
	"database/sql"
	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/db"
	"domain-connect-backend/internal/handler"
	mailforge "domain-connect-backend/internal/mailforge/server"
	"domain-connect-backend/internal/queue"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/telemetry"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	appmiddleware "domain-connect-backend/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func StartServer() {
	log.Printf("INFO: Starting server initialization")

	config.LoadConfig()
	config.RequireProduction()
	config.InitLogger()

	// Tracing and metrics for this whole process, mailforge included. Init
	// must come first: mailforge's otelgin middleware resolves the global
	// tracer provider when its router is built.
	shutdownTracing := telemetry.Init(context.Background(), "vnytros-server")
	defer shutdownTracing()
	shutdownMetrics := telemetry.Serve("9101")
	defer shutdownMetrics()

	log.Printf("INFO: Initializing database")
	db.InitDB()
	sqlDb, err := db.DB.DB()
	if err != nil {
		log.Printf("FATAL: Failed to connect to database: %v", err)
		log.Fatal(err)
	}

	defer func(sqlDb *sql.DB) {
		log.Printf("INFO: Closing database connection")
		err := sqlDb.Close()
		if err != nil {
			log.Printf("ERROR: while Closing DB connection: %s", err)
		}
	}(sqlDb)

	var Queue = []string{
		queue.QueueEmailSend,
		queue.QueueVerifyDomain,
		queue.QueueWebhookDispatch,
		queue.QueueCampaignSend,
		queue.QueueCampaignTrack,
	}

	log.Printf("INFO: Connecting to RabbitMQ")
	if url := config.AppConfig.RabitMQ; url != "" {
		qc, err := queue.NewClient(url)
		if err != nil {
			log.Printf("WARN: RabbitMQ unavailable, emails will send synchronously: %v", err)
		} else {
			defer qc.Close()
			for _, qn := range Queue {
				if err := qc.DeclareQueue(qn); err != nil {
					log.Printf("WARN: declare email queue: %v", err)
				}
			}

			queue.SetPublisher(qc)
			log.Printf("INFO: mail publisher wired to RabbitMQ")
		}
	} else {
		log.Printf("WARN: RABITMQ not set, emails will send synchronously")
	}

	log.Printf("INFO: Creating server")

	providerRepo := repository.NewProviderRepository(db.DB)
	domainRepo := repository.NewDomainRepository(db.DB)
	userRepo := repository.NewUserRepository(db.DB)
	apiKeyRepo := repository.NewApiKeyRepository(db.DB)
	webhookLogRepo := repository.NewWebhookEventLogRepository(db.DB)
	oauthRepo := repository.NewOAuthRepository(db.DB)

	domainService := service.NewDomainService(providerRepo, domainRepo, userRepo, apiKeyRepo, webhookLogRepo)
	domainHandler := handler.NewDomainHandler(domainService)

	authService := service.NewAuthService(userRepo, apiKeyRepo, domainRepo)
	oauthService := service.NewOAuthService(oauthRepo, userRepo, authService)
	oauthHandler := handler.NewOAuthHandler(oauthService)
	authHandler := handler.NewAuthHandler(authService, oauthService)

	webhookLogService := service.NewWebhookLogService(webhookLogRepo)
	webhookLogHandler := handler.NewWebhookLogHandler(webhookLogService)

	netcheckService := service.NewNetcheckService()
	toolsHandler := handler.NewToolsHandler(netcheckService)

	log.Printf("INFO: Initializing mailforge")
	mailforgeShutdown := mailforge.Init(context.Background(), db.DB)
	defer mailforgeShutdown()
	mailforgeHandler := mailforge.NewHandler()

	router := setupRoutes(domainHandler, authHandler, webhookLogHandler, oauthHandler, toolsHandler, mailforgeHandler)
	server := createServer(router)
	log.Printf("INFO: Running server on %s", server.Addr)
	if err := runServer(context.Background(), server, 10*time.Second); err != nil {
		log.Printf("FATAL: Failed to Start Server %s", err)
		log.Fatalf("Failed to Start Server %s: ", err)
	}
}

func setupRoutes(
	domainHandler *handler.DomainHandler,
	authHandler *handler.AuthHandler,
	webhookLogHandler *handler.WebhookLogHandler,
	oauthHandler *handler.OAuthHandler,
	toolsHandler *handler.ToolsHandler,
	mailforgeHandler http.Handler,
) *chi.Mux {
	apiKeyAuth := appmiddleware.APIKeyAuth()
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(appmiddleware.CORSMiddleware)
	// Prometheus metrics + the server span + http.route, in the one order that
	// produces a correctly named span AND the route attribute. See
	// telemetry.HTTPMiddlewares.
	r.Use(telemetry.HTTPMiddlewares()...)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Domain Connect API is running"))
	})

	r.Handle("/api", mailforgeHandler)
	r.Handle("/api/*", mailforgeHandler)

	r.Get("/v1/detect", domainHandler.DetectProvider)
	r.Get("/v1/dns/lookup", domainHandler.DNSLookup)
	r.Post("/v1/connect/direct", apiKeyAuth(domainHandler.DirectProviderConnect))
	r.Post("/v1/status", domainHandler.GetDomainStatus)
	r.Post("/v1/connect/verify", apiKeyAuth(domainHandler.VerifyDomain))
	r.Get("/v1/connect/domains", apiKeyAuth(domainHandler.ListUserDomains))
	r.Get("/v1/connect/domain", apiKeyAuth(domainHandler.GetUserDomain))
	r.Get("/_domainconnect/_dckeypubv1.public", domainHandler.GetPublicKey)
	// The template route is served for the configured provider domain only;
	// see DC_PROVIDER_DOMAIN in .env.example.
	if dc := config.AppConfig.DCProviderDomain; dc != "" {
		r.Get("/.well-known/domainconnect/"+dc+"/custom-domain", domainHandler.GetTemplate)
	}
	r.Get("/.well-known/domainconnect", domainHandler.HandleDCDiscovery)

	r.Post("/v1/auth/signup", authHandler.Signup)
	r.Post("/v1/auth/login", authHandler.Login)
	r.Post("/v1/auth/logout", authHandler.Logout)
	r.Get("/v1/auth/verify", authHandler.Verify)
	r.Post("/v1/auth/refresh", authHandler.Refresh)
	r.Post("/v1/auth/forget-password", authHandler.ForgetPassword)
	r.Post("/v1/auth/reset-password", authHandler.ResetPassword)
	r.Get("/v1/auth/google/login", authHandler.GoogleLogin)
	r.Get("/v1/auth/google/callback", authHandler.GoogleCallback)
	r.Get("/v1/auth/github/login", authHandler.GithubLogin)
	r.Get("/v1/auth/github/callback", authHandler.GithubCallback)

	r.Get("/.well-known/oauth-authorization-server", oauthHandler.Metadata)
	r.Post("/oauth/register", oauthHandler.Register)
	r.Get("/oauth/authorize", oauthHandler.Authorize)
	r.Post("/oauth/login", oauthHandler.LoginSubmit)
	r.Get("/oauth/authorize/google", oauthHandler.GoogleStart)
	r.Get("/oauth/google/callback", oauthHandler.GoogleCallback)
	r.Get("/oauth/consent", oauthHandler.Consent)
	r.Post("/oauth/decision", oauthHandler.Decision)
	r.Post("/oauth/token", oauthHandler.Token)

	r.Post("/v1/create/api", appmiddleware.AuthorizeRoles("user", "admin")(authHandler.CreateAPIKey))
	r.Get("/v1/get/apis", appmiddleware.AuthorizeRoles("user", "admin")(authHandler.GetAllAPIKey))
	r.Patch("/v1/update/api", appmiddleware.AuthorizeRoles("user", "admin")(authHandler.UpdateAPIKey))
	r.Delete("/v1/delete/api", appmiddleware.AuthorizeRoles("user", "admin")(authHandler.DeleteAPIKey))

	r.Get("/v1/get/webhooks", appmiddleware.AuthorizeRoles("user", "admin")(webhookLogHandler.GetAll))
	r.Get("/v1/filter/webhooks", appmiddleware.AuthorizeRoles("user", "admin")(webhookLogHandler.GetFiltered))

	r.Get("/v1/me", appmiddleware.AuthorizeRoles("user", "admin")(authHandler.Me))

	userOrAdminAuth := appmiddleware.AuthorizeRoles("user", "admin")
	r.Get("/v1/dashboard/domains", userOrAdminAuth(domainHandler.ListUserDomains))
	r.Get("/v1/dashboard/domain", userOrAdminAuth(domainHandler.GetUserDomain))
	r.Post("/v1/dashboard/domain/verify", userOrAdminAuth(domainHandler.VerifyDomain))

	r.Post("/v1/vnytros/webhook", handler.ReceiveWebhook)

	r.Group(func(tools chi.Router) {
		tools.Use(appmiddleware.RateLimit(60, 15))

		tools.Get("/v1/tools", toolsHandler.Catalogue)

		tools.Get("/v1/tools/dns/propagation", toolsHandler.DNSPropagation)
		tools.Get("/v1/tools/dns/nameservers", toolsHandler.Nameservers)
		tools.Get("/v1/tools/dns/cname", toolsHandler.CNAMEChain)
		tools.Get("/v1/tools/dns/caa", toolsHandler.CAA)
		tools.Get("/v1/tools/dns/soa", toolsHandler.SOA)
		tools.Get("/v1/tools/dns/dnssec", toolsHandler.DNSSEC)

		tools.Get("/v1/tools/email/auth", toolsHandler.EmailAuth)
		tools.Get("/v1/tools/email/spf", toolsHandler.SPF)
		tools.Get("/v1/tools/email/dkim", toolsHandler.DKIM)
		tools.Get("/v1/tools/email/dmarc", toolsHandler.DMARC)

		tools.Get("/v1/tools/domain/info", toolsHandler.DomainInfo)
		tools.Get("/v1/tools/tls/certificate", toolsHandler.TLSCertificate)
		tools.Get("/v1/tools/http/headers", toolsHandler.SecurityHeaders)
		tools.Get("/v1/tools/http/redirects", toolsHandler.RedirectChain)
		tools.Get("/v1/tools/ip/info", toolsHandler.IPInfo)
	})

	return r
}

func createServer(handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:         ":8000",
		Handler:      handler,
		WriteTimeout: time.Hour * 2,
		ReadTimeout:  time.Hour * 2,
	}

	return server
}

func runServer(
	ctx context.Context,
	server *http.Server,
	shutdownTimeout time.Duration,
) error {
	errCh := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); !errors.Is(
			err, http.ErrServerClosed,
		) {
			errCh <- err
		}
		close(errCh)
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case <-stop:
		log.Printf("Received Server Shutdown Signal.")
	case <-ctx.Done():
		log.Printf("Context Time Limit Exceed. Server ShutDown.")
	}

	shutDownCtx, cancel := context.WithTimeout(
		context.Background(),
		shutdownTimeout,
	)
	defer cancel()

	if err := server.Shutdown(shutDownCtx); err != nil {
		if closeErr := server.Close(); closeErr != nil {
			return errors.Join(err, closeErr)
		}
		return err
	}

	log.Println("Server Stopped Gracefully.")
	return nil
}
