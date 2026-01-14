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
// @Summary 회원가입
// @Description 새로운 사용자를 등록합니다
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RegisterRequest true "회원가입 정보"
// @Success 201 {object} RegisterResponse "회원가입 성공"
// @Failure 400 {object} ErrorResponse "유효성 검증 실패"
// @Failure 409 {object} ErrorResponse "이메일 중복"
// @Failure 500 {object} ErrorResponse "서버 에러"
// @Router /auth/register [post]
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
// @Summary 로그인
// @Description 이메일과 비밀번호로 로그인하여 JWT 토큰을 발급받습니다
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body LoginRequest true "로그인 정보"
// @Success 200 {object} LoginResponse "로그인 성공"
// @Failure 400 {object} ErrorResponse "유효성 검증 실패"
// @Failure 401 {object} ErrorResponse "인증 실패"
// @Failure 500 {object} ErrorResponse "서버 에러"
// @Router /auth/login [post]
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
// @Summary 로그아웃
// @Description 현재 액세스 토큰을 무효화하고 로그아웃합니다
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LogoutResponse "로그아웃 성공"
// @Failure 401 {object} ErrorResponse "인증 실패"
// @Failure 500 {object} ErrorResponse "서버 에러"
// @Router /auth/logout [post]
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
// @Summary 토큰 갱신
// @Description 리프레시 토큰으로 새로운 액세스 토큰을 발급받습니다
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshRequest true "리프레시 토큰"
// @Success 200 {object} RefreshResponse "토큰 갱신 성공"
// @Failure 400 {object} ErrorResponse "유효성 검증 실패"
// @Failure 401 {object} ErrorResponse "인증 실패"
// @Failure 500 {object} ErrorResponse "서버 에러"
// @Router /auth/refresh [post]
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
