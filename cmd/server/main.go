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
	"github.com/rs/cors"

	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/config"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/database"
	"github.com/kyvra-tech/pactus-nodes-tracker-backend/internal/handlers"
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
	// Initialize services
	nodeChecker := services.NewNodeChecker(
		cfg.Monitor.ConnectionTimeout,
		cfg.Monitor.MaxRetryAttempts,
		appLogger,
	)

	bootstrapMonitor := services.NewBootstrapMonitor(db, nodeChecker, appLogger)

	// Initialize scheduler
	cronScheduler := scheduler.NewCronScheduler(bootstrapMonitor, appLogger)
	cronScheduler.Start()
	defer cronScheduler.Stop()

	// Initialize HTTP handlers
	bootstrapHandler := handlers.NewBootstrapHandler(bootstrapMonitor, appLogger)

	// Setup Gin router
	if cfg.Logger.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	})
	router.Use(func(ctx *gin.Context) {
		c.HandlerFunc(ctx.Writer, ctx.Request)
		ctx.Next()
	})

	// API routes
	api := router.Group("/api/v1")
	{
		api.GET("/bootstrap", bootstrapHandler.GetBootstrapNodes)
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"status":    "healthy",
				"timestamp": time.Now().UTC(),
				"version":   "1.0.0",
			})
		})
	}

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
