package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	authService *AuthService
	jwtService  *JWTService
}

// NewHandler creates a new auth Handler instance.
func NewHandler(authService *AuthService, jwtService *JWTService) *Handler {
	return &Handler{
		authService: authService,
		jwtService:  jwtService,
	}
}

// RouteConfig holds optional middleware configuration for routes.
type RouteConfig struct {
	RegisterRateLimit gin.HandlerFunc
	LoginRateLimit    gin.HandlerFunc
}

// RegisterRoutes registers auth routes to the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, cfg *RouteConfig) {
	auth := rg.Group("/auth")
	{
		if cfg != nil && cfg.RegisterRateLimit != nil {
			auth.POST("/register", cfg.RegisterRateLimit, h.Register)
		} else {
			auth.POST("/register", h.Register)
		}

		if cfg != nil && cfg.LoginRateLimit != nil {
			auth.POST("/login", cfg.LoginRateLimit, h.Login)
		} else {
			auth.POST("/login", h.Login)
		}

		auth.POST("/logout", h.Logout)
		auth.POST("/refresh", h.Refresh)
	}
}

// Register handles user registration.
// POST /api/v1/auth/register
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Code:    "AUTH001",
			Details: err.Error(),
		})
		return
	}

	user, err := h.authService.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error: "Email already registered",
				Code:  "AUTH003",
			})
			return
		}
		log.Printf("failed to register user: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to register user",
		})
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		User: NewUserResponse(user),
	})
}

// Login handles user login.
// POST /api/v1/auth/login
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Code:    "AUTH001",
			Details: err.Error(),
		})
		return
	}

	result, err := h.authService.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid email or password",
				Code:  "AUTH004",
			})
			return
		}
		log.Printf("failed to login user: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to login",
		})
		return
	}

	// Record login history asynchronously (non-blocking)
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	go func() {
		if err := h.authService.RecordLoginHistory(context.Background(), result.UserID, ip, userAgent); err != nil {
			log.Printf("failed to record login history: %v", err)
		}
	}()

	c.JSON(http.StatusOK, LoginResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    result.ExpiresIn,
	})
}

// Logout handles user logout.
// POST /api/v1/auth/logout
func (h *Handler) Logout(c *gin.Context) {
	// Extract token from Authorization header
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Authorization header required",
			Code:  "AUTH006",
		})
		return
	}

	// Parse Bearer token
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid authorization header format",
			Code:  "AUTH006",
		})
		return
	}

	tokenString := parts[1]

	// Validate token and get claims
	claims, err := h.jwtService.ValidateAccessToken(c.Request.Context(), tokenString)
	if err != nil {
		if errors.Is(err, ErrTokenExpired) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Token expired",
				Code:  "AUTH005",
			})
			return
		}
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error: "Invalid token",
			Code:  "AUTH006",
		})
		return
	}

	// Logout user
	if err := h.authService.Logout(c.Request.Context(), claims); err != nil {
		log.Printf("failed to logout user: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to logout",
		})
		return
	}

	c.JSON(http.StatusOK, LogoutResponse{
		Message: "Successfully logged out",
	})
}

// Refresh handles token refresh.
// POST /api/v1/auth/refresh
func (h *Handler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Validation failed",
			Code:    "AUTH001",
			Details: err.Error(),
		})
		return
	}

	result, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) || errors.Is(err, ErrRefreshTokenRevoked) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Invalid refresh token",
				Code:  "AUTH006",
			})
			return
		}
		if errors.Is(err, ErrRefreshTokenExpired) {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error: "Refresh token expired",
				Code:  "AUTH005",
			})
			return
		}
		log.Printf("failed to refresh token: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error: "Failed to refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, RefreshResponse{
		AccessToken: result.AccessToken,
		TokenType:   "Bearer",
		ExpiresIn:   result.ExpiresIn,
	})
}
