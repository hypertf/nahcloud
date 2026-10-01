package service

import (
	"encoding/base64"
	"fmt"
	"sync"
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

func TestBucketIdentityIsStableAndProjectScoped(t *testing.T) {
	svc, _ := newTestService(t)

	org, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "bucket-org", Name: "Bucket Org"})
	require.NoError(t, err)
	firstProject, err := svc.CreateProject(org.ID, domain.CreateProjectRequest{Slug: "first", Name: "First"})
	require.NoError(t, err)
	secondProject, err := svc.CreateProject(org.ID, domain.CreateProjectRequest{Slug: "second", Name: "Second"})
	require.NoError(t, err)

	first, err := svc.CreateBucket(firstProject.ID, domain.CreateBucketRequest{Name: "artifacts"})
	require.NoError(t, err)
	second, err := svc.CreateBucket(secondProject.ID, domain.CreateBucketRequest{Name: "artifacts"})
	require.NoError(t, err)
	require.NotEqual(t, first.ID, second.ID)
	require.NotEqual(t, first.Name, first.ID)

	renamed, err := svc.UpdateBucket(first.ID, domain.UpdateBucketRequest{Name: "releases"})
	require.NoError(t, err)
	require.Equal(t, first.ID, renamed.ID)
	require.Equal(t, "releases", renamed.Name)

	_, err = svc.GetBucketByName(firstProject.ID, "artifacts")
	require.True(t, domain.IsNotFound(err))
	byNewName, err := svc.GetBucketByName(firstProject.ID, "releases")
	require.NoError(t, err)
	require.Equal(t, first.ID, byNewName.ID)
}

func TestDeletingBucketsCascadesObjectsDuringConcurrentRecreation(t *testing.T) {
	svc, _ := newTestService(t)
	org, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "cascade-org", Name: "Cascade Org"})
	require.NoError(t, err)

	const workers = 8
	var wg sync.WaitGroup
	errors := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			project, err := svc.CreateProject(org.ID, domain.CreateProjectRequest{
				Slug: fmt.Sprintf("project-%d", i), Name: fmt.Sprintf("Project %d", i),
			})
			if err != nil {
				errors <- err
				return
			}
			bucket, err := svc.CreateBucket(project.ID, domain.CreateBucketRequest{Name: "artifacts"})
			if err != nil {
				errors <- err
				return
			}
			object, err := svc.CreateObject(domain.CreateObjectRequest{
				BucketID: bucket.ID, Path: "build/output.txt", Content: "first",
			})
			if err != nil {
				errors <- err
				return
			}
			if err := svc.DeleteBucket(bucket.ID); err != nil {
				errors <- err
				return
			}
			if _, err := svc.GetObject(object.ID); !domain.IsNotFound(err) {
				errors <- fmt.Errorf("object %s survived bucket deletion: %v", object.ID, err)
				return
			}
			recreated, err := svc.CreateBucket(project.ID, domain.CreateBucketRequest{Name: "artifacts"})
			if err != nil {
				errors <- err
				return
			}
			if _, err := svc.CreateObject(domain.CreateObjectRequest{
				BucketID: recreated.ID, Path: "build/output.txt", Content: "second",
			}); err != nil {
				errors <- err
			}
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
}
