package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/config"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/database"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/handlers"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/middleware"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/repositories"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/scheduler"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/services"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/pkg/logger"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	appLogger := logger.New(cfg.Logger.Level, cfg.Logger.Format)

	// Connect to database
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		appLogger.WithError(err).Fatal("Failed to connect to database")
	}
	defer db.Close()

	// Initialize repositories
	bootstrapRepo := repositories.NewBootstrapRepository(db.DB)
	statusRepo := repositories.NewStatusRepository(db.DB)
	grpcRepo := repositories.NewGRPCRepository(db.DB)
	grpcStatusRepo := repositories.NewGRPCStatusRepository(db.DB)

	// Initialize services
	nodeChecker := services.NewNodeChecker(
		cfg.Monitor.ConnectionTimeout,
		cfg.Monitor.MaxRetryAttempts,
		appLogger,
	)

	bootstrapService := services.NewBootstrapService(
		appLogger,
		"./internal/database/bootstrap.json",
	)

	bootstrapMonitor := services.NewBootstrapMonitor(
		bootstrapRepo,
		statusRepo,
		nodeChecker,
		appLogger,
		bootstrapService,
	)

	// Initialize gRPC services
	grpcServerService := services.NewGRPCServerService(
		appLogger,
		"./internal/database/servers.json",
	)
	grpcChecker := services.NewGRPCChecker(
		cfg.Monitor.ConnectionTimeout,
		cfg.Monitor.MaxRetryAttempts,
		appLogger,
	)
	grpcMonitor := services.NewGRPCMonitor(
		grpcRepo,
		grpcStatusRepo,
		grpcChecker,
		appLogger,
		grpcServerService,
	)

	// Initialize scheduler
	cronScheduler := scheduler.NewCronScheduler(bootstrapMonitor, grpcMonitor, appLogger)
	cronScheduler.Start()
	defer cronScheduler.Stop()

	// Initialize HTTP handlers
	bootstrapHandler := handlers.NewBootstrapHandler(bootstrapMonitor, appLogger)
	grpcHandler := handlers.NewGRPCHandler(grpcMonitor, appLogger)
	healthHandler := handlers.NewHealthHandler(db.DB, appLogger, "1.0.0")
	// Setup Gin router
	if cfg.Logger.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// ============ MIDDLEWARE SETUP ============

	// 1. Request ID - must be first to ensure all logs have request ID
	router.Use(middleware.RequestID())

	// 2. Recovery - catch panics
	router.Use(middleware.Recovery(appLogger))

	// 3. Structured Logging
	router.Use(middleware.StructuredLogger(appLogger))

	// 4. Security Headers
	router.Use(middleware.Security())

	// 5. CORS
	corsConfig := middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:5173", "http://localhost:3000", "https://tracker.kyvra.xyz"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           3600,
	}
	router.Use(middleware.CORS(corsConfig))

	// 6. Rate Limiting - 100 requests per minute per IP
	rateLimiter := middleware.NewRateLimiter(100, time.Minute, appLogger)
	router.Use(rateLimiter.Middleware())

	// 7. Request Timeout - 60 seconds max
	router.Use(middleware.Timeout(60*time.Second, appLogger))

	// ============ API ROUTES ============

	api := router.Group("/api/v1")
	{
		// Bootstrap endpoints
		api.GET("/bootstrap", bootstrapHandler.GetBootstrapNodes)
		api.POST("/bootstrap/sync", bootstrapHandler.SyncBootstrapNodes)
		api.GET("/bootstrap/check", bootstrapHandler.CheckAllNodes)
		api.GET("/bootstrap/count", bootstrapHandler.GetBootstrapNodeCount)

		// gRPC endpoints
		api.GET("/grpc", grpcHandler.GetGRPCServers)
		api.POST("/grpc/sync", grpcHandler.SyncGRPCServers)
		api.GET("/grpc/check", grpcHandler.CheckAllServers)
		api.GET("/grpc/count", grpcHandler.GetGRPCServerCount)

		// Simple health check
		api.GET("/health", healthHandler.Health)

		// Rate limiter stats (for monitoring)
		api.GET("/stats/rate-limiter", func(c *gin.Context) {
			c.JSON(http.StatusOK, rateLimiter.GetStats())
		})
	}
	// Metrics endpoint at root level (outside /api/v1)
	// router.GET("/metrics", gin.WrapH(metrics.Handler()))  // <-- COMMENT THIS OUT FOR NOW

	// Start server
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	// Graceful shutdown
	go func() {
		appLogger.WithField("addr", serverAddr).Info("Starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Fatal("Failed to start server")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		appLogger.WithError(err).Fatal("Server forced to shutdown")
	}

	appLogger.Info("Server exited")
}
