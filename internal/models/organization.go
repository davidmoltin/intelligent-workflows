package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Organization represents a tenant/organization in the system
type Organization struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Slug        string    `json:"slug" db:"slug"`
	Description *string   `json:"description,omitempty" db:"description"`
	Settings    JSONB     `json:"settings,omitempty" db:"settings"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
	CreatedBy   uuid.UUID `json:"created_by,omitempty" db:"created_by"`
}

// OrganizationUser represents a user's membership in an organization
type OrganizationUser struct {
	ID             uuid.UUID  `json:"id" db:"id"`
	OrganizationID uuid.UUID  `json:"organization_id" db:"organization_id"`
	UserID         uuid.UUID  `json:"user_id" db:"user_id"`
	RoleID         uuid.UUID  `json:"role_id" db:"role_id"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	JoinedAt       time.Time  `json:"joined_at" db:"joined_at"`
	InvitedBy      *uuid.UUID `json:"invited_by,omitempty" db:"invited_by"`

	// Populated via joins
	User         *User         `json:"user,omitempty" db:"-"`
	Role         *Role         `json:"role,omitempty" db:"-"`
	Organization *Organization `json:"organization,omitempty" db:"-"`
}

// CreateOrganizationRequest represents a request to create an organization
type CreateOrganizationRequest struct {
	Name        string                 `json:"name" validate:"required,min=2,max=255"`
	Slug        string                 `json:"slug" validate:"required,min=2,max=255,alphanum"`
	Description *string                `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
}

// UpdateOrganizationRequest represents a request to update an organization
type UpdateOrganizationRequest struct {
	Name        *string                `json:"name,omitempty" validate:"omitempty,min=2,max=255"`
	Description *string                `json:"description,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	IsActive    *bool                  `json:"is_active,omitempty"`
}

// InviteUserToOrganizationRequest represents a request to invite a user to an organization
type InviteUserToOrganizationRequest struct {
	UserID uuid.UUID `json:"user_id" validate:"required"`
	RoleID uuid.UUID `json:"role_id" validate:"required"`
}

// UpdateOrganizationUserRequest represents a request to update an organization user
type UpdateOrganizationUserRequest struct {
	RoleID   *uuid.UUID `json:"role_id,omitempty"`
	IsActive *bool      `json:"is_active,omitempty"`
}

// OrganizationListResponse represents a paginated list of organizations
type OrganizationListResponse struct {
	Organizations []Organization `json:"organizations"`
	Total         int64          `json:"total"`
	Page          int            `json:"page"`
	PageSize      int            `json:"page_size"`
}

// OrganizationUsersListResponse represents a paginated list of organization users
type OrganizationUsersListResponse struct {
	Users    []OrganizationUser `json:"users"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

// Scan implementation for Organization Settings (JSONB)
func (s *JSONB) ScanOrgSettings(value interface{}) error {
	if value == nil {
		*s = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		*s = make(map[string]interface{})
		return nil
	}

	result := make(map[string]interface{})
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}

	*s = result
	return nil
}

func (s JSONB) ValueOrgSettings() (driver.Value, error) {
	if s == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(s)
}
