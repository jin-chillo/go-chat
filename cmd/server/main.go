package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jin-chillo/go-chat/internal/auth"
	"github.com/jin-chillo/go-chat/internal/cache"
	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/database"
	"github.com/jin-chillo/go-chat/internal/middleware"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize PostgreSQL
	db, err := database.NewPostgresDB(&cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer func() {
		if err := database.Close(db); err != nil {
			log.Printf("failed to close database: %v", err)
		}
	}()

	// Initialize Redis
	redisClient, err := cache.NewRedisClient(&cfg.Redis)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Printf("failed to close redis: %v", err)
		}
	}()

	// Initialize repositories
	userRepo := auth.NewUserRepository(db)
	tokenRepo := auth.NewTokenRepository(db)

	// Initialize services
	jwtService := auth.NewJWTService(&cfg.JWT, redisClient)
	authService := auth.NewAuthService(userRepo, tokenRepo, jwtService)

	// Initialize handlers
	authHandler := auth.NewHandler(authService, jwtService)

	// Initialize rate limiter
	rateLimiter, err := middleware.NewRateLimiter(redisClient, nil)
	if err != nil {
		log.Fatalf("failed to create rate limiter: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Create router
	router := setupRouter(authHandler, jwtService, rateLimiter)

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server starting on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

func setupRouter(authHandler *auth.Handler, jwtService *auth.JWTService, rateLimiter *middleware.RateLimiter) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", healthHandler)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes with rate limiting
		authHandler.RegisterRoutes(v1, &auth.RouteConfig{
			RegisterRateLimit: rateLimiter.RegisterRateLimit(),
			LoginRateLimit:    rateLimiter.LoginRateLimit(),
		})

		// Protected routes (example - to be used by future handlers)
		// protected := v1.Group("")
		// protected.Use(middleware.AuthMiddleware(jwtService))
		// protected.Use(rateLimiter.GeneralRateLimit())
	}

	// Export jwtService for future protected routes
	_ = jwtService

	return router
}

func corsMiddleware() gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	})
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
