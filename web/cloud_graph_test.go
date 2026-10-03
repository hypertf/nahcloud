package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/pkg/auth"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/storage/sqlite"
	"github.com/stretchr/testify/require"
)

type fixtureGraphConsole struct {
	snapshot GraphSnapshot
	err      error
	scope    GraphScope
}

func (f *fixtureGraphConsole) Snapshot(scope GraphScope) (GraphSnapshot, error) {
	f.scope = scope
	return f.snapshot, f.err
}
func (f *fixtureGraphConsole) Create(scope GraphScope, _ GraphCreateInput) error {
	f.scope = scope
	return f.err
}
func (f *fixtureGraphConsole) Delete(scope GraphScope, _, _ string) error {
	f.scope = scope
	return f.err
}

func newWebGraphTestHandler(t *testing.T) (*Handler, *service.Service) {
	t.Helper()
	db, err := sqlite.NewDB(t.TempDir() + "/nahcloud.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	svc := service.NewService(
		sqlite.NewOrganizationRepository(db), sqlite.NewAPIKeyRepository(db),
		sqlite.NewSessionRepository(db), sqlite.NewProjectRepository(db),
		sqlite.NewInstanceRepository(db), sqlite.NewMetadataRepository(db),
		sqlite.NewBucketRepository(db), sqlite.NewObjectRepository(db),
		sqlite.NewGraphRepository(db),
	)
	return NewHandler(svc), svc
}

func graphFixture() GraphSnapshot {
	const (
		networkID    = "11111111111111111111111111111111"
		subnetID     = "22222222222222222222222222222222"
		instanceID   = "33333333333333333333333333333333"
		diskID       = "44444444444444444444444444444444"
		attachmentID = "55555555555555555555555555555555"
		lbID         = "66666666666666666666666666666666"
		backendID    = "77777777777777777777777777777777"
		policyID     = "88888888888888888888888888888888"
		bindingID    = "99999999999999999999999999999999"
	)
	return GraphSnapshot{
		Networks:      []GraphNetwork{{ID: networkID, Name: "production", Region: "us-east-1", Subnets: []GraphSubnet{{ID: subnetID, Name: "private-a", CIDR: "10.24.1.0/24", InstanceCount: 1, LBCount: 1}}}},
		Disks:         []GraphDisk{{ID: diskID, Name: "api-data", Region: "us-east-1", Type: "ssd", SizeGB: 100, Attachment: &GraphAttachment{ID: attachmentID, InstanceID: instanceID, InstanceName: "api-01", Device: "vdb"}}},
		Policies:      []GraphPolicy{{ID: policyID, Name: "service-operators", Description: "Operate production services", Effect: "allow", Actions: []string{"instance.read", "disk.read"}, Bindings: []GraphBinding{{ID: bindingID, PrincipalType: "api_key", PrincipalID: "terraform", TargetType: "network", TargetID: networkID, TargetName: "production"}}}},
		LoadBalancers: []GraphLoadBalancer{{ID: lbID, Name: "public-api", SubnetID: subnetID, SubnetName: "private-a", Region: "us-east-1", Protocol: "http", Port: 443, Algorithm: "round_robin", HealthCheckPath: "/healthz", Status: "active", Backends: []GraphBackend{{ID: backendID, InstanceID: instanceID, InstanceName: "api-01", Port: 8080, Weight: 10, Enabled: true, Healthy: true}}}},
		Instances:     []GraphInstance{{ID: instanceID, Name: "api-01", Region: "us-east-1", Status: "running", SubnetID: subnetID, SubnetName: "private-a", Disks: []GraphResourceLink{{ID: diskID, Name: "api-data"}}, LoadBalancers: []GraphLBMembership{{ID: lbID, Name: "public-api", BackendID: backendID, Port: 8080, Healthy: true}}}},
	}
}

func createGraphOrgProject(t *testing.T, svc *service.Service) (*domain.OrganizationWithAPIKey, *domain.Project) {
	t.Helper()
	org, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "graph-org", Name: "Graph Org"})
	require.NoError(t, err)
	project, err := svc.CreateProject(org.ID, domain.CreateProjectRequest{Slug: "edge", Name: "Edge"})
	require.NoError(t, err)
	return org, project
}

func graphRequest(org *domain.Organization, project *domain.Project, htmx bool) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Slug+"/cloud", nil)
	req = mux.SetURLVars(req, map[string]string{"project": project.Slug})
	req = req.WithContext(auth.WithOrg(req.Context(), org))
	if htmx {
		req.Header.Set("HX-Request", "true")
	}
	return req
}

func TestCloudGraphRendersFrozenRelationshipsAndReadOnlyFaultLab(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	org, project := createGraphOrgProject(t, svc)
	fixture := &fixtureGraphConsole{snapshot: graphFixture()}
	h.graph = fixture

	response := httptest.NewRecorder()
	h.ListCloud(response, graphRequest(&org.Organization, project, false))
	body := response.Body.String()
	require.Equal(t, http.StatusOK, response.Code, body)
	require.Contains(t, body, "Networks → subnets")
	require.Contains(t, body, "Disks ↔ attachment")
	require.Contains(t, body, "Policies → binding graph")
	require.Contains(t, body, "Instance relationship detail")
	require.Contains(t, body, "healthy")
	require.Contains(t, body, "Copy edge ID")
	require.Contains(t, body, "restricted while subnets exist")
	require.Contains(t, body, "restricted while instances or load balancers reference")
	require.Contains(t, body, "restricted while attached")
	require.Contains(t, body, "disk and instance will both remain")
	require.Contains(t, body, "backend edges will be deleted automatically")
	require.Contains(t, body, "Fault Lab")
	require.Contains(t, body, "Read only")
	require.NotContains(t, body, "/cloud/faults/")
	require.NotContains(t, body, ">Inject<")
	require.Equal(t, GraphScope{OrgID: org.ID, ProjectID: project.ID}, fixture.scope)

	partial := httptest.NewRecorder()
	h.ListCloud(partial, graphRequest(&org.Organization, project, true))
	require.NotContains(t, partial.Body.String(), "<!DOCTYPE html>")
	require.Contains(t, partial.Body.String(), "Instance relationship detail")
}

