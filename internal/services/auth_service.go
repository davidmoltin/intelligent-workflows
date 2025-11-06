package services

import (
	"context"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/davidmoltin/intelligent-workflows/internal/repository/postgres"
	"github.com/davidmoltin/intelligent-workflows/pkg/auth"
	"github.com/davidmoltin/intelligent-workflows/pkg/logger"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// AuthService handles authentication business logic
type AuthService struct {
	userRepo         *postgres.UserRepository
	apiKeyRepo       *postgres.APIKeyRepository
	refreshTokenRepo *postgres.RefreshTokenRepository
	orgRepo          *postgres.OrganizationRepository
	jwtManager       *auth.JWTManager
	logger           *logger.Logger
}

// NewAuthService creates a new auth service
func NewAuthService(
	userRepo *postgres.UserRepository,
	apiKeyRepo *postgres.APIKeyRepository,
	refreshTokenRepo *postgres.RefreshTokenRepository,
	orgRepo *postgres.OrganizationRepository,
	jwtManager *auth.JWTManager,
	log *logger.Logger,
) *AuthService {
	return &AuthService{
		userRepo:         userRepo,
		apiKeyRepo:       apiKeyRepo,
		refreshTokenRepo: refreshTokenRepo,
		orgRepo:          orgRepo,
		jwtManager:       jwtManager,
		logger:           log,
	}
}

// Register registers a new user
func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.User, error) {
	// Check if username already exists
	existingUser, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("username already exists")
	}

	// Check if email already exists
	existingUser, err = s.userRepo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("email already exists")
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		IsActive:     true,
		IsVerified:   false, // Requires email verification
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	s.logger.Info("User registered", zap.String("user_id", user.ID.String()), zap.String("username", user.Username))

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.LoginResponse, error) {
	// Get user
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Verify password
	if err := auth.VerifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Get user's organizations
	orgs, err := s.orgRepo.GetUserOrganizations(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	// Ensure user belongs to at least one organization
	if len(orgs) == 0 {
		return nil, fmt.Errorf("user does not belong to any organization")
	}

	// Use the first organization as default (in future, we can add user preference for default org)
	organizationID := orgs[0].ID

	// Get user roles and permissions
	roles, err := s.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	permissions, err := s.userRepo.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Generate access token with organization ID
	accessToken, err := s.jwtManager.GenerateAccessToken(
		user.ID,
		organizationID,
		user.Username,
		user.Email,
		roles,
		permissions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshTokenString, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	refreshToken := &models.RefreshToken{
		TokenHash: auth.HashRefreshToken(refreshTokenString),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.jwtManager.GetRefreshTokenTTL()),
	}

	if err := s.refreshTokenRepo.Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Update last login
	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("Failed to update last login", zap.Error(err))
	}

	s.logger.Info("User logged in", zap.String("user_id", user.ID.String()), zap.String("username", user.Username))

	// Remove password hash from response
	user.PasswordHash = ""

	return &models.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		ExpiresIn:    s.jwtManager.GetAccessTokenTTL(),
		TokenType:    "Bearer",
		User:         *user,
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (s *AuthService) RefreshToken(ctx context.Context, refreshTokenString string) (*models.TokenPair, error) {
	// Hash the refresh token
	tokenHash := auth.HashRefreshToken(refreshTokenString)

	// Get refresh token from database
	refreshToken, err := s.refreshTokenRepo.GetByHash(ctx, tokenHash)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Check if token is expired
	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, fmt.Errorf("refresh token expired")
	}

	// Check if token is revoked
	if refreshToken.RevokedAt != nil {
		return nil, fmt.Errorf("refresh token revoked")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, refreshToken.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Get user's organizations
	orgs, err := s.orgRepo.GetUserOrganizations(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user organizations: %w", err)
	}

	// Ensure user belongs to at least one organization
	if len(orgs) == 0 {
		return nil, fmt.Errorf("user does not belong to any organization")
	}

	// Use the first organization as default
	organizationID := orgs[0].ID

	// Get user roles and permissions
	roles, err := s.userRepo.GetUserRoles(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user roles: %w", err)
	}

	permissions, err := s.userRepo.GetUserPermissions(ctx, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user permissions: %w", err)
	}

	// Generate new access token with organization ID
	accessToken, err := s.jwtManager.GenerateAccessToken(
		user.ID,
		organizationID,
		user.Username,
		user.Email,
		roles,
		permissions,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate new refresh token
	newRefreshTokenString, err := s.jwtManager.GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Revoke old refresh token
	if err := s.refreshTokenRepo.Revoke(ctx, tokenHash); err != nil {
		s.logger.Warn("Failed to revoke old refresh token", zap.Error(err))
	}

	// Store new refresh token
	newRefreshToken := &models.RefreshToken{
		TokenHash: auth.HashRefreshToken(newRefreshTokenString),
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(s.jwtManager.GetRefreshTokenTTL()),
	}

	if err := s.refreshTokenRepo.Create(ctx, newRefreshToken); err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: newRefreshTokenString,
		ExpiresIn:    s.jwtManager.GetAccessTokenTTL(),
	}, nil
}

// Logout logs out a user by revoking their refresh token
func (s *AuthService) Logout(ctx context.Context, refreshTokenString string) error {
	tokenHash := auth.HashRefreshToken(refreshTokenString)
	if err := s.refreshTokenRepo.Revoke(ctx, tokenHash); err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}
	return nil
}

