package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"go-aiq-backend/docs"
	"go-aiq-backend/internal/airquality"
	"go-aiq-backend/internal/device"
	"go-aiq-backend/internal/devicereading"
	"go-aiq-backend/internal/platform/config"
	"go-aiq-backend/internal/platform/database"
	"go-aiq-backend/internal/platform/middleware"
	"go-aiq-backend/internal/platform/problem"
)

// version is the build version, injected at release time via
// -ldflags "-X main.version=<tag>". Defaults to "dev" for local builds.
var version = "dev"

// @title Air Quality API
// @version 1.0
// @description Air quality monitoring API
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Clerk session token using the Bearer scheme. Example: "Bearer {token}"
func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}
	clerk.SetKey(cfg.ClerkSecretKey)

	// Connect to Postgres. Schema changes are applied out-of-band via versioned
	// migrations (cmd/migrate / `make migrate-up`), never on startup.
	client, err := database.New(cfg)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer client.Close()

	// Wire up the air quality domain (aggregated reads over device readings).
	handler := airquality.NewHandler(airquality.NewService(airquality.NewEntRepository(client)))

	// Wire up the device domain. Its service doubles as the authenticator for
	// device data uploads.
	deviceService := device.NewService(device.NewEntRepository(client))
	deviceHandler := device.NewHandler(deviceService)

	// Wire up the device reading domain (data ingestion).
	readingHandler := devicereading.NewHandler(
		devicereading.NewService(devicereading.NewEntRepository(client)),
		deviceService,
	)

	// Serve Swagger with a relative host so "Try it out" targets whatever
	// origin served the docs (localhost in dev, the real domain in prod, or
	// behind a reverse proxy) instead of the baked-in @host. Leaving Host and
	// Schemes empty makes Swagger UI fall back to the browser's location.
	docs.SwaggerInfo.Host = ""
	docs.SwaggerInfo.Schemes = nil

	// HTTP router.
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(middleware.RequestLogger(logger), gin.CustomRecovery(func(c *gin.Context, recovered any) {
		logger.Error("panic recovered", "error", recovered)
		if !c.Writer.Written() {
			problem.Write(c, problem.Internal, "The server could not complete the request.")
		}
	}))
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.NoRoute(func(c *gin.Context) { problem.Write(c, problem.NotFound, "Route not found.") })
	r.NoMethod(func(c *gin.Context) { problem.Write(c, problem.MethodNotAllowed, "Method not allowed.") })

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	authenticated := api.Group("")
	authenticated.Use(middleware.ClerkAuth(cfg.ClerkSecretKey, cfg.ClerkAuthorizedParties))
	handler.RegisterRoutes(authenticated)
	deviceHandler.RegisterRoutes(authenticated)
	deviceHandler.RegisterPublicRoutes(api)
	handler.RegisterPublicRoutes(api)
	readingHandler.RegisterRoutes(api)

	srv := &http.Server{
		Addr:    cfg.Addr(),
		Handler: r,
	}

	// Run the server in a goroutine so we can listen for shutdown signals.
	go func() {
		logger.Info("server listening", "address", cfg.Addr(), "environment", cfg.Env, "version", version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server stopped unexpectedly", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("forced shutdown", "error", err)
		os.Exit(1)
	}
	logger.Info("server stopped")
}
