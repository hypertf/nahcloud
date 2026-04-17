package service

import (
	"encoding/base64"
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/storage/sqlite"
	"github.com/stretchr/testify/require"
)

func newTestService(t *testing.T) (*Service, *sqlite.DB) {
	t.Helper()

	db, err := sqlite.NewDB(":memory:")
	require.NoError(t, err)

	svc := NewService(
		sqlite.NewOrganizationRepository(db),
		sqlite.NewAPIKeyRepository(db),
		sqlite.NewSessionRepository(db),
		sqlite.NewProjectRepository(db),
		sqlite.NewInstanceRepository(db),
		sqlite.NewMetadataRepository(db),
		sqlite.NewBucketRepository(db),
		sqlite.NewObjectRepository(db),
	)

	t.Cleanup(func() {
		require.NoError(t, db.Close())
	})

	return svc, db
}

func TestCreateOrganizationWithSessionStartsWithoutAPIKeys(t *testing.T) {
	svc, _ := newTestService(t)

	orgWithSession, err := svc.CreateOrganizationWithSession()
	require.NoError(t, err)

	keys, err := svc.ListAPIKeys(orgWithSession.ID)
	require.NoError(t, err)
	require.Empty(t, keys)
}

func TestResetOrganizationClearsResourcesAndKeepsOrg(t *testing.T) {
	svc, _ := newTestService(t)

	orgWithKey, err := svc.CreateOrganization(domain.CreateOrganizationRequest{
		Slug: "reset-org",
		Name: "Reset Org",
	})
	require.NoError(t, err)

	project, err := svc.CreateProject(orgWithKey.ID, domain.CreateProjectRequest{
		Slug: "sandbox",
		Name: "Sandbox",
	})
	require.NoError(t, err)

	_, err = svc.CreateInstance(domain.CreateInstanceRequest{
		ProjectID: project.ID,
		Name:      "vm-1",
		Region:    domain.RegionUSEast1,
		CPU:       1,
		MemoryMB:  512,
		Image:     "ubuntu:22.04",
		Status:    domain.StatusRunning,
	})
	require.NoError(t, err)

	_, err = svc.CreateBucket(project.ID, domain.CreateBucketRequest{Name: "artifacts"})
	require.NoError(t, err)

	bucket, err := svc.GetBucketByName(project.ID, "artifacts")
	require.NoError(t, err)

	_, err = svc.CreateObject(domain.CreateObjectRequest{
		BucketID: bucket.ID,
		Path:     "build/output.txt",
		Content:  base64.StdEncoding.EncodeToString([]byte("hello")),
	})
	require.NoError(t, err)

	_, err = svc.CreateMetadata(domain.CreateMetadataRequest{
		OrgID: orgWithKey.ID,
		Path:  "config/settings/mode",
		Value: "test",
	})
	require.NoError(t, err)

	err = svc.ResetOrganization(orgWithKey.ID)
	require.NoError(t, err)

	org, err := svc.GetOrganization(orgWithKey.ID)
	require.NoError(t, err)
	require.Equal(t, "reset-org", org.Slug)

	projects, err := svc.ListProjects(domain.ProjectListOptions{OrgID: orgWithKey.ID})
	require.NoError(t, err)
	require.Empty(t, projects)

	metadata, err := svc.ListMetadata(domain.MetadataListOptions{OrgID: orgWithKey.ID})
	require.NoError(t, err)
	require.Empty(t, metadata)

	keys, err := svc.ListAPIKeys(orgWithKey.ID)
	require.NoError(t, err)
	require.Len(t, keys, 1)
}
