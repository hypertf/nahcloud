package sqlite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGraphSchemaIsAddedWithoutChangingExistingData(t *testing.T) {
	path := t.TempDir() + "/existing.db"
	db, err := NewDB(path)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO organizations(id,slug,name) VALUES('org','org','Org')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO projects(id,org_id,slug,name) VALUES('project','org','project','Project')`)
	require.NoError(t, err)
	for _, table := range []string{"load_balancer_backends", "load_balancers", "policy_bindings", "policies", "disk_attachments", "disks", "subnets", "networks"} {
		_, err = db.Exec(`DROP TABLE ` + table)
		require.NoError(t, err)
	}
	require.NoError(t, db.Close())

	db, err = NewDB(path)
	require.NoError(t, err)
	defer db.Close()
	var name string
	require.NoError(t, db.QueryRow(`SELECT name FROM projects WHERE id='project'`).Scan(&name))
	require.Equal(t, "Project", name)
	for _, table := range []string{"networks", "subnets", "disks", "disk_attachments", "policies", "policy_bindings", "load_balancers", "load_balancer_backends"} {
		var found string
		require.NoError(t, db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&found))
		require.Equal(t, table, found)
	}
}
