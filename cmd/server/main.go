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
	"github.com/jin-chillo/go-chat/internal/chat"
	"github.com/jin-chillo/go-chat/internal/config"
	"github.com/jin-chillo/go-chat/internal/database"
	"github.com/jin-chillo/go-chat/internal/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/jin-chillo/go-chat/docs"
)

// @title GoChat API
// @version 1.0
// @description JWT 기반 인증과 WebSocket 실시간 채팅 기능을 제공하는 Go 백엔드 서버
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/jin-chillo/go-chat

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer 토큰을 입력하세요 (예: Bearer eyJhbGciOiJIUzI1NiIs...)

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

	// Initialize MongoDB
	mongoDB, err := database.NewMongoDB(&cfg.MongoDB)
	if err != nil {
		log.Fatalf("failed to connect to mongodb: %v", err)
	}
	defer func() {
		if err := mongoDB.Close(context.Background()); err != nil {
			log.Printf("failed to close mongodb: %v", err)
		}
	}()

	// Create MongoDB indexes
	if err := mongoDB.CreateIndexes(context.Background()); err != nil {
		log.Printf("warning: failed to create mongodb indexes: %v", err)
	}

	// Initialize repositories
	userRepo := auth.NewUserRepository(db)
	tokenRepo := auth.NewTokenRepository(db)
	loginHistoryRepo := auth.NewLoginHistoryRepository(mongoDB.Database())

	// Initialize services
	jwtService := auth.NewJWTService(&cfg.JWT, redisClient)
	authService := auth.NewAuthService(userRepo, tokenRepo, jwtService)
	authService.SetLoginHistoryRepo(loginHistoryRepo)

	// Initialize chat repository and service
	channelRepo := chat.NewChannelRepository(db)
	messageRepo := chat.NewMessageRepository(mongoDB.Database())
	chatService := chat.NewService(channelRepo, messageRepo)

	// Initialize presence service for online status
	presenceService := chat.NewPresenceService(redisClient)

	// Initialize WebSocket hub
	hub := chat.NewHub()
	go hub.Run()

	// Initialize handlers
	authHandler := auth.NewHandler(authService, jwtService)
	chatHandler := chat.NewHandler(chatService, presenceService)
	wsHandler := chat.NewWebSocketHandler(hub, chatService, jwtService, presenceService)

	// Setup disconnect handler for presence cleanup
	wsHandler.SetupDisconnectHandler()

	// Initialize rate limiter
	rateLimiter, err := middleware.NewRateLimiter(redisClient, nil)
	if err != nil {
		log.Fatalf("failed to create rate limiter: %v", err)
	}

	// Set Gin mode
	gin.SetMode(cfg.Server.GinMode)

	// Create router
	router := setupRouter(authHandler, chatHandler, wsHandler, jwtService, rateLimiter)

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

func setupRouter(authHandler *auth.Handler, chatHandler *chat.Handler, wsHandler *chat.WebSocketHandler, jwtService *auth.JWTService, rateLimiter *middleware.RateLimiter) *gin.Engine {
	router := gin.New()

	// Middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(corsMiddleware())

	// Health check
	router.GET("/health", healthHandler)

	// Swagger documentation
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// WebSocket routes
	wsHandler.RegisterRoutes(router)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Auth routes with rate limiting
		authHandler.RegisterRoutes(v1, &auth.RouteConfig{
			RegisterRateLimit: rateLimiter.RegisterRateLimit(),
			LoginRateLimit:    rateLimiter.LoginRateLimit(),
		})

		// Chat routes (protected)
		authMiddleware := middleware.AuthMiddleware(jwtService)
		chatHandler.RegisterRoutes(v1, authMiddleware)
	}

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
