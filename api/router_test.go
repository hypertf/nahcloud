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
		sqlite.NewGraphRepository(db),
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

func TestGraphResourceAPIContractsAndTenantIsolation(t *testing.T) {
	handler := newTestRouter(t)
	first := createTestOrganization(t, handler, "graph-first")
	second := createTestOrganization(t, handler, "graph-second")
	createProject := func(org domain.OrganizationWithAPIKey) domain.Project {
		response := performRequest(t, handler, http.MethodPost, "/v1/projects", domain.CreateProjectRequest{Slug: "cloud", Name: "Cloud"}, org.APIKey.Token)
		require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
		var project domain.Project
		require.NoError(t, json.Unmarshal(response.Body.Bytes(), &project))
		return project
	}
	project := createProject(first)
	createProject(second)
	for _, path := range []string{"/v1/projects/cloud/networks", "/v1/projects/cloud/disks", "/v1/projects/cloud/load-balancers", "/v1/policies"} {
		response := performRequest(t, handler, http.MethodGet, path, nil, first.APIKey.Token)
		require.Equal(t, http.StatusOK, response.Code, response.Body.String())
		require.JSONEq(t, `[]`, response.Body.String(), path)
	}

	instanceResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/instances", domain.CreateInstanceRequest{Name: "app", Region: domain.RegionUSEast1, CPU: 1, MemoryMB: 512, Image: "test"}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, instanceResponse.Code, instanceResponse.Body.String())
	var instance domain.Instance
	require.NoError(t, json.Unmarshal(instanceResponse.Body.Bytes(), &instance))

	networkResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/networks", domain.CreateNetworkRequest{Name: "private", Region: domain.RegionUSEast1}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, networkResponse.Code, networkResponse.Body.String())
	var network domain.Network
	require.NoError(t, json.Unmarshal(networkResponse.Body.Bytes(), &network))
	require.NotEqual(t, network.Name, network.ID)
	subnetResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/networks/"+network.ID+"/subnets", domain.CreateSubnetRequest{Name: "apps", CIDR: "10.1.1.0/24"}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, subnetResponse.Code, subnetResponse.Body.String())
	var subnet domain.Subnet
	require.NoError(t, json.Unmarshal(subnetResponse.Body.Bytes(), &subnet))
	boundInstanceResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/instances", domain.CreateInstanceRequest{Name: "backend", Region: domain.RegionUSEast1, CPU: 1, MemoryMB: 512, Image: "test", SubnetID: &subnet.ID}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, boundInstanceResponse.Code, boundInstanceResponse.Body.String())
	var boundInstance domain.Instance
	require.NoError(t, json.Unmarshal(boundInstanceResponse.Body.Bytes(), &boundInstance))
	newName := "services"
	response := performRequest(t, handler, http.MethodPatch, "/v1/projects/cloud/networks/"+network.ID+"/subnets/"+subnet.ID, domain.UpdateSubnetRequest{Name: &newName}, first.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), newName)
	response = performRequest(t, handler, http.MethodGet, "/v1/projects/cloud/networks/"+network.ID, nil, second.APIKey.Token)
	require.Equal(t, http.StatusNotFound, response.Code)

	diskResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/disks", domain.CreateDiskRequest{Name: "data", Region: domain.RegionUSEast1, Type: "ssd", SizeGB: 10}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, diskResponse.Code, diskResponse.Body.String())
	var disk domain.Disk
	require.NoError(t, json.Unmarshal(diskResponse.Body.Bytes(), &disk))
	attachmentResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/disks/"+disk.ID+"/attachments", domain.CreateDiskAttachmentRequest{InstanceID: instance.ID, Device: "vdb"}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, attachmentResponse.Code, attachmentResponse.Body.String())
	var attachment domain.DiskAttachment
	require.NoError(t, json.Unmarshal(attachmentResponse.Body.Bytes(), &attachment))
	require.Equal(t, instance.ID, attachment.InstanceID)
	response = performRequest(t, handler, http.MethodPatch, "/v1/projects/cloud/disks/"+disk.ID+"/attachments/"+attachment.ID, map[string]any{"device": "vdc"}, first.APIKey.Token)
	require.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, response.Code)

	orgPolicyResponse := performRequest(t, handler, http.MethodPost, "/v1/policies", domain.CreatePolicyRequest{Name: "organization", Effect: "allow", Actions: []string{"read"}}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, orgPolicyResponse.Code, orgPolicyResponse.Body.String())
	var orgPolicy domain.Policy
	require.NoError(t, json.Unmarshal(orgPolicyResponse.Body.Bytes(), &orgPolicy))
	bindingResponse := performRequest(t, handler, http.MethodPost, "/v1/policies/"+orgPolicy.ID+"/bindings", domain.CreatePolicyBindingRequest{PrincipalType: "organization", PrincipalID: first.ID, TargetType: "project", TargetID: project.ID}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, bindingResponse.Code, bindingResponse.Body.String())
	var binding domain.PolicyBinding
	require.NoError(t, json.Unmarshal(bindingResponse.Body.Bytes(), &binding))
	response = performRequest(t, handler, http.MethodPatch, "/v1/policies/"+orgPolicy.ID+"/bindings/"+binding.ID, map[string]any{"target_id": first.ID}, first.APIKey.Token)
	require.Contains(t, []int{http.StatusNotFound, http.StatusMethodNotAllowed}, response.Code)
	evaluationResponse := performRequest(t, handler, http.MethodPost, "/v1/policy-evaluations", domain.PolicyEvaluationRequest{PrincipalType: "organization", PrincipalID: first.ID, Action: "read", TargetType: "network", TargetID: network.ID}, first.APIKey.Token)
	require.Equal(t, http.StatusOK, evaluationResponse.Code, evaluationResponse.Body.String())
	var evaluation domain.PolicyEvaluation
	require.NoError(t, json.Unmarshal(evaluationResponse.Body.Bytes(), &evaluation))
	require.Equal(t, "allow", evaluation.Result)
	require.Equal(t, []string{binding.ID}, evaluation.MatchingBindingIDs)

	lbResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/load-balancers", domain.CreateLoadBalancerRequest{Name: "public", SubnetID: subnet.ID, Protocol: "http", Port: 80, Algorithm: "round_robin", HealthCheckPath: "/health"}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, lbResponse.Code, lbResponse.Body.String())
	var lb domain.LoadBalancer
	require.NoError(t, json.Unmarshal(lbResponse.Body.Bytes(), &lb))
	backendResponse := performRequest(t, handler, http.MethodPost, "/v1/projects/cloud/load-balancers/"+lb.ID+"/backends", domain.CreateLoadBalancerBackendRequest{InstanceID: boundInstance.ID, Port: 8080, Weight: 10}, first.APIKey.Token)
	require.Equal(t, http.StatusCreated, backendResponse.Code, backendResponse.Body.String())
	var backend domain.LoadBalancerBackend
	require.NoError(t, json.Unmarshal(backendResponse.Body.Bytes(), &backend))
	require.True(t, backend.Healthy)
	weight := 20
	response = performRequest(t, handler, http.MethodPatch, "/v1/projects/cloud/load-balancers/"+lb.ID+"/backends/"+backend.ID, domain.UpdateLoadBalancerBackendRequest{Weight: &weight}, first.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Contains(t, response.Body.String(), `"weight":20`)

	for _, path := range []string{"/v1/projects/cloud/networks", "/v1/projects/cloud/disks", "/v1/policies", "/v1/projects/cloud/load-balancers"} {
		response = performRequest(t, handler, http.MethodGet, path, nil, first.APIKey.Token)
		require.Equal(t, http.StatusOK, response.Code, path+": "+response.Body.String())
		require.NotEqual(t, "null\n", response.Body.String(), path)
	}

	response = performRequest(t, handler, http.MethodDelete, "/v1/projects/cloud/networks/"+network.ID, nil, first.APIKey.Token)
	require.Equal(t, http.StatusConflict, response.Code, response.Body.String())
}
