package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	// Token expiration times
	AccessTokenDuration  = 15 * time.Minute
	RefreshTokenDuration = 7 * 24 * time.Hour
)

// JWTManager handles JWT token operations
type JWTManager struct {
	secretKey        []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
}

// JWTClaims represents the claims in our JWT token
type JWTClaims struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
	jwt.RegisteredClaims
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string) *JWTManager {
	return &JWTManager{
		secretKey:        []byte(secretKey),
		accessTokenTTL:   AccessTokenDuration,
		refreshTokenTTL:  RefreshTokenDuration,
	}
}

// NewJWTManagerWithTTL creates a new JWT manager with custom TTL
func NewJWTManagerWithTTL(secretKey string, accessTTL, refreshTTL time.Duration) *JWTManager {
	return &JWTManager{
		secretKey:        []byte(secretKey),
		accessTokenTTL:   accessTTL,
		refreshTokenTTL:  refreshTTL,
	}
}

// GenerateAccessToken generates a new JWT access token
func (m *JWTManager) GenerateAccessToken(userID uuid.UUID, username, email string, roles, permissions []string) (string, error) {
	now := time.Now()

	claims := &JWTClaims{
		UserID:      userID,
		Username:    username,
		Email:       email,
		Roles:       roles,
		Permissions: permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "intelligent-workflows",
			Subject:   userID.String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

// GenerateRefreshToken generates a cryptographically secure refresh token
func (m *JWTManager) GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// ValidateAccessToken validates and parses a JWT access token
func (m *JWTManager) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GetAccessTokenTTL returns the access token TTL in seconds
func (m *JWTManager) GetAccessTokenTTL() int {
	return int(m.accessTokenTTL.Seconds())
}

// GetRefreshTokenTTL returns the refresh token TTL
func (m *JWTManager) GetRefreshTokenTTL() time.Duration {
	return m.refreshTokenTTL
}
