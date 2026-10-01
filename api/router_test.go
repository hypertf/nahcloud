package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/storage/sqlite"
	"github.com/stretchr/testify/require"
)

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	db, err := sqlite.NewDB(t.TempDir() + "/nahcloud.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })

	svc := service.NewService(
		sqlite.NewOrganizationRepository(db),
		sqlite.NewAPIKeyRepository(db),
		sqlite.NewSessionRepository(db),
		sqlite.NewProjectRepository(db),
		sqlite.NewInstanceRepository(db),
		sqlite.NewMetadataRepository(db),
		sqlite.NewBucketRepository(db),
		sqlite.NewObjectRepository(db),
	)
	return SetupRouter(NewHandler(svc), svc, "test")
}

func performRequest(t *testing.T, handler http.Handler, method, path string, body any, token string) *httptest.ResponseRecorder {
	t.Helper()

	var requestBody bytes.Buffer
	if body != nil {
		require.NoError(t, json.NewEncoder(&requestBody).Encode(body))
	}
	req := httptest.NewRequest(method, path, &requestBody)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response
}

func createTestOrganization(t *testing.T, handler http.Handler, slug string) domain.OrganizationWithAPIKey {
	t.Helper()

	response := performRequest(t, handler, http.MethodPost, "/v1/orgs", domain.CreateOrganizationRequest{
		Slug: slug,
		Name: "Test " + slug,
	}, "")
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())

	var org domain.OrganizationWithAPIKey
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &org))
	require.NotEmpty(t, org.ID)
	require.NotEmpty(t, org.APIKey.Token)
	return org
}

func TestAPITokenIsRequired(t *testing.T) {
	handler := newTestRouter(t)

	response := performRequest(t, handler, http.MethodGet, "/v1/org", nil, "")
	require.Equal(t, http.StatusUnauthorized, response.Code)
	require.JSONEq(t, `{"error":"API token required"}`, response.Body.String())
}

func TestResourceLifecycleAndTenantIsolation(t *testing.T) {
	handler := newTestRouter(t)
	firstOrg := createTestOrganization(t, handler, "first-org")
	secondOrg := createTestOrganization(t, handler, "second-org")

	projectResponse := performRequest(t, handler, http.MethodPost, "/v1/projects", domain.CreateProjectRequest{
		Slug: "sandbox",
		Name: "Sandbox",
	}, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusCreated, projectResponse.Code, projectResponse.Body.String())
	var project domain.Project
	require.NoError(t, json.Unmarshal(projectResponse.Body.Bytes(), &project))

	keyResponse := performRequest(t, handler, http.MethodPost, "/v1/api-keys", domain.CreateAPIKeyRequest{Name: "terraform"}, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusCreated, keyResponse.Code, keyResponse.Body.String())
	var key domain.APIKeyWithToken
	require.NoError(t, json.Unmarshal(keyResponse.Body.Bytes(), &key))
	require.NotEmpty(t, key.Token)
	keyResponse = performRequest(t, handler, http.MethodGet, "/v1/api-keys/"+key.ID, nil, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusOK, keyResponse.Code, keyResponse.Body.String())
	keyResponse = performRequest(t, handler, http.MethodGet, "/v1/api-keys/"+key.ID, nil, secondOrg.APIKey.Token)
	require.Equal(t, http.StatusNotFound, keyResponse.Code, keyResponse.Body.String())

	instanceResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/sandbox/instances", domain.CreateInstanceRequest{
		Name:     "web",
		Region:   domain.RegionUSEast1,
		CPU:      2,
		MemoryMB: 1024,
		Image:    "ubuntu:24.04",
		Status:   domain.StatusRunning,
	}, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusCreated, instanceResponse.Code, instanceResponse.Body.String())
	var instance domain.Instance
	require.NoError(t, json.Unmarshal(instanceResponse.Body.Bytes(), &instance))
	require.Equal(t, project.ID, instance.ProjectID)

	otherProjectResponse := performRequest(t, handler, http.MethodPost, "/v1/projects", domain.CreateProjectRequest{
		Slug: "sandbox",
		Name: "Sandbox",
	}, secondOrg.APIKey.Token)
	require.Equal(t, http.StatusCreated, otherProjectResponse.Code, otherProjectResponse.Body.String())

	response := performRequest(t, handler, http.MethodGet, "/v1/projects/sandbox/instances/"+instance.ID, nil, secondOrg.APIKey.Token)
	require.Equal(t, http.StatusNotFound, response.Code)

	bucketResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/sandbox/buckets", domain.CreateBucketRequest{Name: "artifacts"}, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusCreated, bucketResponse.Code, bucketResponse.Body.String())
	var bucket domain.Bucket
	require.NoError(t, json.Unmarshal(bucketResponse.Body.Bytes(), &bucket))
	require.NotEqual(t, bucket.Name, bucket.ID)

	renameResponse := performRequest(t, handler, http.MethodPatch, "/v1/projects/sandbox/buckets/artifacts", domain.UpdateBucketRequest{Name: "releases"}, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusOK, renameResponse.Code, renameResponse.Body.String())
	var renamed domain.Bucket
	require.NoError(t, json.Unmarshal(renameResponse.Body.Bytes(), &renamed))
	require.Equal(t, bucket.ID, renamed.ID)
	require.Equal(t, "releases", renamed.Name)

	stableBucketPath := "/v1/projects/sandbox/buckets-by-id/" + bucket.ID
	getByIDResponse := performRequest(t, handler, http.MethodGet, stableBucketPath, nil, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusOK, getByIDResponse.Code, getByIDResponse.Body.String())

	objectResponse := performRequest(t, handler, http.MethodPost, stableBucketPath+"/objects", domain.CreateObjectRequest{
		Path:    "build/output.txt",
		Content: "aGVsbG8=",
	}, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusCreated, objectResponse.Code, objectResponse.Body.String())
	var object domain.Object
	require.NoError(t, json.Unmarshal(objectResponse.Body.Bytes(), &object))
	require.Equal(t, bucket.ID, object.BucketID)

	metadataResponse := performRequest(t, handler, http.MethodPost, "/v1/metadata", domain.CreateMetadataRequest{
		Path:  "config/environment",
		Value: "test",
	}, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusCreated, metadataResponse.Code, metadataResponse.Body.String())
}

