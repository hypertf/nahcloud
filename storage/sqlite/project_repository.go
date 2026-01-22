package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

// ProjectRepository handles project data operations
type ProjectRepository struct {
	db *DB
}

// NewProjectRepository creates a new project repository
func NewProjectRepository(db *DB) *ProjectRepository {
	return &ProjectRepository{db: db}
}

// Create creates a new project
func (r *ProjectRepository) Create(project *domain.Project) error {
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now

	query := `INSERT INTO projects (id, org_id, slug, name, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`

	_, err := r.db.Exec(query, project.ID, project.OrgID, project.Slug, project.Name, project.CreatedAt, project.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: projects.org_id, projects.slug") {
			return domain.AlreadyExistsError("project", "slug", project.Slug)
		}
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return domain.ForeignKeyViolationError("organization", "id", project.OrgID)
		}
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// GetByID retrieves a project by ID
func (r *ProjectRepository) GetByID(id string) (*domain.Project, error) {
	project := &domain.Project{}
	query := `SELECT id, org_id, slug, name, created_at, updated_at FROM projects WHERE id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&project.ID,
		&project.OrgID,
		&project.Slug,
		&project.Name,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("project", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}

// GetBySlug retrieves a project by org ID and slug
func (r *ProjectRepository) GetBySlug(orgID, slug string) (*domain.Project, error) {
	project := &domain.Project{}
	query := `SELECT id, org_id, slug, name, created_at, updated_at FROM projects WHERE org_id = ? AND slug = ?`

	err := r.db.QueryRow(query, orgID, slug).Scan(
		&project.ID,
		&project.OrgID,
		&project.Slug,
		&project.Name,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("project", slug)
		}
		return nil, fmt.Errorf("failed to get project by slug: %w", err)
	}

	return project, nil
}

// GetByName retrieves a project by name (for backwards compatibility)
func (r *ProjectRepository) GetByName(name string) (*domain.Project, error) {
	project := &domain.Project{}
	query := `SELECT id, org_id, slug, name, created_at, updated_at FROM projects WHERE name = ?`

	err := r.db.QueryRow(query, name).Scan(
		&project.ID,
		&project.OrgID,
		&project.Slug,
		&project.Name,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("project", name)
		}
		return nil, fmt.Errorf("failed to get project by name: %w", err)
	}

	return project, nil
}

// List retrieves projects with optional filtering
func (r *ProjectRepository) List(opts domain.ProjectListOptions) ([]*domain.Project, error) {
	var projects []*domain.Project
	var args []interface{}

	query := `SELECT id, org_id, slug, name, created_at, updated_at FROM projects`
	var conditions []string

	if opts.OrgID != "" {
		conditions = append(conditions, "org_id = ?")
		args = append(args, opts.OrgID)
	}

	if opts.Slug != "" {
		conditions = append(conditions, "slug = ?")
		args = append(args, opts.Slug)
	}

	if opts.Name != "" {
		conditions = append(conditions, "name = ?")
		args = append(args, opts.Name)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		project := &domain.Project{}
		err := rows.Scan(
			&project.ID,
			&project.OrgID,
			&project.Slug,
			&project.Name,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating projects: %w", err)
	}

	return projects, nil
}

// Update updates an existing project
func (r *ProjectRepository) Update(id string, req domain.UpdateProjectRequest) (*domain.Project, error) {
	// First check if project exists
	existing, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	existing.UpdatedAt = time.Now()

	query := `UPDATE projects SET name = ?, updated_at = ? WHERE id = ?`

	_, err = r.db.Exec(query, existing.Name, existing.UpdatedAt, id)
	if err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	return existing, nil
}

// Delete deletes a project by ID
func (r *ProjectRepository) Delete(id string) error {
	// First check if project exists
	_, err := r.GetByID(id)
	if err != nil {
		return err
	}

	// Check if project has instances (enforced by FK constraint, but we want specific error)
	var instanceCount int
	err = r.db.QueryRow("SELECT COUNT(*) FROM instances WHERE project_id = ?", id).Scan(&instanceCount)
	if err != nil {
		return fmt.Errorf("failed to check project instances: %w", err)
	}

	if instanceCount > 0 {
		return domain.InvalidInputError("cannot delete project with existing instances", map[string]interface{}{
			"project_id":     id,
			"instance_count": instanceCount,
		})
	}

	// Check if project has buckets
	var bucketCount int
	err = r.db.QueryRow("SELECT COUNT(*) FROM buckets WHERE project_id = ?", id).Scan(&bucketCount)
	if err != nil {
		return fmt.Errorf("failed to check project buckets: %w", err)
	}

	if bucketCount > 0 {
		return domain.InvalidInputError("cannot delete project with existing buckets", map[string]interface{}{
			"project_id":   id,
			"bucket_count": bucketCount,
		})
	}

	query := `DELETE FROM projects WHERE id = ?`

	_, err = r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	return nil
}
