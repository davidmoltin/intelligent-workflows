package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/services"
	"github.com/davidmoltin/intelligent-workflows/pkg/auth"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/davidmoltin/intelligent-workflows/pkg/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	logger      *logger.Logger
	authService *services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(log *logger.Logger, authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		logger:      log,
		authService: authService,
	}
}

// Register handles user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	user, err := h.authService.Register(r.Context(), &req)
	if err != nil {
		h.logger.Errorf("Failed to register user", logger.Err(err))
		// Don't leak internal error details
		h.respondError(w, http.StatusBadRequest, "Failed to register user")
		return
	}

	h.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user":    user,
	})
}

// Login handles user login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	loginResp, err := h.authService.Login(r.Context(), &req)
	if err != nil {
		h.logger.Errorf("Failed to login", logger.Err(err))
		h.respondError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	h.respondJSON(w, http.StatusOK, loginResp)
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var req models.RefreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokenPair, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Errorf("Failed to refresh token", logger.Err(err))
		h.respondError(w, http.StatusUnauthorized, "Invalid or expired refresh token")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  tokenPair.AccessToken,
		"refresh_token": tokenPair.RefreshToken,
		"expires_in":    tokenPair.ExpiresIn,
		"token_type":    "Bearer",
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from body
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := h.authService.Logout(r.Context(), req.RefreshToken); err != nil {
		h.logger.Errorf("Failed to logout", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to logout")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// Me returns the current user's information
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	// Get user from context (set by auth middleware)
	claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":     claims.UserID,
		"username":    claims.Username,
		"email":       claims.Email,
		"roles":       claims.Roles,
		"permissions": claims.Permissions,
	})
}

// ChangePassword handles password change
func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.ChangePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authService.ChangePassword(r.Context(), claims.UserID, &req); err != nil {
		h.logger.Errorf("Failed to change password", logger.Err(err))
		// Don't leak internal error details
		h.respondError(w, http.StatusBadRequest, "Failed to change password")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "Password changed successfully"})
}

// CreateAPIKey creates a new API key for the authenticated user
func (h *AuthHandler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
	if !ok {
		h.respondError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var req models.CreateAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := validator.Validate(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	apiKeyResp, err := h.authService.CreateAPIKey(r.Context(), claims.OrganizationID, claims.UserID, &req)
	if err != nil {
		h.logger.Errorf("Failed to create API key", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to create API key")
		return
	}

	h.respondJSON(w, http.StatusCreated, apiKeyResp)
}

// RevokeAPIKey revokes an API key
func (h *AuthHandler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "Invalid API key ID")
		return
	}

	// Get organization ID from context (set by auth middleware)
	organizationID, ok := r.Context().Value("organization_id").(uuid.UUID)
	if !ok {
		organizationID = uuid.Nil
	}

	if err := h.authService.RevokeAPIKey(r.Context(), organizationID, id); err != nil {
		h.logger.Errorf("Failed to revoke API key", logger.Err(err))
		h.respondError(w, http.StatusInternalServerError, "Failed to revoke API key")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]string{"message": "API key revoked successfully"})
}

// GetUserFromToken extracts user ID from authorization header (helper for other handlers)
func GetUserFromToken(r *http.Request) (*auth.JWTClaims, error) {
	claims, ok := r.Context().Value("claims").(*auth.JWTClaims)
	if !ok {
		return nil, nil
	}
	return claims, nil
}

// ExtractBearerToken extracts the Bearer token from Authorization header
func ExtractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// Helper methods

func (h *AuthHandler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *AuthHandler) respondError(w http.ResponseWriter, status int, message string) {
	h.respondJSON(w, status, map[string]string{"error": message})
}
