package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// User represents a system user
type User struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	Username     string     `json:"username" db:"username"`
	Email        string     `json:"email" db:"email"`
	PasswordHash string     `json:"-" db:"password_hash"` // Never expose in JSON
	FirstName    *string    `json:"first_name,omitempty" db:"first_name"`
	LastName     *string    `json:"last_name,omitempty" db:"last_name"`
	IsActive     bool       `json:"is_active" db:"is_active"`
	IsVerified   bool       `json:"is_verified" db:"is_verified"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
	Roles        []Role     `json:"roles,omitempty" db:"-"` // Loaded separately
}

// Role represents a user role
type Role struct {
	ID          uuid.UUID    `json:"id" db:"id"`
	Name        string       `json:"name" db:"name"`
	Description *string      `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at" db:"updated_at"`
	Permissions []Permission `json:"permissions,omitempty" db:"-"` // Loaded separately
}

// Permission represents a specific permission
type Permission struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Resource    string    `json:"resource" db:"resource"`
	Action      string    `json:"action" db:"action"`
	Description *string   `json:"description,omitempty" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// UserRole represents the user-role association
type UserRole struct {
	UserID     uuid.UUID  `json:"user_id" db:"user_id"`
	RoleID     uuid.UUID  `json:"role_id" db:"role_id"`
	AssignedAt time.Time  `json:"assigned_at" db:"assigned_at"`
	AssignedBy *uuid.UUID `json:"assigned_by,omitempty" db:"assigned_by"`
}

// RolePermission represents the role-permission association
type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id" db:"role_id"`
	PermissionID uuid.UUID `json:"permission_id" db:"permission_id"`
	GrantedAt    time.Time `json:"granted_at" db:"granted_at"`
}

// APIKey represents an API key for agent authentication
type APIKey struct {
	ID         uuid.UUID      `json:"id" db:"id"`
	KeyHash    string         `json:"-" db:"key_hash"` // Never expose
	KeyPrefix  string         `json:"key_prefix" db:"key_prefix"`
	Name       string         `json:"name" db:"name"`
	UserID     uuid.UUID      `json:"user_id" db:"user_id"`
	Scopes     pq.StringArray `json:"scopes" db:"scopes"`
	IsActive   bool           `json:"is_active" db:"is_active"`
	LastUsedAt *time.Time     `json:"last_used_at,omitempty" db:"last_used_at"`
	ExpiresAt  *time.Time     `json:"expires_at,omitempty" db:"expires_at"`
	CreatedAt  time.Time      `json:"created_at" db:"created_at"`
	CreatedBy  *uuid.UUID     `json:"created_by,omitempty" db:"created_by"`
}

// RefreshToken represents a JWT refresh token
type RefreshToken struct {
	ID        uuid.UUID  `json:"id" db:"id"`
	TokenHash string     `json:"-" db:"token_hash"` // Never expose
	UserID    uuid.UUID  `json:"user_id" db:"user_id"`
	ExpiresAt time.Time  `json:"expires_at" db:"expires_at"`
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// RateLimit represents rate limit tracking
type RateLimit struct {
	ID             uuid.UUID `json:"id" db:"id"`
	Identifier     string    `json:"identifier" db:"identifier"`
	IdentifierType string    `json:"identifier_type" db:"identifier_type"`
	Endpoint       string    `json:"endpoint" db:"endpoint"`
	RequestCount   int       `json:"request_count" db:"request_count"`
	WindowStart    time.Time `json:"window_start" db:"window_start"`
	LastRequestAt  time.Time `json:"last_request_at" db:"last_request_at"`
}

// LoginAttempt represents a login attempt for security tracking
type LoginAttempt struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Username    string    `json:"username" db:"username"`
	IPAddress   string    `json:"ip_address" db:"ip_address"`
	Success     bool      `json:"success" db:"success"`
	AttemptedAt time.Time `json:"attempted_at" db:"attempted_at"`
}

// Request/Response DTOs

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username  string  `json:"username" validate:"required,min=3,max=50"`
	Email     string  `json:"email" validate:"required,email"`
	Password  string  `json:"password" validate:"required,min=8"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse represents a login response with tokens
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int       `json:"expires_in"` // seconds
	TokenType    string    `json:"token_type"`
	User         User      `json:"user"`
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// CreateAPIKeyRequest represents a request to create an API key
type CreateAPIKeyRequest struct {
	Name      string    `json:"name" validate:"required"`
	Scopes    []string  `json:"scopes" validate:"required"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// CreateAPIKeyResponse represents the response when creating an API key
type CreateAPIKeyResponse struct {
	APIKey    string    `json:"api_key"` // Full key, only shown once
	KeyPrefix string    `json:"key_prefix"`
	Name      string    `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// UpdateUserRequest represents a request to update user information
type UpdateUserRequest struct {
	FirstName  *string `json:"first_name,omitempty"`
	LastName   *string `json:"last_name,omitempty"`
	Email      *string `json:"email,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
	IsVerified *bool   `json:"is_verified,omitempty"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// AssignRoleRequest represents a request to assign a role to a user
type AssignRoleRequest struct {
	RoleID uuid.UUID `json:"role_id" validate:"required"`
}

// Claims represents JWT token claims
type Claims struct {
	UserID      uuid.UUID `json:"user_id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int
}

// Scan implementations for custom types

func (c *Claims) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, c)
}

func (c Claims) Value() (driver.Value, error) {
	return json.Marshal(c)
}
