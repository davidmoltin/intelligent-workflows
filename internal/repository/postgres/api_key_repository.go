package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/davidmoltin/intelligent-workflows/internal/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

// APIKeyRepository handles API key database operations
type APIKeyRepository struct {
	db *sql.DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *sql.DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create creates a new API key
func (r *APIKeyRepository) Create(ctx context.Context, apiKey *models.APIKey) error {
	query := `
		INSERT INTO api_keys (
			id, key_hash, key_prefix, name, user_id, scopes,
			is_active, expires_at, created_at, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	apiKey.ID = uuid.New()
	apiKey.CreatedAt = time.Now()

	err := r.db.QueryRowContext(
		ctx, query,
		apiKey.ID, apiKey.KeyHash, apiKey.KeyPrefix, apiKey.Name,
		apiKey.UserID, pq.Array(apiKey.Scopes), apiKey.IsActive,
		apiKey.ExpiresAt, apiKey.CreatedAt, apiKey.CreatedBy,
	).Scan(&apiKey.ID, &apiKey.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetByHash retrieves an API key by its hash
func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*models.APIKey, error) {
	apiKey := &models.APIKey{}
	query := `
		SELECT id, key_hash, key_prefix, name, user_id, scopes,
		       is_active, last_used_at, expires_at, created_at, created_by
		FROM api_keys
		WHERE key_hash = $1`

	var scopes pq.StringArray
	err := r.db.QueryRowContext(ctx, query, keyHash).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Name,
		&apiKey.UserID, &scopes, &apiKey.IsActive,
		&apiKey.LastUsedAt, &apiKey.ExpiresAt, &apiKey.CreatedAt, &apiKey.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	apiKey.Scopes = scopes
	return apiKey, nil
}

// GetByID retrieves an API key by ID
func (r *APIKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.APIKey, error) {
	apiKey := &models.APIKey{}
	query := `
		SELECT id, key_hash, key_prefix, name, user_id, scopes,
		       is_active, last_used_at, expires_at, created_at, created_by
		FROM api_keys
		WHERE id = $1`

	var scopes pq.StringArray
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&apiKey.ID, &apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Name,
		&apiKey.UserID, &scopes, &apiKey.IsActive,
		&apiKey.LastUsedAt, &apiKey.ExpiresAt, &apiKey.CreatedAt, &apiKey.CreatedBy,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("API key not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	apiKey.Scopes = scopes
	return apiKey, nil
}

// ListByUser retrieves all API keys for a user
func (r *APIKeyRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*models.APIKey, error) {
	query := `
		SELECT id, key_hash, key_prefix, name, user_id, scopes,
		       is_active, last_used_at, expires_at, created_at, created_by
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []*models.APIKey
	for rows.Next() {
		apiKey := &models.APIKey{}
		var scopes pq.StringArray
		err := rows.Scan(
			&apiKey.ID, &apiKey.KeyHash, &apiKey.KeyPrefix, &apiKey.Name,
			&apiKey.UserID, &scopes, &apiKey.IsActive,
			&apiKey.LastUsedAt, &apiKey.ExpiresAt, &apiKey.CreatedAt, &apiKey.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKey.Scopes = scopes
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

// UpdateLastUsed updates the last used timestamp
func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE api_keys
		SET last_used_at = $1
		WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// Revoke revokes an API key (sets is_active to false)
func (r *APIKeyRepository) Revoke(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE api_keys
		SET is_active = false
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to revoke API key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// Delete deletes an API key
func (r *APIKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM api_keys WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// RefreshTokenRepository handles refresh token database operations
type RefreshTokenRepository struct {
	db *sql.DB
}

// NewRefreshTokenRepository creates a new refresh token repository
func NewRefreshTokenRepository(db *sql.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

// Create creates a new refresh token
func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, token_hash, user_id, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at`

	token.ID = uuid.New()
	token.CreatedAt = time.Now()

	err := r.db.QueryRowContext(
		ctx, query,
		token.ID, token.TokenHash, token.UserID, token.ExpiresAt, token.CreatedAt,
	).Scan(&token.ID, &token.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create refresh token: %w", err)
	}

	return nil
}

// GetByHash retrieves a refresh token by its hash
func (r *RefreshTokenRepository) GetByHash(ctx context.Context, tokenHash string) (*models.RefreshToken, error) {
	token := &models.RefreshToken{}
	query := `
		SELECT id, token_hash, user_id, expires_at, created_at, revoked_at
		FROM refresh_tokens
		WHERE token_hash = $1`

	err := r.db.QueryRowContext(ctx, query, tokenHash).Scan(
		&token.ID, &token.TokenHash, &token.UserID,
		&token.ExpiresAt, &token.CreatedAt, &token.RevokedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("refresh token not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return token, nil
}

// Revoke revokes a refresh token
func (r *RefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE token_hash = $2`

	result, err := r.db.ExecContext(ctx, query, time.Now(), tokenHash)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("refresh token not found")
	}

	return nil
}

// RevokeAllForUser revokes all refresh tokens for a user
func (r *RefreshTokenRepository) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `
		UPDATE refresh_tokens
		SET revoked_at = $1
		WHERE user_id = $2 AND revoked_at IS NULL`

	_, err := r.db.ExecContext(ctx, query, time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to revoke refresh tokens: %w", err)
	}

	return nil
}

// DeleteExpired deletes expired refresh tokens
func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context) error {
	query := `DELETE FROM refresh_tokens WHERE expires_at < $1`

	_, err := r.db.ExecContext(ctx, query, time.Now())
	if err != nil {
		return fmt.Errorf("failed to delete expired tokens: %w", err)
	}

	return nil
}