func TestCloudGraphRendersSectionEmptyStates(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	org, project := createGraphOrgProject(t, svc)
	h.graph = &fixtureGraphConsole{snapshot: GraphSnapshot{}}

	response := httptest.NewRecorder()
	h.ListCloud(response, graphRequest(&org.Organization, project, false))
	body := response.Body.String()
	require.Contains(t, body, "No networks")
	require.Contains(t, body, "No load balancers")
	require.Contains(t, body, "No disks in this project")
	require.Contains(t, body, "No organization policies")
	require.Contains(t, body, "No instances")
	require.Contains(t, body, "Loading the latest graph")
}

func TestCloudGraphProductionAdapterShowsRealEmptyState(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	org, project := createGraphOrgProject(t, svc)

	response := httptest.NewRecorder()
	h.ListCloud(response, graphRequest(&org.Organization, project, false))
	body := response.Body.String()
	require.Equal(t, http.StatusOK, response.Code, body)
	require.Contains(t, body, "No networks")
	require.Contains(t, body, "No load balancers")
	require.Contains(t, body, "No disks in this project")
	require.Contains(t, body, "No organization policies")
	require.Contains(t, body, "Fault Lab")
}

func TestCloudGraphProductionAdapterCreatesAndReadsRelationships(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	org, project := createGraphOrgProject(t, svc)
	adapter := h.graph
	scope := graphScope(&org.Organization, project)

	require.NoError(t, adapter.Create(scope, GraphCreateInput{Kind: "network", Name: "private", Region: domain.RegionUSEast1}))
	networks, err := svc.ListNetworks(project.ID)
	require.NoError(t, err)
	require.Len(t, networks, 1)
	require.NoError(t, adapter.Create(scope, GraphCreateInput{Kind: "subnet", ParentID: networks[0].ID, Name: "apps", CIDR: "10.42.0.0/24"}))
	require.NoError(t, adapter.Create(scope, GraphCreateInput{Kind: "disk", Name: "data", Region: domain.RegionUSEast1, DiskType: "ssd", SizeGB: 100}))
	require.NoError(t, adapter.Create(scope, GraphCreateInput{Kind: "policy", Name: "readers", Effect: "allow", Actions: "instances.get, disks.get"}))

	snapshot, err := adapter.Snapshot(scope)
	require.NoError(t, err)
	require.Len(t, snapshot.Networks, 1)
	require.Len(t, snapshot.Networks[0].Subnets, 1)
	require.Equal(t, "10.42.0.0/24", snapshot.Networks[0].Subnets[0].CIDR)
	require.Len(t, snapshot.Disks, 1)
	require.Equal(t, 100, snapshot.Disks[0].SizeGB)
	require.Len(t, snapshot.Policies, 1)
	require.Equal(t, []string{"instances.get", "disks.get"}, snapshot.Policies[0].Actions)
}

func TestCloudGraphDeleteFailureTargetsVisibleAlert(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	org, project := createGraphOrgProject(t, svc)
	h.graph = &fixtureGraphConsole{err: domain.ConflictError("network is in use")}
	req := httptest.NewRequest(http.MethodDelete, "/projects/edge/cloud/network/network-id", nil)
	req = mux.SetURLVars(req, map[string]string{"project": project.Slug, "kind": "network", "id": "network-id"})
	req = req.WithContext(auth.WithOrg(req.Context(), &org.Organization))
	response := httptest.NewRecorder()

	h.DeleteGraphResource(response, req)

	require.Equal(t, http.StatusConflict, response.Code)
	require.Equal(t, "#graph-error", response.Header().Get("HX-Retarget"))
	require.Contains(t, response.Body.String(), "network is in use")
}

func TestCloudGraphPageHasResponsiveAndAccessibleShell(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	org, project := createGraphOrgProject(t, svc)
	h.graph = &fixtureGraphConsole{snapshot: graphFixture()}
	response := httptest.NewRecorder()
	h.ListCloud(response, graphRequest(&org.Organization, project, false))
	body := response.Body.String()
	require.Contains(t, body, `<html lang="en">`)
	require.Contains(t, body, `name="viewport"`)
	require.Contains(t, body, `role="dialog"`)
	require.Contains(t, body, `id="graph-error" role="alert"`)
	require.Contains(t, body, `.cloud-graph`)
}

func TestCloudGraphHandlerRejectsAnotherOrganizationsProjectBeforeAdapter(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	first, project := createGraphOrgProject(t, svc)
	second, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "second-org", Name: "Second Org"})
	require.NoError(t, err)
	fixture := &fixtureGraphConsole{snapshot: graphFixture()}
	h.graph = fixture

	req := httptest.NewRequest(http.MethodGet, "/projects/"+project.Slug+"/cloud", nil)
	req = mux.SetURLVars(req, map[string]string{"project": project.Slug})
	req = req.WithContext(auth.WithOrg(req.Context(), &second.Organization))
	response := httptest.NewRecorder()
	h.ListCloud(response, req)

	require.Equal(t, http.StatusNotFound, response.Code)
	require.Equal(t, GraphScope{}, fixture.scope)
	require.NotEqual(t, first.ID, second.ID)
}
