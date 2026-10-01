package sqlite

import (
	"database/sql"
	"fmt"
	"time"
)

const latestSchemaVersion = 3

type migration struct {
	version int
	apply   func(*sql.Tx) error
}

func (db *DB) runMigrations() error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME NOT NULL
	)`); err != nil {
		return err
	}
	var current int
	if err := db.QueryRow(`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return err
	}
	if current > latestSchemaVersion {
		return fmt.Errorf("database schema version %d is newer than supported version %d", current, latestSchemaVersion)
	}
	for _, item := range []migration{{2, migrateV2}, {3, migrateV3}} {
		if current >= item.version {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err = item.apply(tx); err == nil {
			_, err = tx.Exec(`INSERT INTO schema_migrations(version, applied_at) VALUES(?, ?)`, item.version, time.Now().UTC())
		}
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("apply schema migration %d: %w", item.version, err)
		}
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("commit schema migration %d: %w", item.version, err)
		}
		current = item.version
	}
	return nil
}

func migrateV2(tx *sql.Tx) error {
	for _, statement := range baseSchema {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	projectsExist, err := tableExists(tx, "projects")
	if err != nil || !projectsExist {
		return err
	}
	legacy := false
	for _, item := range []struct{ table, column, definition string }{
		{"instances", "region", `TEXT NOT NULL DEFAULT 'us-east-1'`},
		{"projects", "org_id", `TEXT`}, {"projects", "slug", `TEXT`},
		{"buckets", "project_id", `TEXT`}, {"metadata", "org_id", `TEXT`},
	} {
		has, err := columnExists(tx, item.table, item.column)
		if err != nil {
			return err
		}
		if !has {
			legacy = true
			if _, err := tx.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, item.table, item.column, item.definition)); err != nil {
				return err
			}
		}
	}
	if !legacy {
		return nil
	}
	var orgID string
	err = tx.QueryRow(`SELECT id FROM organizations ORDER BY created_at LIMIT 1`).Scan(&orgID)
	if err == sql.ErrNoRows {
		orgID = "default-org"
		if _, err = tx.Exec(`INSERT INTO organizations(id,slug,name) VALUES(?,?,?)`, orgID, orgID, "Default Organization"); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE projects SET org_id=?, slug=name WHERE org_id IS NULL OR slug IS NULL`, orgID); err != nil {
		return err
	}
	var projectID string
	err = tx.QueryRow(`SELECT id FROM projects ORDER BY created_at LIMIT 1`).Scan(&projectID)
	if err == sql.ErrNoRows {
		projectID = "default-project"
		if _, err = tx.Exec(`INSERT INTO projects(id,name,org_id,slug) VALUES(?,?,?,?)`, projectID, "Default Project", orgID, projectID); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE buckets SET project_id=? WHERE project_id IS NULL`, projectID); err != nil {
		return err
	}
	_, err = tx.Exec(`UPDATE metadata SET org_id=? WHERE org_id IS NULL`, orgID)
	return err
}

