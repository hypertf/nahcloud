package sqlite

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/stretchr/testify/require"
)

func TestGraphCreateForeignKeyRaceIsConflict(t *testing.T) {
	err := graphWriteError("network", "private", errors.New("FOREIGN KEY constraint failed"))
	require.True(t, domain.IsConflict(err), err)
}

func graphRepositoryFixture(t *testing.T) (*DB, *GraphRepository) {
	t.Helper()
	db, err := NewDB(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	_, err = db.Exec(`INSERT INTO organizations(id,slug,name) VALUES('org','org','Org')`)
	require.NoError(t, err)
	_, err = db.Exec(`INSERT INTO projects(id,org_id,slug,name) VALUES('project','org','project','Project')`)
	require.NoError(t, err)
	return db, NewGraphRepository(db)
}

func TestProjectNodeLimitIsEnforcedInCreateTransaction(t *testing.T) {
	db, repo := graphRepositoryFixture(t)
	_, err := db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<1000)
		INSERT INTO networks(id,project_id,name,region) SELECT printf('network-%04d',x),'project',printf('network-%04d',x),'us-east-1' FROM n`)
	require.NoError(t, err)
	err = repo.CreateNetwork(&domain.Network{ID: "network-over-limit", ProjectID: "project", Name: "over-limit", Region: domain.RegionUSEast1})
	require.True(t, domain.IsLimitExceeded(err), err)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM networks WHERE project_id='project'`).Scan(&count))
	require.Equal(t, 1000, count)
}

func TestOrganizationEdgeLimitCountsAllEdgeFamilies(t *testing.T) {
	db, repo := graphRepositoryFixture(t)
	_, err := db.Exec(`INSERT INTO policies(id,org_id,name,effect,actions) VALUES('policy','org','Policy','allow','["*"]')`)
	require.NoError(t, err)
	_, err = db.Exec(`WITH RECURSIVE n(x) AS (SELECT 1 UNION ALL SELECT x+1 FROM n WHERE x<5000)
		INSERT INTO policy_bindings(id,org_id,policy_id,principal_type,principal_id,target_type,target_id)
		SELECT printf('binding-%04d',x),'org','policy','organization','org','organization',printf('target-%04d',x) FROM n`)
	require.NoError(t, err)
	err = repo.CreateBinding(&domain.PolicyBinding{
		ID: "binding-over-limit", OrgID: "org", PolicyID: "policy",
		PrincipalType: "organization", PrincipalID: "org", TargetType: "organization", TargetID: "org",
	})
	require.True(t, domain.IsLimitExceeded(err), fmt.Sprintf("expected edge quota error, got %v", err))
}
