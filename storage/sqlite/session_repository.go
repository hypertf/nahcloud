package sqlite

import (
	"database/sql"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

// SessionRepository implements session data operations for SQLite
type SessionRepository struct {
	db *DB
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *DB) *SessionRepository {
	return &SessionRepository{db: db}
}

// Create creates a new session
func (r *SessionRepository) Create(session *domain.Session) error {
	_, err := r.db.Exec(`
		INSERT INTO sessions (id, org_id, token_hash, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?)`,
		session.ID, session.OrgID, session.TokenHash, session.CreatedAt, session.ExpiresAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: sessions.token_hash") {
			return domain.AlreadyExistsError("session", "token_hash", session.TokenHash)
		}
		return err
	}
	return nil
}

// GetByTokenHash retrieves a session by its token hash
func (r *SessionRepository) GetByTokenHash(tokenHash string) (*domain.Session, error) {
	var session domain.Session
	err := r.db.QueryRow(`
		SELECT id, org_id, token_hash, created_at, expires_at
		FROM sessions WHERE token_hash = ?`, tokenHash).Scan(
		&session.ID, &session.OrgID, &session.TokenHash, &session.CreatedAt, &session.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, domain.NotFoundError("session", tokenHash)
	}
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// Delete deletes a session by ID
func (r *SessionRepository) Delete(id string) error {
	result, err := r.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.NotFoundError("session", id)
	}
	return nil
}

// DeleteExpired deletes all expired sessions
func (r *SessionRepository) DeleteExpired() error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	return err
}