// ValidateAccessToken validates an access token and returns the claims
func (s *AuthService) ValidateAccessToken(tokenString string) (*auth.JWTClaims, error) {
	return s.jwtManager.ValidateAccessToken(tokenString)
}

// CreateAPIKey creates a new API key for a user
func (s *AuthService) CreateAPIKey(ctx context.Context, userID uuid.UUID, req *models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	// Generate API key
	apiKeyString, err := auth.GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Create API key record
	apiKey := &models.APIKey{
		KeyHash:   auth.HashAPIKey(apiKeyString),
		KeyPrefix: auth.GetAPIKeyPrefix(apiKeyString),
		Name:      req.Name,
		UserID:    userID,
		Scopes:    req.Scopes,
		IsActive:  true,
		ExpiresAt: req.ExpiresAt,
		CreatedBy: &userID,
	}

	if err := s.apiKeyRepo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	s.logger.Info("API key created", zap.String("user_id", userID.String()), zap.String("key_name", req.Name))

	return &models.CreateAPIKeyResponse{
		APIKey:    apiKeyString,
		KeyPrefix: apiKey.KeyPrefix,
		Name:      apiKey.Name,
		ExpiresAt: apiKey.ExpiresAt,
	}, nil
}

// ValidateAPIKey validates an API key and returns the associated user
func (s *AuthService) ValidateAPIKey(ctx context.Context, apiKeyString string) (*models.User, uuid.UUID, []string, error) {
	// Hash the API key
	keyHash := auth.HashAPIKey(apiKeyString)

	// Get API key from database
	apiKey, err := s.apiKeyRepo.GetByHash(ctx, keyHash)
	if err != nil {
		return nil, uuid.Nil, nil, fmt.Errorf("invalid API key")
	}

	// Check if API key is active
	if !apiKey.IsActive {
		return nil, uuid.Nil, nil, fmt.Errorf("API key is disabled")
	}

	// Check if API key is expired
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, uuid.Nil, nil, fmt.Errorf("API key expired")
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, apiKey.UserID)
	if err != nil {
		return nil, uuid.Nil, nil, fmt.Errorf("user not found")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, uuid.Nil, nil, fmt.Errorf("user account is disabled")
	}

	// Update last used timestamp
	if err := s.apiKeyRepo.UpdateLastUsed(ctx, apiKey.ID); err != nil {
		s.logger.Warn("Failed to update API key last used", zap.Error(err))
	}

	return user, apiKey.OrganizationID, apiKey.Scopes, nil
}

// RevokeAPIKey revokes an API key
func (s *AuthService) RevokeAPIKey(ctx context.Context, keyID uuid.UUID) error {
	return s.apiKeyRepo.Revoke(ctx, keyID)
}

// ChangePassword changes a user's password
func (s *AuthService) ChangePassword(ctx context.Context, userID uuid.UUID, req *models.ChangePasswordRequest) error {
	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	// Verify old password
	if err := auth.VerifyPassword(req.OldPassword, user.PasswordHash); err != nil {
		return fmt.Errorf("invalid old password")
	}

	// Hash new password
	newPasswordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(ctx, userID, newPasswordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all refresh tokens for security
	if err := s.refreshTokenRepo.RevokeAllForUser(ctx, userID); err != nil {
		s.logger.Warn("Failed to revoke refresh tokens", zap.Error(err))
	}

	s.logger.Info("Password changed", zap.String("user_id", userID.String()))

	return nil
}
