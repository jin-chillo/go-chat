package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jin-chillo/go-chat/internal/auth"
)

// Context keys for user information.
const (
	ContextKeyUserID   = "user_id"
	ContextKeyEmail    = "email"
	ContextKeyNickname = "nickname"
	ContextKeyClaims   = "claims"
)

// AuthMiddleware creates a JWT authentication middleware.
func AuthMiddleware(jwtService *auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Authorization header required",
				"code":  "AUTH006",
			})
			return
		}

		// Parse Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid authorization header format",
				"code":  "AUTH006",
			})
			return
		}

		tokenString := parts[1]

		// Validate token
		claims, err := jwtService.ValidateAccessToken(c.Request.Context(), tokenString)
		if err != nil {
			if errors.Is(err, auth.ErrTokenExpired) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Token expired",
					"code":  "AUTH005",
				})
				return
			}
			if errors.Is(err, auth.ErrTokenBlacklisted) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "Token has been revoked",
					"code":  "AUTH006",
				})
				return
			}
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid token",
				"code":  "AUTH006",
			})
			return
		}

		// Set user information in context
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyNickname, claims.Nickname)
		c.Set(ContextKeyClaims, claims)

		c.Next()
	}
}

// GetUserID extracts user ID from the context.
func GetUserID(c *gin.Context) (string, bool) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	return userID.(string), true
}

// GetEmail extracts email from the context.
func GetEmail(c *gin.Context) (string, bool) {
	email, exists := c.Get(ContextKeyEmail)
	if !exists {
		return "", false
	}
	return email.(string), true
}

// GetNickname extracts nickname from the context.
func GetNickname(c *gin.Context) (string, bool) {
	nickname, exists := c.Get(ContextKeyNickname)
	if !exists {
		return "", false
	}
	return nickname.(string), true
}

// GetClaims extracts token claims from the context.
func GetClaims(c *gin.Context) (*auth.TokenClaims, bool) {
	claims, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}
	return claims.(*auth.TokenClaims), true
}
