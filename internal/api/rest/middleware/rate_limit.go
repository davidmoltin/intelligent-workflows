package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/davidmoltin/intelligent-workflows/pkg/auth"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimiter manages rate limiting for requests
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   *logger.Logger
}

// NewRateLimiter creates a new rate limiter
// rate: requests per second
// burst: maximum burst size
func NewRateLimiter(rps int, burst int, log *logger.Logger) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		logger:   log,
	}
}

// getLimiter returns a rate limiter for the given identifier
func (rl *RateLimiter) getLimiter(identifier string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[identifier]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[identifier] = limiter
	}

	return limiter
}

// Cleanup removes stale limiters (should be called periodically)
func (rl *RateLimiter) Cleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			// Remove limiters that haven't been used recently
			// This is a simple cleanup - in production, you might want to track last access time
			rl.limiters = make(map[string]*rate.Limiter)
			rl.mu.Unlock()
		case <-ctx.Done():
			return
		}
	}
}

// RateLimit is a middleware that applies rate limiting per user/IP
func RateLimit(rl *RateLimiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Determine identifier (user ID if authenticated, otherwise IP)
			identifier := getIdentifier(r)

			limiter := rl.getLimiter(identifier)

			if !limiter.Allow() {
				rl.logger.Warn("Rate limit exceeded", zap.String("identifier", identifier), zap.String("path", r.URL.Path))
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rl.burst))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				respondError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitWithConfig creates a rate limiter middleware with specific limits
func RateLimitWithConfig(rps int, burst int, log *logger.Logger) func(next http.Handler) http.Handler {
	rl := NewRateLimiter(rps, burst, log)
	return RateLimit(rl)
}

// GlobalRateLimit is a simpler rate limiter that applies globally
func GlobalRateLimit(rps int, burst int, log *logger.Logger) func(next http.Handler) http.Handler {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow() {
				log.Warn("Global rate limit exceeded", zap.String("path", r.URL.Path))
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", burst))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				respondError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// getIdentifier extracts an identifier for rate limiting
func getIdentifier(r *http.Request) string {
	// Try to get user ID from context (if authenticated)
	if claims, ok := r.Context().Value("claims").(*auth.JWTClaims); ok {
		return fmt.Sprintf("user:%s", claims.UserID.String())
	}

	// Try to get from user_id context value
	if userID, ok := r.Context().Value("user_id").(string); ok {
		return fmt.Sprintf("user:%s", userID)
	}

	// Fall back to IP address
	ip := r.RemoteAddr
	// If behind a proxy, try to get the real IP
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		ip = forwarded
	} else if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		ip = realIP
	}

	return fmt.Sprintf("ip:%s", ip)
}

// RateLimitConfig holds configuration for different endpoints
type RateLimitConfig struct {
	Endpoint string
	RPS      int
	Burst    int
}

// EndpointRateLimiter manages rate limits for specific endpoints
type EndpointRateLimiter struct {
	configs        map[string]*RateLimiter
	defaultLimiter *RateLimiter
	logger         *logger.Logger
}

// NewEndpointRateLimiter creates a rate limiter with per-endpoint configs
func NewEndpointRateLimiter(configs []RateLimitConfig, defaultRPS, defaultBurst int, log *logger.Logger) *EndpointRateLimiter {
	erl := &EndpointRateLimiter{
		configs:        make(map[string]*RateLimiter),
		defaultLimiter: NewRateLimiter(defaultRPS, defaultBurst, log),
		logger:         log,
	}

	for _, config := range configs {
		erl.configs[config.Endpoint] = NewRateLimiter(config.RPS, config.Burst, log)
	}

	return erl
}

// Middleware returns the rate limiting middleware
func (erl *EndpointRateLimiter) Middleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get the appropriate rate limiter
			limiter, exists := erl.configs[r.URL.Path]
			if !exists {
				limiter = erl.defaultLimiter
			}

			identifier := getIdentifier(r)
			rateLimiter := limiter.getLimiter(identifier)

			if !rateLimiter.Allow() {
				erl.logger.Warn("Rate limit exceeded",
					zap.String("identifier", identifier),
					zap.String("path", r.URL.Path),
				)
				respondError(w, http.StatusTooManyRequests, "Rate limit exceeded")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
