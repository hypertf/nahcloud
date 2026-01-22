package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

// OrganizationRepository handles organization data operations
type OrganizationRepository struct {
	db *DB
}

// NewOrganizationRepository creates a new organization repository
func NewOrganizationRepository(db *DB) *OrganizationRepository {
	return &OrganizationRepository{db: db}
}

// Create creates a new organization
func (r *OrganizationRepository) Create(org *domain.Organization) error {
	now := time.Now()
	org.CreatedAt = now
	org.UpdatedAt = now

	query := `INSERT INTO organizations (id, slug, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query, org.ID, org.Slug, org.Name, org.CreatedAt, org.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: organizations.slug") {
			return domain.AlreadyExistsError("organization", "slug", org.Slug)
		}
		if strings.Contains(err.Error(), "UNIQUE constraint failed: organizations.id") || strings.Contains(strings.ToLower(err.Error()), "primary key constraint failed") {
			return domain.AlreadyExistsError("organization", "id", org.ID)
		}
		return fmt.Errorf("failed to create organization: %w", err)
	}

	return nil
}

// GetByID retrieves an organization by ID
func (r *OrganizationRepository) GetByID(id string) (*domain.Organization, error) {
	org := &domain.Organization{}
	query := `SELECT id, slug, name, created_at, updated_at FROM organizations WHERE id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&org.ID,
		&org.Slug,
		&org.Name,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("organization", id)
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	return org, nil
}

// GetBySlug retrieves an organization by slug
func (r *OrganizationRepository) GetBySlug(slug string) (*domain.Organization, error) {
	org := &domain.Organization{}
	query := `SELECT id, slug, name, created_at, updated_at FROM organizations WHERE slug = ?`

	err := r.db.QueryRow(query, slug).Scan(
		&org.ID,
		&org.Slug,
		&org.Name,
		&org.CreatedAt,
		&org.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("organization", slug)
		}
		return nil, fmt.Errorf("failed to get organization by slug: %w", err)
	}

	return org, nil
}

// List retrieves organizations with optional filtering
func (r *OrganizationRepository) List(opts domain.OrganizationListOptions) ([]*domain.Organization, error) {
	var orgs []*domain.Organization
	var args []interface{}

	query := `SELECT id, slug, name, created_at, updated_at FROM organizations`
	var conditions []string

	if opts.Slug != "" {
		conditions = append(conditions, "slug = ?")
		args = append(args, opts.Slug)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		org := &domain.Organization{}
		err := rows.Scan(
			&org.ID,
			&org.Slug,
			&org.Name,
			&org.CreatedAt,
			&org.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan organization: %w", err)
		}
		orgs = append(orgs, org)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating organizations: %w", err)
	}

	return orgs, nil
}

// Update updates an existing organization
func (r *OrganizationRepository) Update(id string, req domain.UpdateOrganizationRequest) (*domain.Organization, error) {
	// First check if organization exists
	existing, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	existing.UpdatedAt = time.Now()

	query := `UPDATE organizations SET name = ?, updated_at = ? WHERE id = ?`

	_, err = r.db.Exec(query, existing.Name, existing.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update organization: %w", err)
	}

	return existing, nil
}

// Delete deletes an organization by ID
func (r *OrganizationRepository) Delete(id string) error {
	// First check if organization exists
	_, err := r.GetByID(id)
	if err != nil {
		return err
	}

	// Check if organization has projects
	var projectCount int
	err = r.db.QueryRow("SELECT COUNT(*) FROM projects WHERE org_id = ?", id).Scan(&projectCount)
	if err != nil {
		return fmt.Errorf("failed to check organization projects: %w", err)
	}

	if projectCount > 0 {
		return domain.InvalidInputError("cannot delete organization with existing projects", map[string]interface{}{
			"org_id":        id,
			"project_count": projectCount,
		})
	}

	query := `DELETE FROM organizations WHERE id = ?`

	_, err = r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete organization: %w", err)
	}

	return nil
}
