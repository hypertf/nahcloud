package sqlite

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
)

func rawDB(t *testing.T, path string) *sql.DB {
	t.Helper()
	db, e := sql.Open("sqlite3", path)
	require.NoError(t, e)
	return db
}
func execAll(t *testing.T, db *sql.DB, statements []string) {
	t.Helper()
	for _, statement := range statements {
		_, e := db.Exec(statement)
		require.NoError(t, e, statement)
	}
}
func assertLatest(t *testing.T, path string) *DB {
	t.Helper()
	db, e := NewDB(path)
	require.NoError(t, e)
	var version int
	require.NoError(t, db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version))
	require.Equal(t, latestSchemaVersion, version)
	var subnetColumn int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('instances') WHERE name='subnet_id'`).Scan(&subnetColumn))
	require.Equal(t, 1, subnetColumn)
	return db
}

func TestMigrateFreshDatabase(t *testing.T) {
	path := t.TempDir() + "/fresh.db"
	db := assertLatest(t, path)
	defer db.Close()
	for _, table := range []string{"organizations", "instances", "networks", "policy_bindings", "load_balancer_backends"} {
		var found string
		require.NoError(t, db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found))
	}
}

func v01Schema() []string {
	return []string{
		`CREATE TABLE projects(id TEXT PRIMARY KEY,name TEXT UNIQUE NOT NULL,created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE instances(id TEXT PRIMARY KEY,project_id TEXT NOT NULL,name TEXT NOT NULL,region TEXT NOT NULL DEFAULT 'us-east-1',cpu INTEGER NOT NULL,memory_mb INTEGER NOT NULL,image TEXT NOT NULL,status TEXT NOT NULL DEFAULT 'running',created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE,UNIQUE(project_id,name))`,
		`CREATE TABLE metadata(id TEXT PRIMARY KEY,path TEXT NOT NULL UNIQUE,value TEXT NOT NULL,created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE buckets(id TEXT PRIMARY KEY,name TEXT UNIQUE NOT NULL,created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`CREATE TABLE objects(id TEXT PRIMARY KEY,bucket_id TEXT NOT NULL,path TEXT NOT NULL,content TEXT NOT NULL,created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,FOREIGN KEY(bucket_id) REFERENCES buckets(id) ON DELETE CASCADE,UNIQUE(bucket_id,path))`,
		`INSERT INTO projects(id,name) VALUES('legacy-project','legacy')`,
		`INSERT INTO instances(id,project_id,name,cpu,memory_mb,image) VALUES('legacy-instance','legacy-project','vm',1,512,'image')`,
		`INSERT INTO metadata(id,path,value) VALUES('legacy-metadata','key','value')`,
		`INSERT INTO buckets(id,name) VALUES('legacy-bucket','bucket')`,
		`INSERT INTO objects(id,bucket_id,path,content) VALUES('legacy-object','legacy-bucket','path','data')`,
	}
}

func TestMigrateV01PreservesData(t *testing.T) {
	path := t.TempDir() + "/v01.db"
	raw := rawDB(t, path)
	execAll(t, raw, v01Schema())
	require.NoError(t, raw.Close())
	db := assertLatest(t, path)
	defer db.Close()
	for table, id := range map[string]string{"projects": "legacy-project", "instances": "legacy-instance", "metadata": "legacy-metadata", "buckets": "legacy-bucket", "objects": "legacy-object"} {
		var got string
		require.NoError(t, db.QueryRow(`SELECT id FROM `+table+` WHERE id=?`, id).Scan(&got))
		require.Equal(t, id, got)
	}
	var org, slug string
	require.NoError(t, db.QueryRow(`SELECT org_id,slug FROM projects WHERE id='legacy-project'`).Scan(&org, &slug))
	require.Equal(t, "default-org", org)
	require.Equal(t, "legacy", slug)
	assertV2OwnershipConstraints(t, db)

	_, err := db.Exec(`INSERT INTO organizations(id,slug,name) VALUES('second-org','second-org','Second')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO projects(id,org_id,slug,name) VALUES('second-project','second-org','second','legacy')`)
	require.NoError(t, err, "project names must not retain the v0.1 global uniqueness constraint")
	_, err = db.Exec(`INSERT INTO buckets(id,project_id,name) VALUES('orphan',NULL,'orphan')`)
	require.Error(t, err)
}

func TestMigrateInterruptedV01(t *testing.T) {
	path := t.TempDir() + "/interrupted.db"
	raw := rawDB(t, path)
	execAll(t, raw, v01Schema())
	execAll(t, raw, []string{
		`CREATE TABLE organizations(id TEXT PRIMARY KEY,slug TEXT UNIQUE NOT NULL,name TEXT NOT NULL,created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`INSERT INTO organizations(id,slug,name) VALUES('partial-org','partial','Partial')`,
		`ALTER TABLE projects ADD COLUMN org_id TEXT`,
	})
	require.NoError(t, raw.Close())
	db := assertLatest(t, path)
	defer db.Close()
	var org, slug, project string
	require.NoError(t, db.QueryRow(`SELECT org_id,slug FROM projects WHERE id='legacy-project'`).Scan(&org, &slug))
	require.Equal(t, "partial-org", org)
	require.Equal(t, "legacy", slug)
	require.NoError(t, db.QueryRow(`SELECT project_id FROM buckets WHERE id='legacy-bucket'`).Scan(&project))
	require.Equal(t, "legacy-project", project)
	assertV2OwnershipConstraints(t, db)
}

func TestMigrateInterruptedV01AfterAllColumnsBeforeBackfill(t *testing.T) {
	path := t.TempDir() + "/interrupted-all-columns.db"
	raw := rawDB(t, path)
	execAll(t, raw, v01Schema())
	execAll(t, raw, []string{
		`CREATE TABLE organizations(id TEXT PRIMARY KEY,slug TEXT UNIQUE NOT NULL,name TEXT NOT NULL,created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP)`,
		`INSERT INTO organizations(id,slug,name) VALUES('partial-org','partial','Partial')`,
		`ALTER TABLE projects ADD COLUMN org_id TEXT`,
		`ALTER TABLE projects ADD COLUMN slug TEXT`,
		`ALTER TABLE buckets ADD COLUMN project_id TEXT`,
		`ALTER TABLE metadata ADD COLUMN org_id TEXT`,
	})
	require.NoError(t, raw.Close())

	db := assertLatest(t, path)
	defer db.Close()
	for table, column := range map[string]string{"projects": "org_id", "buckets": "project_id", "metadata": "org_id"} {
		var nulls int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE `+column+` IS NULL`).Scan(&nulls))
		require.Zero(t, nulls)
	}
	assertV2OwnershipConstraints(t, db)
}