func migrateV3(tx *sql.Tx) error {
	hasSubnet, err := columnExists(tx, "instances", "subnet_id")
	if err != nil {
		return err
	}
	if !hasSubnet {
		if _, err = tx.Exec(`ALTER TABLE instances ADD COLUMN subnet_id TEXT REFERENCES subnets(id) ON DELETE RESTRICT`); err != nil {
			return err
		}
	}
	for _, statement := range graphSchema {
		if _, err := tx.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func tableExists(tx *sql.Tx, table string) (bool, error) {
	var count int
	err := tx.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&count)
	return count == 1, err
}

func columnExists(tx *sql.Tx, table, column string) (bool, error) {
	rows, err := tx.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, dataType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

var baseSchema = []string{
	`CREATE TABLE IF NOT EXISTS organizations (id TEXT PRIMARY KEY, slug TEXT UNIQUE NOT NULL, name TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
	`CREATE TABLE IF NOT EXISTS api_keys (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, name TEXT NOT NULL DEFAULT '', token_hash TEXT UNIQUE NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, last_used_at DATETIME, FOREIGN KEY(org_id) REFERENCES organizations(id) ON DELETE CASCADE)`,
	`CREATE TABLE IF NOT EXISTS projects (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, slug TEXT NOT NULL, name TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(org_id) REFERENCES organizations(id) ON DELETE CASCADE, UNIQUE(org_id,slug))`,
	`CREATE TABLE IF NOT EXISTS instances (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, region TEXT NOT NULL DEFAULT 'us-east-1', cpu INTEGER NOT NULL, memory_mb INTEGER NOT NULL, image TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'running', created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE, UNIQUE(project_id,name))`,
	`CREATE TABLE IF NOT EXISTS metadata (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, path TEXT NOT NULL, value TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(org_id) REFERENCES organizations(id) ON DELETE CASCADE, UNIQUE(org_id,path))`,
	`CREATE TABLE IF NOT EXISTS buckets (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE, UNIQUE(project_id,name))`,
	`CREATE TABLE IF NOT EXISTS objects (id TEXT PRIMARY KEY, bucket_id TEXT NOT NULL, path TEXT NOT NULL, content TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(bucket_id) REFERENCES buckets(id) ON DELETE CASCADE, UNIQUE(bucket_id,path))`,
	`CREATE TABLE IF NOT EXISTS sessions (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, token_hash TEXT UNIQUE NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, expires_at DATETIME NOT NULL, FOREIGN KEY(org_id) REFERENCES organizations(id) ON DELETE CASCADE)`,
}

var graphSchema = []string{
	`CREATE TABLE networks (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, region TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE RESTRICT, UNIQUE(project_id,name))`,
	`CREATE TABLE subnets (id TEXT PRIMARY KEY, network_id TEXT NOT NULL, project_id TEXT NOT NULL, name TEXT NOT NULL, cidr TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(network_id) REFERENCES networks(id) ON DELETE RESTRICT, FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE RESTRICT, UNIQUE(network_id,name))`,
	`CREATE TABLE disks (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, region TEXT NOT NULL, type TEXT NOT NULL CHECK(type IN ('standard','ssd')), size_gb INTEGER NOT NULL CHECK(size_gb BETWEEN 1 AND 16384), created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE RESTRICT, UNIQUE(project_id,name))`,
	`CREATE TABLE disk_attachments (id TEXT PRIMARY KEY, disk_id TEXT NOT NULL UNIQUE, project_id TEXT NOT NULL, instance_id TEXT NOT NULL, device TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(disk_id) REFERENCES disks(id) ON DELETE RESTRICT, FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE, FOREIGN KEY(instance_id) REFERENCES instances(id) ON DELETE CASCADE, UNIQUE(instance_id,device))`,
	`CREATE TABLE policies (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', effect TEXT NOT NULL CHECK(effect IN ('allow','deny')), actions TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(org_id) REFERENCES organizations(id) ON DELETE CASCADE, UNIQUE(org_id,name))`,
	`CREATE TABLE policy_bindings (id TEXT PRIMARY KEY, org_id TEXT NOT NULL, policy_id TEXT NOT NULL, principal_type TEXT NOT NULL, principal_id TEXT NOT NULL, target_type TEXT NOT NULL, target_id TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(org_id) REFERENCES organizations(id) ON DELETE CASCADE, FOREIGN KEY(policy_id) REFERENCES policies(id) ON DELETE CASCADE, UNIQUE(policy_id,principal_type,principal_id,target_type,target_id))`,
	`CREATE TABLE load_balancers (id TEXT PRIMARY KEY, project_id TEXT NOT NULL, name TEXT NOT NULL, subnet_id TEXT NOT NULL, region TEXT NOT NULL, protocol TEXT NOT NULL CHECK(protocol IN ('http','tcp')), port INTEGER NOT NULL CHECK(port BETWEEN 1 AND 65535), algorithm TEXT NOT NULL CHECK(algorithm IN ('round_robin','least_connections')), health_check_path TEXT NOT NULL DEFAULT '', status TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE RESTRICT, FOREIGN KEY(subnet_id) REFERENCES subnets(id) ON DELETE RESTRICT, UNIQUE(project_id,name))`,
	`CREATE TABLE load_balancer_backends (id TEXT PRIMARY KEY, load_balancer_id TEXT NOT NULL, instance_id TEXT NOT NULL, port INTEGER NOT NULL CHECK(port BETWEEN 1 AND 65535), weight INTEGER NOT NULL CHECK(weight BETWEEN 1 AND 100), enabled INTEGER NOT NULL CHECK(enabled IN (0,1)), created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, FOREIGN KEY(load_balancer_id) REFERENCES load_balancers(id) ON DELETE CASCADE, FOREIGN KEY(instance_id) REFERENCES instances(id) ON DELETE CASCADE, UNIQUE(load_balancer_id,instance_id,port))`,
	`CREATE INDEX graph_bindings_principal ON policy_bindings(org_id,principal_type,principal_id)`,
	`CREATE INDEX graph_bindings_target ON policy_bindings(org_id,target_type,target_id)`,
	`CREATE INDEX graph_subnets_scope ON subnets(project_id,network_id)`,
}
