package auth

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for authentication.
type Handler struct {
	authService *AuthService
}

// NewHandler creates a new auth Handler instance.
func NewHandler(authService *AuthService) *Handler {
	return &Handler{
		authService: authService,
	}
}

// RegisterRoutes registers auth routes to the router group.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/register", h.Register)
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
