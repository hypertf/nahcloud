package sqlite

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

// APIKeyRepository handles API key data operations
type APIKeyRepository struct {
	db *DB
}

// NewAPIKeyRepository creates a new API key repository
func NewAPIKeyRepository(db *DB) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

// Create creates a new API key
func (r *APIKeyRepository) Create(key *domain.APIKey) error {
	key.CreatedAt = time.Now()

	query := `INSERT INTO api_keys (id, org_id, name, token_hash, created_at) VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query, key.ID, key.OrgID, key.Name, key.TokenHash, key.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create API key: %w", err)
	}

	return nil
}

// GetByID retrieves an API key by ID
func (r *APIKeyRepository) GetByID(id string) (*domain.APIKey, error) {
	key := &domain.APIKey{}
	query := `SELECT id, org_id, name, token_hash, created_at, last_used_at FROM api_keys WHERE id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&key.ID,
		&key.OrgID,
		&key.Name,
		&key.TokenHash,
		&key.CreatedAt,
		&key.LastUsedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("api_key", id)
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	return key, nil
}

// GetByTokenHash retrieves an API key by token hash
func (r *APIKeyRepository) GetByTokenHash(tokenHash string) (*domain.APIKey, error) {
	key := &domain.APIKey{}
	query := `SELECT id, org_id, name, token_hash, created_at, last_used_at FROM api_keys WHERE token_hash = ?`

	err := r.db.QueryRow(query, tokenHash).Scan(
		&key.ID,
		&key.OrgID,
		&key.Name,
		&key.TokenHash,
		&key.CreatedAt,
		&key.LastUsedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("api_key", "token")
		}
		return nil, fmt.Errorf("failed to get API key by token: %w", err)
	}

	return key, nil
}

// ListByOrgID retrieves all API keys for an organization
func (r *APIKeyRepository) ListByOrgID(orgID string) ([]*domain.APIKey, error) {
	query := `SELECT id, org_id, name, token_hash, created_at, last_used_at FROM api_keys WHERE org_id = ? ORDER BY created_at DESC`

	rows, err := r.db.Query(query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var keys []*domain.APIKey
	for rows.Next() {
		key := &domain.APIKey{}
		err := rows.Scan(
			&key.ID,
			&key.OrgID,
			&key.Name,
			&key.TokenHash,
			&key.CreatedAt,
			&key.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		keys = append(keys, key)
	}

	return keys, nil
}

// UpdateLastUsed updates the last_used_at timestamp
func (r *APIKeyRepository) UpdateLastUsed(id string) error {
	query := `UPDATE api_keys SET last_used_at = ? WHERE id = ?`
	_, err := r.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update last_used_at: %w", err)
	}
	return nil
}

// Delete deletes an API key by ID
func (r *APIKeyRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return domain.NotFoundError("api_key", id)
	}

	return nil
}
