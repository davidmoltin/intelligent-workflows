package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/auth"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"go.uber.org/zap"
)

// JWTAuth is a middleware that validates JWT tokens
func JWTAuth(authService *services.AuthService, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				respondError(w, http.StatusUnauthorized, "Missing authorization header")
				return
			}

			// Check Bearer token format
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				respondError(w, http.StatusUnauthorized, "Invalid authorization header format")
				return
			}

			tokenString := parts[1]

			// Validate token
			claims, err := authService.ValidateAccessToken(tokenString)
			if err != nil {
				log.Warn("Invalid JWT token", zap.Error(err))
				respondError(w, http.StatusUnauthorized, "Invalid or expired token")
				return
			}

			// Add claims to context
			ctx := context.WithValue(r.Context(), "claims", claims)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "username", claims.Username)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// APIKeyAuth is a middleware that validates API keys
func APIKeyAuth(authService *services.AuthService, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from X-API-Key header
			apiKey := r.Header.Get("X-API-Key")
			if apiKey == "" {
				respondError(w, http.StatusUnauthorized, "Missing API key")
				return
			}

			// Validate API key
			user, scopes, err := authService.ValidateAPIKey(r.Context(), apiKey)
			if err != nil {
				log.Warn("Invalid API key", zap.Error(err))
				respondError(w, http.StatusUnauthorized, "Invalid API key")
				return
			}

			// Create claims-like structure for API key
			apiClaims := &auth.JWTClaims{
				UserID:   user.ID,
				Username: user.Username,
				Email:    user.Email,
			}

			// Add to context
			ctx := context.WithValue(r.Context(), "claims", apiClaims)
			ctx = context.WithValue(ctx, "user_id", user.ID)
			ctx = context.WithValue(ctx, "username", user.Username)
			ctx = context.WithValue(ctx, "scopes", scopes)
			ctx = context.WithValue(ctx, "auth_type", "api_key")

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuth allows both JWT and API key authentication
func OptionalAuth(authService *services.AuthService, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try JWT first
			authHeader := r.Header.Get("Authorization")
			if authHeader != "" {
				parts := strings.SplitN(authHeader, " ", 2)
				if len(parts) == 2 && parts[0] == "Bearer" {
					claims, err := authService.ValidateAccessToken(parts[1])
					if err == nil {
						ctx := context.WithValue(r.Context(), "claims", claims)
						ctx = context.WithValue(ctx, "user_id", claims.UserID)
						ctx = context.WithValue(ctx, "username", claims.Username)
						ctx = context.WithValue(ctx, "auth_type", "jwt")
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}

			// Try API key
			apiKey := r.Header.Get("X-API-Key")
			if apiKey != "" {
				user, scopes, err := authService.ValidateAPIKey(r.Context(), apiKey)
				if err == nil {
					apiClaims := &auth.JWTClaims{
						UserID:   user.ID,
						Username: user.Username,
						Email:    user.Email,
					}
					ctx := context.WithValue(r.Context(), "claims", apiClaims)
					ctx = context.WithValue(ctx, "user_id", user.ID)
					ctx = context.WithValue(ctx, "username", user.Username)
					ctx = context.WithValue(ctx, "scopes", scopes)
					ctx = context.WithValue(ctx, "auth_type", "api_key")
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			// No valid auth provided
			respondError(w, http.StatusUnauthorized, "Authentication required")
		})
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + message + `"}`))
}
