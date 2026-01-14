package middleware

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	sredis "github.com/ulule/limiter/v3/drivers/store/redis"

	"github.com/jin-chillo/go-chat/internal/cache"
)

// RateLimitConfig holds rate limit configuration.
type RateLimitConfig struct {
	// LoginLimit is the rate limit for login endpoint (requests per minute).
	LoginLimit int
	// RegisterLimit is the rate limit for register endpoint (requests per minute).
	RegisterLimit int
	// GeneralLimit is the rate limit for general API endpoints (requests per minute).
	GeneralLimit int
}

// DefaultRateLimitConfig returns the default rate limit configuration.
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		LoginLimit:    5,   // 5 requests per minute
		RegisterLimit: 3,   // 3 requests per minute
		GeneralLimit:  100, // 100 requests per minute
	}
}

// RateLimiter holds the rate limiters for different endpoints.
type RateLimiter struct {
	loginLimiter    *limiter.Limiter
	registerLimiter *limiter.Limiter
	generalLimiter  *limiter.Limiter
	store           limiter.Store
}

// NewRateLimiter creates a new RateLimiter instance with Redis store.
func NewRateLimiter(redisClient *cache.RedisClient, cfg *RateLimitConfig) (*RateLimiter, error) {
	if cfg == nil {
		cfg = DefaultRateLimitConfig()
	}

	// Create Redis store for rate limiter
	store, err := sredis.NewStoreWithOptions(redisClient.Client(), limiter.StoreOptions{
		Prefix: "ratelimit",
	})
	if err != nil {
		return nil, err
	}

	return &RateLimiter{
		loginLimiter: limiter.New(store, limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  int64(cfg.LoginLimit),
		}),
		registerLimiter: limiter.New(store, limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  int64(cfg.RegisterLimit),
		}),
		generalLimiter: limiter.New(store, limiter.Rate{
			Period: 1 * time.Minute,
			Limit:  int64(cfg.GeneralLimit),
		}),
		store: store,
	}, nil
}

// LoginRateLimit returns the rate limit middleware for login endpoint.
func (r *RateLimiter) LoginRateLimit() gin.HandlerFunc {
	return mgin.NewMiddleware(r.loginLimiter, mgin.WithLimitReachedHandler(rateLimitExceededHandler))
}

// RegisterRateLimit returns the rate limit middleware for register endpoint.
func (r *RateLimiter) RegisterRateLimit() gin.HandlerFunc {
	return mgin.NewMiddleware(r.registerLimiter, mgin.WithLimitReachedHandler(rateLimitExceededHandler))
}

// GeneralRateLimit returns the rate limit middleware for general API endpoints.
func (r *RateLimiter) GeneralRateLimit() gin.HandlerFunc {
	return mgin.NewMiddleware(r.generalLimiter, mgin.WithLimitReachedHandler(rateLimitExceededHandler))
}

// rateLimitExceededHandler handles rate limit exceeded responses.
func rateLimitExceededHandler(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"error": "Rate limit exceeded. Please try again later.",
		"code":  "RATE001",
	})
}
