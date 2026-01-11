package auth

import (
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

// RegisterRoutes registers auth routes to the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/logout", h.Logout)
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
