package main

import (
	"context"
	"time"

	"domain-connect-backend/internal/config"
	"domain-connect-backend/internal/db"
	mcptools "domain-connect-backend/internal/mcp"
	"domain-connect-backend/internal/middleware"
	"domain-connect-backend/internal/repository"
	"domain-connect-backend/internal/service"
	"domain-connect-backend/internal/telemetry"
	"log"
	"net/http"
	"strings"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

func main() {
	config.LoadConfig()
	config.InitLogger()

	shutdownTracing := telemetry.Init(context.Background(), "vnytros-mcp")
	defer shutdownTracing()
	shutdownMetrics := telemetry.Serve("9102")
	defer shutdownMetrics()

	db.InitDB()
	sqlDb, err := db.DB.DB()
	if err != nil {
		log.Fatalf("FATAL: Failed to connect to database: %v", err)
	}
	defer sqlDb.Close()

	providerRepo := repository.NewProviderRepository(db.DB)
	domainRepo := repository.NewDomainRepository(db.DB)
	userRepo := repository.NewUserRepository(db.DB)
	apiKeyRepo := repository.NewApiKeyRepository(db.DB)
	webhookLogRepo := repository.NewWebhookEventLogRepository(db.DB)

	domainService := service.NewDomainService(providerRepo, domainRepo, userRepo, apiKeyRepo, webhookLogRepo)

	const (
		serverName    = "vnytros-domainConnect-mcp"
		serverVersion = "1.0.0"
	)

	s := mcp.NewServer(&mcp.Implementation{
		Name:       serverName,
		Version:    serverVersion,
		WebsiteURL: config.AppConfig.FrontendURL,
	}, nil)

	mcptools.RegisterTools(s, domainService)
	mcptools.RegisterNetTools(s, service.NewNetcheckService())

	mcpHandler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return s },
		&mcp.StreamableHTTPOptions{
			Stateless:                  true,
			DisableLocalhostProtection: true,
		},
	)

	baseURL := strings.TrimRight(config.AppConfig.MCPBaseURL, "/")
	resource := baseURL + "/mcp"
	metaPath := "/.well-known/oauth-protected-resource"

	authServer := strings.TrimRight(config.AppConfig.OAuthIssuerURL, "/")
	protected := mcpsdk.RequireBearerToken(
		middleware.MCPTokenVerifier,
		&mcpsdk.RequireBearerTokenOptions{
			ResourceMetadataURL: baseURL + metaPath,
		},
	)(mcpHandler)

	mux := http.NewServeMux()
	mux.Handle("/mcp", protected)
	mux.Handle(metaPath, mcpsdk.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               resource,
		AuthorizationServers:   []string{authServer},
		BearerMethodsSupported: []string{"header"},
	}))
	mux.Handle("/", mcptools.LandingHandler(mcptools.LandingInfo{
		ServerName:    serverName,
		ServerVersion: serverVersion,
		WebsiteURL:    config.AppConfig.FrontendURL,
		Resource:      resource,
		AuthServer:    authServer,
		MetadataURL:   baseURL + metaPath,
	}))

	// One server span per request. Tool calls are the slow thing here, so
	// this is where a stuck MCP client becomes visible.
	handler := otelhttp.NewHandler(mux, "mcp.server")

	log.Println("MCP server starting on :5000 (stateless, /mcp)")
	srv := &http.Server{
		Addr:              ":5000",
		Handler:           handler,
		ReadHeaderTimeout: 15 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
