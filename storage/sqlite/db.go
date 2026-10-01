package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultDSN = "file:nah.db?_busy_timeout=5000&_fk=1"
)

// DB wraps the SQLite database connection
type DB struct {
	*sql.DB
}

// NewDB creates a new SQLite database connection and initializes the schema
func NewDB(dsn string) (*DB, error) {
	if dsn == "" {
		dsn = defaultDSN
	}
	// Foreign-key enforcement is connection-local in SQLite. Put it in the
	// driver DSN so replacement connections cannot silently disable cascades.
	separator := "?"
	if strings.Contains(dsn, "?") {
		separator = "&"
	}
	dsn += separator + "_foreign_keys=on&_busy_timeout=5000"

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	// SQLite settings such as foreign_keys are connection-local. A single
	// shared connection also avoids lock churn for this intentionally small,
	// low-latency test cloud.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Set pragmas
	pragmas := []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA busy_timeout = 5000",
	}

	for _, pragma := range pragmas {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to set pragma %s: %w", pragma, err)
		}
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	sqliteDB := &DB{DB: db}

	// Initialize schema
	if err := sqliteDB.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return sqliteDB, nil
}

// initSchema creates the necessary tables if they don't exist
func (db *DB) initSchema() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS organizations (
			id TEXT PRIMARY KEY,
			slug TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			token_hash TEXT UNIQUE NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			last_used_at DATETIME,
			FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS projects (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			slug TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE,
			UNIQUE(org_id, slug)
		)`,
		`CREATE TABLE IF NOT EXISTS instances (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			region TEXT NOT NULL DEFAULT 'us-east-1',
			cpu INTEGER NOT NULL,
			memory_mb INTEGER NOT NULL,
			image TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'running',
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			UNIQUE(project_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS metadata (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			path TEXT NOT NULL,
			value TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE,
			UNIQUE(org_id, path)
		)`,
		`CREATE TABLE IF NOT EXISTS buckets (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			UNIQUE(project_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS objects (
			id TEXT PRIMARY KEY,
			bucket_id TEXT NOT NULL,
			path TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (bucket_id) REFERENCES buckets(id) ON DELETE CASCADE,
			UNIQUE(bucket_id, path)
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			org_id TEXT NOT NULL,
			token_hash TEXT UNIQUE NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS networks (
			id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, cidr TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE, UNIQUE(project_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS subnets (
			id TEXT PRIMARY KEY, network_id TEXT NOT NULL, project_id TEXT NOT NULL, name TEXT NOT NULL, cidr TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (network_id) REFERENCES networks(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE, UNIQUE(network_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS disks (
			id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, size_gb INTEGER NOT NULL CHECK(size_gb > 0),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE, UNIQUE(project_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS disk_attachments (
			id TEXT PRIMARY KEY, disk_id TEXT NOT NULL, project_id TEXT NOT NULL, instance_id TEXT NOT NULL, device TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (disk_id) REFERENCES disks(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE,
			UNIQUE(disk_id, instance_id), UNIQUE(instance_id, device)
		)`,
		`CREATE TABLE IF NOT EXISTS policies (
			id TEXT PRIMARY KEY, org_id TEXT NOT NULL, project_id TEXT, name TEXT NOT NULL, document TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (org_id) REFERENCES organizations(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS policies_org_name ON policies(org_id, name) WHERE project_id IS NULL`,
		`CREATE UNIQUE INDEX IF NOT EXISTS policies_project_name ON policies(project_id, name) WHERE project_id IS NOT NULL`,
		`CREATE TABLE IF NOT EXISTS policy_bindings (
			id TEXT PRIMARY KEY, policy_id TEXT NOT NULL, principal TEXT NOT NULL, role TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (policy_id) REFERENCES policies(id) ON DELETE CASCADE, UNIQUE(policy_id, principal, role)
		)`,
		`CREATE TABLE IF NOT EXISTS load_balancers (
			id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, protocol TEXT NOT NULL, port INTEGER NOT NULL CHECK(port BETWEEN 1 AND 65535),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE, UNIQUE(project_id, name)
		)`,
		`CREATE TABLE IF NOT EXISTS load_balancer_backends (
			id TEXT PRIMARY KEY, load_balancer_id TEXT NOT NULL, project_id TEXT NOT NULL, instance_id TEXT NOT NULL,
			port INTEGER NOT NULL CHECK(port BETWEEN 1 AND 65535), weight INTEGER NOT NULL CHECK(weight BETWEEN 1 AND 100),
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (load_balancer_id) REFERENCES load_balancers(id) ON DELETE CASCADE,
			FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
			FOREIGN KEY (instance_id) REFERENCES instances(id) ON DELETE CASCADE,
			UNIQUE(load_balancer_id, instance_id, port)
		)`,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Run migrations for existing databases
	if err := db.runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// runMigrations applies schema migrations for existing databases
// TODO: Implement a proper migration system (e.g., golang-migrate)
func (db *DB) runMigrations() error {
	// Check if we need to migrate from old schema (no organizations)
	if err := db.migrateToOrganizations(); err != nil {
		return fmt.Errorf("failed to migrate to organizations: %w", err)
	}

	return nil
}

// migrateToOrganizations migrates existing data to the new organization-based schema
func (db *DB) migrateToOrganizations() error {
	// Check if there's data to migrate by looking at old projects table structure
	// We need to check if old projects exist (ones without org_id)
	var hasOldProjects bool
	rows, err := db.Query("PRAGMA table_info(projects)")
	if err != nil {
		return fmt.Errorf("failed to get projects table info: %w", err)
	}

	hasOrgID := false
	for rows.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan table info: %w", err)
		}
		if name == "org_id" {
			hasOrgID = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("failed to inspect projects table: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("failed to close projects schema query: %w", err)
	}

	// Check if there are any existing projects
	var projectCount int
	err = db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&projectCount)
	if err != nil {
		// Table might have just been created with new schema
		return nil
	}

	hasOldProjects = !hasOrgID && projectCount > 0

	if !hasOldProjects {
		// No migration needed - orgs will be auto-created when users visit
		return nil
	}

	log.Println("Migrating to organization-based schema...")

	// Reuse an existing organization when a partial migration already created
	// one; otherwise create the legacy default organization.
	var defaultOrgID string
	err = db.QueryRow(`SELECT id FROM organizations ORDER BY created_at LIMIT 1`).Scan(&defaultOrgID)
	if err == sql.ErrNoRows {
		defaultOrgID = "default-org"
		_, err = db.Exec(`INSERT INTO organizations (id, slug, name) VALUES (?, ?, ?)`,
			defaultOrgID, "default-org", "Default Organization")
		if err != nil {
			return fmt.Errorf("failed to create default organization: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to find migration organization: %w", err)
	}

	// Migrate projects: add org_id and slug columns if they don't exist
	// First, check current columns
	if !hasOrgID {
		// Add org_id column
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN org_id TEXT`)
		if err != nil && err.Error() != "duplicate column name: org_id" {
			return fmt.Errorf("failed to add org_id column: %w", err)
		}

		// Add slug column
		_, err = db.Exec(`ALTER TABLE projects ADD COLUMN slug TEXT`)
		if err != nil && err.Error() != "duplicate column name: slug" {
			return fmt.Errorf("failed to add slug column: %w", err)
		}

		// Update existing projects
		_, err = db.Exec(`UPDATE projects SET org_id = ?, slug = name WHERE org_id IS NULL`, defaultOrgID)
		if err != nil {
			return fmt.Errorf("failed to update projects with org_id: %w", err)
		}
	}

	// Migrate buckets: add project_id if not present
	var hasBucketProjectID bool
	rows2, err := db.Query("PRAGMA table_info(buckets)")
	if err != nil {
		return fmt.Errorf("failed to get buckets table info: %w", err)
	}

	for rows2.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows2.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan table info: %w", err)
		}
		if name == "project_id" {
			hasBucketProjectID = true
			break
		}
	}
	if err := rows2.Err(); err != nil {
		rows2.Close()
		return fmt.Errorf("failed to inspect buckets table: %w", err)
	}
	if err := rows2.Close(); err != nil {
		return fmt.Errorf("failed to close buckets schema query: %w", err)
	}

	if !hasBucketProjectID {
		// Add project_id column
		_, err = db.Exec(`ALTER TABLE buckets ADD COLUMN project_id TEXT`)
		if err != nil && err.Error() != "duplicate column name: project_id" {
			return fmt.Errorf("failed to add project_id to buckets: %w", err)
		}

		// Get or create a default project to assign buckets to
		var defaultProjectID string
		err = db.QueryRow(`SELECT id FROM projects LIMIT 1`).Scan(&defaultProjectID)
		if err == sql.ErrNoRows {
			// Create a default project
			defaultProjectID = "default-project"
			_, err = db.Exec(`INSERT INTO projects (id, org_id, slug, name) VALUES (?, ?, ?, ?)`,
				defaultProjectID, defaultOrgID, "default-project", "Default Project")
			if err != nil {
				return fmt.Errorf("failed to create default project: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("failed to get default project: %w", err)
		}

		// Update existing buckets
		_, err = db.Exec(`UPDATE buckets SET project_id = ? WHERE project_id IS NULL`, defaultProjectID)
		if err != nil {
			return fmt.Errorf("failed to update buckets with project_id: %w", err)
		}
	}

	// Migrate metadata: add org_id if not present
	var hasMetadataOrgID bool
	rows3, err := db.Query("PRAGMA table_info(metadata)")
	if err != nil {
		return fmt.Errorf("failed to get metadata table info: %w", err)
	}

	for rows3.Next() {
		var cid int
		var name, dtype string
		var notnull, pk int
		var dfltValue sql.NullString
		if err := rows3.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			return fmt.Errorf("failed to scan table info: %w", err)
		}
		if name == "org_id" {
			hasMetadataOrgID = true
			break
		}
	}
	if err := rows3.Err(); err != nil {
		rows3.Close()
		return fmt.Errorf("failed to inspect metadata table: %w", err)
	}
	if err := rows3.Close(); err != nil {
		return fmt.Errorf("failed to close metadata schema query: %w", err)
	}

	if !hasMetadataOrgID {
		// Add org_id column
		_, err = db.Exec(`ALTER TABLE metadata ADD COLUMN org_id TEXT`)
		if err != nil && err.Error() != "duplicate column name: org_id" {
			return fmt.Errorf("failed to add org_id to metadata: %w", err)
		}

		// Update existing metadata
		_, err = db.Exec(`UPDATE metadata SET org_id = ? WHERE org_id IS NULL`, defaultOrgID)
		if err != nil {
			return fmt.Errorf("failed to update metadata with org_id: %w", err)
		}
	}

	log.Println("Migration to organization-based schema completed successfully")
	return nil
}

// ensureDefaultOrg creates a default organization if none exists
func (db *DB) ensureDefaultOrg() error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM organizations").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err = db.Exec(`INSERT INTO organizations (id, slug, name) VALUES (?, ?, ?)`,
			"default-org", "default-org", "Default Organization")
		if err != nil {
			return fmt.Errorf("failed to create default organization: %w", err)
		}
		log.Println("Created default organization")
	}

	return nil
}
