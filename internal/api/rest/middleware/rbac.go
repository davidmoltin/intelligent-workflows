package middleware

import (
	"net/http"

	"github.com/davidmoltin/intelligent-workflows/pkg/auth"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"go.uber.org/zap"
)

// RequirePermission is a middleware that checks if the user has a specific permission
func RequirePermission(permission string, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context
			claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
			if !ok {
				log.Warn("No claims found in context for permission check")
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Check if user has the required permission
			if !hasPermission(claims.Permissions, permission) {
				log.Warn("Permission denied",
					zap.String("user_id", claims.UserID.String()),
					zap.String("username", claims.Username),
					zap.String("required_permission", permission),
				)
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission is a middleware that checks if the user has any of the specified permissions
func RequireAnyPermission(permissions []string, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context
			claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
			if !ok {
				log.Warn("No claims found in context for permission check")
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Check if user has any of the required permissions
			hasAny := false
			for _, perm := range permissions {
				if hasPermission(claims.Permissions, perm) {
					hasAny = true
					break
				}
			}

			if !hasAny {
				log.Warn("Permission denied - no matching permissions",
					zap.String("user_id", claims.UserID.String()),
					zap.String("username", claims.Username),
					zap.Strings("required_permissions", permissions),
				)
				respondError(w, http.StatusForbidden, "Insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllPermissions is a middleware that checks if the user has all of the specified permissions
func RequireAllPermissions(permissions []string, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context
			claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
			if !ok {
				log.Warn("No claims found in context for permission check")
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Check if user has all required permissions
			for _, perm := range permissions {
				if !hasPermission(claims.Permissions, perm) {
					log.Warn("Permission denied - missing permission",
						zap.String("user_id", claims.UserID.String()),
						zap.String("username", claims.Username),
						zap.String("missing_permission", perm),
					)
					respondError(w, http.StatusForbidden, "Insufficient permissions")
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireRole is a middleware that checks if the user has a specific role
func RequireRole(role string, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context
			claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
			if !ok {
				log.Warn("No claims found in context for role check")
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Check if user has the required role
			if !hasRole(claims.Roles, role) {
				log.Warn("Role check failed",
					zap.String("user_id", claims.UserID.String()),
					zap.String("username", claims.Username),
					zap.String("required_role", role),
				)
				respondError(w, http.StatusForbidden, "Insufficient privileges")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyRole is a middleware that checks if the user has any of the specified roles
func RequireAnyRole(roles []string, log *logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get claims from context
			claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
			if !ok {
				log.Warn("No claims found in context for role check")
				respondError(w, http.StatusUnauthorized, "Unauthorized")
				return
			}

			// Check if user has any of the required roles
			hasAny := false
			for _, role := range roles {
				if hasRole(claims.Roles, role) {
					hasAny = true
					break
				}
			}

			if !hasAny {
				log.Warn("Role check failed - no matching roles",
					zap.String("user_id", claims.UserID.String()),
					zap.String("username", claims.Username),
					zap.Strings("required_roles", roles),
				)
				respondError(w, http.StatusForbidden, "Insufficient privileges")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Helper functions

func hasPermission(userPermissions []string, required string) bool {
	for _, perm := range userPermissions {
		if perm == required {
			return true
		}
	}
	return false
}

func hasRole(userRoles []string, required string) bool {
	for _, role := range userRoles {
		if role == required {
			return true
		}
	}
	return false
}
