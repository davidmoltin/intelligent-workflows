package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/auth"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
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
			ctx = context.WithValue(ctx, "organization_id", claims.OrganizationID)
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
			user, organizationID, scopes, err := authService.ValidateAPIKey(r.Context(), apiKey)
			if err != nil {
				log.Warn("Invalid API key", zap.Error(err))
				respondError(w, http.StatusUnauthorized, "Invalid API key")
				return
			}

			// Create claims-like structure for API key
			apiClaims := &auth.JWTClaims{
				UserID:         user.ID,
				OrganizationID: organizationID,
				Username:       user.Username,
				Email:          user.Email,
			}

			// Add to context
			ctx := context.WithValue(r.Context(), "claims", apiClaims)
			ctx = context.WithValue(ctx, "user_id", user.ID)
			ctx = context.WithValue(ctx, "organization_id", organizationID)
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
						ctx = context.WithValue(ctx, "organization_id", claims.OrganizationID)
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
				user, organizationID, scopes, err := authService.ValidateAPIKey(r.Context(), apiKey)
				if err == nil {
					apiClaims := &auth.JWTClaims{
						UserID:         user.ID,
						OrganizationID: organizationID,
						Username:       user.Username,
						Email:          user.Email,
					}
					ctx := context.WithValue(r.Context(), "claims", apiClaims)
					ctx = context.WithValue(ctx, "user_id", user.ID)
					ctx = context.WithValue(ctx, "organization_id", organizationID)
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

// respondError sends an error response with proper JSON encoding
func respondError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	// Use proper JSON encoding to prevent injection attacks
	response := map[string]string{"error": message}
	json.NewEncoder(w).Encode(response)
}

// GetOrganizationID extracts organization ID from request context
// Returns uuid.Nil if not found
func GetOrganizationID(ctx context.Context) uuid.UUID {
	if orgID, ok := ctx.Value("organization_id").(uuid.UUID); ok {
		return orgID
	}
	return uuid.Nil
}

// GetUserID extracts user ID from request context
// Returns uuid.Nil if not found
func GetUserID(ctx context.Context) uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return userID
	}
	return uuid.Nil
}

// GetClaims extracts JWT claims from request context
func GetClaims(ctx context.Context) *auth.JWTClaims {
	if claims, ok := ctx.Value("claims").(*auth.JWTClaims); ok {
		return claims
	}
	return nil
}
