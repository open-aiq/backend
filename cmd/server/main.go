package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"go-aiq-backend/docs"
	"go-aiq-backend/internal/airquality"
	"go-aiq-backend/internal/device"
	"go-aiq-backend/internal/platform/config"
	"go-aiq-backend/internal/platform/database"
	"go-aiq-backend/internal/platform/middleware"
)

// version is the build version, injected at release time via
// -ldflags "-X main.version=<tag>". Defaults to "dev" for local builds.
var version = "dev"

// @title Air Quality API
// @version 1.0
// @description Air quality monitoring API
// @host localhost:8080
// @BasePath /api/v1
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Connect to Postgres. Schema changes are applied out-of-band via versioned
	// migrations (cmd/migrate / `make migrate-up`), never on startup.
	client, err := database.New(cfg)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer client.Close()

	// Wire up the air quality domain.
	// TODO: swap MockRepository for an Ent-backed repository once implemented.
	service := airquality.NewService(airquality.NewMockRepository())
	handler := airquality.NewHandler(service)

	// Wire up the device domain (Ent-backed).
	deviceHandler := device.NewHandler(device.NewService(device.NewEntRepository(client)))

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
	r := gin.Default()
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)
	deviceHandler.RegisterRoutes(api)

	srv := &http.Server{
		Addr:    cfg.Addr(),
		Handler: r,
	}

	// Run the server in a goroutine so we can listen for shutdown signals.
	go func() {
		log.Printf("listening on %s (env=%s, version=%s)", cfg.Addr(), cfg.Env, version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("forced shutdown: %v", err)
	}
	log.Println("server stopped")
}