func assertV2OwnershipConstraints(t *testing.T, db *DB) {
	t.Helper()
	for _, item := range []struct{ table, column, parent string }{
		{"projects", "org_id", "organizations"},
		{"buckets", "project_id", "projects"},
		{"metadata", "org_id", "organizations"},
	} {
		var notNull int
		require.NoError(t, db.QueryRow(`SELECT "notnull" FROM pragma_table_info(?) WHERE name=?`, item.table, item.column).Scan(&notNull))
		require.Equal(t, 1, notNull, "%s.%s must be NOT NULL", item.table, item.column)
		var foreignKeys int
		require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_list(?) WHERE "from"=? AND "table"=?`, item.table, item.column, item.parent).Scan(&foreignKeys))
		require.Equal(t, 1, foreignKeys, "%s.%s must reference %s", item.table, item.column, item.parent)
	}
}

func TestMigrateV02PreservesData(t *testing.T) {
	path := t.TempDir() + "/v02.db"
	raw := rawDB(t, path)
	execAll(t, raw, baseSchema)
	execAll(t, raw, []string{
		`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,applied_at DATETIME NOT NULL)`,
		`INSERT INTO schema_migrations VALUES(2,CURRENT_TIMESTAMP)`,
		`INSERT INTO organizations(id,slug,name)VALUES('org','org','Org')`,
		`INSERT INTO projects(id,org_id,slug,name)VALUES('project','org','project','Project')`,
		`INSERT INTO instances(id,project_id,name,region,cpu,memory_mb,image,status)VALUES('instance','project','vm','us-east-1',1,512,'image','running')`,
	})
	require.NoError(t, raw.Close())
	db := assertLatest(t, path)
	defer db.Close()
	var project string
	require.NoError(t, db.QueryRow(`SELECT project_id FROM instances WHERE id='instance'`).Scan(&project))
	require.Equal(t, "project", project)
}

func TestMigrationAndVersionRecordRollBackTogether(t *testing.T) {
	path := t.TempDir() + "/rollback.db"
	raw := rawDB(t, path)
	execAll(t, raw, baseSchema)
	execAll(t, raw, []string{
		`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,applied_at DATETIME NOT NULL)`,
		`INSERT INTO schema_migrations VALUES(2,CURRENT_TIMESTAMP)`,
		`CREATE TABLE networks(id TEXT PRIMARY KEY)`,
	})
	require.NoError(t, raw.Close())

	_, err := NewDB(path)
	require.Error(t, err)
	raw = rawDB(t, path)
	defer raw.Close()
	var version, subnetColumn int
	require.NoError(t, raw.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version))
	require.Equal(t, 2, version)
	require.NoError(t, raw.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('instances') WHERE name='subnet_id'`).Scan(&subnetColumn))
	require.Zero(t, subnetColumn)
}

func TestRejectNewerSchema(t *testing.T) {
	path := t.TempDir() + "/newer.db"
	raw := rawDB(t, path)
	execAll(t, raw, []string{`CREATE TABLE schema_migrations(version INTEGER PRIMARY KEY,applied_at DATETIME NOT NULL)`, `INSERT INTO schema_migrations VALUES(99,CURRENT_TIMESTAMP)`})
	require.NoError(t, raw.Close())
	_, e := NewDB(path)
	require.ErrorContains(t, e, "newer than supported")
}
