package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

// InstanceRepository handles instance data operations
type InstanceRepository struct {
	db *DB
}

// NewInstanceRepository creates a new instance repository
func NewInstanceRepository(db *DB) *InstanceRepository {
	return &InstanceRepository{db: db}
}

// Create creates a new instance
func (r *InstanceRepository) Create(instance *domain.Instance) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err = checkProjectNodeLimit(tx, "instances", instance.ProjectID); err != nil {
		return err
	}
	if instance.SubnetID != nil {
		var project, region string
		err = tx.QueryRow(`SELECT s.project_id,n.region FROM subnets s JOIN networks n ON n.id=s.network_id WHERE s.id=?`, *instance.SubnetID).Scan(&project, &region)
		if err != nil || project != instance.ProjectID || region != instance.Region {
			return domain.NotFoundError("subnet", *instance.SubnetID)
		}
	}
	now := time.Now()
	instance.CreatedAt = now
	instance.UpdatedAt = now

	query := `INSERT INTO instances (id, project_id, name, region, cpu, memory_mb, image, status, subnet_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(query, instance.ID, instance.ProjectID, instance.Name, instance.Region, instance.CPU, instance.MemoryMB, instance.Image, instance.Status, instance.SubnetID, instance.CreatedAt, instance.UpdatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: instances.project_id, instances.name") {
			return domain.AlreadyExistsError("instance", "name", instance.Name)
		}
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return domain.ForeignKeyViolationError("project", "id", instance.ProjectID)
		}
		return fmt.Errorf("failed to create instance: %w", err)
	}

	return tx.Commit()
}

// GetByID retrieves an instance by ID
func (r *InstanceRepository) GetByID(id string) (*domain.Instance, error) {
	instance := &domain.Instance{}
	query := `SELECT id, project_id, name, region, cpu, memory_mb, image, status, subnet_id, created_at, updated_at FROM instances WHERE id = ?`

	err := r.db.QueryRow(query, id).Scan(
		&instance.ID,
		&instance.ProjectID,
		&instance.Name,
		&instance.Region,
		&instance.CPU,
		&instance.MemoryMB,
		&instance.Image,
		&instance.Status,
		&instance.SubnetID,
		&instance.CreatedAt,
		&instance.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.NotFoundError("instance", id)
		}
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	return instance, nil
}

// List retrieves instances with optional filtering
func (r *InstanceRepository) List(opts domain.InstanceListOptions) ([]*domain.Instance, error) {
	var instances []*domain.Instance
	var args []interface{}

	query := `SELECT id, project_id, name, region, cpu, memory_mb, image, status, subnet_id, created_at, updated_at FROM instances`
	var conditions []string

	if opts.ProjectID != "" {
		conditions = append(conditions, "project_id = ?")
		args = append(args, opts.ProjectID)
	}

	if opts.Name != "" {
		conditions = append(conditions, "name = ?")
		args = append(args, opts.Name)
	}

	if opts.Region != "" {
		conditions = append(conditions, "region = ?")
		args = append(args, opts.Region)
	}

	if opts.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, opts.Status)
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY name"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		instance := &domain.Instance{}
		err := rows.Scan(
			&instance.ID,
			&instance.ProjectID,
			&instance.Name,
			&instance.Region,
			&instance.CPU,
			&instance.MemoryMB,
			&instance.Image,
			&instance.Status,
			&instance.SubnetID,
			&instance.CreatedAt,
			&instance.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}
		instances = append(instances, instance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating instances: %w", err)
	}

	return instances, nil
}

// Update updates an existing instance
func (r *InstanceRepository) Update(id string, req domain.UpdateInstanceRequest) (*domain.Instance, error) {
	// First check if instance exists
	existing, err := r.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Update fields that are provided
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.CPU != nil {
		existing.CPU = *req.CPU
	}
	if req.MemoryMB != nil {
		existing.MemoryMB = *req.MemoryMB
	}
	if req.Image != nil {
		existing.Image = *req.Image
	}
	if req.Status != nil {
		existing.Status = *req.Status
	}
	existing.UpdatedAt = time.Now()

	query := `UPDATE instances SET name = ?, cpu = ?, memory_mb = ?, image = ?, status = ?, updated_at = ? WHERE id = ?`

	_, err = r.db.Exec(query, existing.Name, existing.CPU, existing.MemoryMB, existing.Image, existing.Status, existing.UpdatedAt, id)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: instances.project_id, instances.name") {
			return nil, domain.AlreadyExistsError("instance", "name", existing.Name)
		}
		return nil, fmt.Errorf("failed to update instance: %w", err)
	}

	return existing, nil
}

// Delete deletes an instance by ID
func (r *InstanceRepository) Delete(id string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var exists string
	if err = tx.QueryRow(`SELECT id FROM instances WHERE id=?`, id).Scan(&exists); err == sql.ErrNoRows {
		return domain.NotFoundError("instance", id)
	} else if err != nil {
		return err
	}
	if _, err = tx.Exec(`DELETE FROM policy_bindings WHERE target_type='instance' AND target_id=?`, id); err != nil {
		return err
	}
	_, err = tx.Exec(`DELETE FROM instances WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	return tx.Commit()
}