func TestTerraformStateBackendIsScopedAndLocksAtomically(t *testing.T) {
	handler := newTestRouter(t)
	firstOrg := createTestOrganization(t, handler, "state-one")
	secondOrg := createTestOrganization(t, handler, "state-two")

	stateRequest := func(method, stateID string, body any, token string) *httptest.ResponseRecorder {
		var requestBody bytes.Buffer
		if body != nil {
			switch value := body.(type) {
			case string:
				requestBody.WriteString(value)
			default:
				require.NoError(t, json.NewEncoder(&requestBody).Encode(value))
			}
		}
		req := httptest.NewRequest(method, "/v1/tfstate/"+stateID, &requestBody)
		req.SetBasicAuth("nah", token)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		return response
	}

	require.Equal(t, http.StatusOK, stateRequest(http.MethodPost, "shared", `{"serial":1}`, firstOrg.APIKey.Token).Code)
	require.Equal(t, http.StatusOK, stateRequest(http.MethodPost, "shared", `{"serial":2}`, secondOrg.APIKey.Token).Code)
	require.JSONEq(t, `{"serial":1}`, stateRequest(http.MethodGet, "shared", nil, firstOrg.APIKey.Token).Body.String())
	require.JSONEq(t, `{"serial":2}`, stateRequest(http.MethodGet, "shared", nil, secondOrg.APIKey.Token).Body.String())

	const contenders = 8
	var wg sync.WaitGroup
	statuses := make(chan int, contenders)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			lock := fmt.Sprintf(`{"ID":"lock-%d","Operation":"apply"}`, id)
			statuses <- stateRequest("LOCK", "contended", lock, firstOrg.APIKey.Token).Code
		}(i)
	}
	wg.Wait()
	close(statuses)

	acquired := 0
	blocked := 0
	for status := range statuses {
		switch status {
		case http.StatusOK:
			acquired++
		case http.StatusLocked:
			blocked++
		default:
			t.Fatalf("unexpected lock status %d", status)
		}
	}
	require.Equal(t, 1, acquired)
	require.Equal(t, contenders-1, blocked)

	lockResponse := stateRequest("LOCK", "deploy", `{"ID":"deploy-lock","Operation":"apply"}`, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusOK, lockResponse.Code, lockResponse.Body.String())
	require.Equal(t, http.StatusLocked, stateRequest(http.MethodPost, "deploy", `{"serial":3}`, firstOrg.APIKey.Token).Code)

	unlockResponse := stateRequest("UNLOCK", "deploy", `{"ID":"wrong-lock"}`, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusConflict, unlockResponse.Code)
	unlockResponse = stateRequest("UNLOCK", "deploy", `{"ID":"deploy-lock"}`, firstOrg.APIKey.Token)
	require.Equal(t, http.StatusOK, unlockResponse.Code, unlockResponse.Body.String())
	require.Equal(t, http.StatusOK, stateRequest(http.MethodPost, "deploy", `{"serial":3}`, firstOrg.APIKey.Token).Code)
}
