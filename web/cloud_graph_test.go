package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/pkg/auth"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/storage/sqlite"
	"github.com/stretchr/testify/require"
)

func newWebGraphTestHandler(t *testing.T) (*Handler, *service.Service) {
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
	return NewHandler(svc), svc
}

func TestCloudGraphRendersFullAndHTMXStates(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	org, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "graph-org", Name: "Graph Org"})
	require.NoError(t, err)
	project, err := svc.CreateProject(org.ID, domain.CreateProjectRequest{Slug: "edge", Name: "Edge"})
	require.NoError(t, err)

	request := func(path string, htmx bool) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = mux.SetURLVars(req, map[string]string{"project": project.Slug})
		req = req.WithContext(auth.WithOrg(req.Context(), &org.Organization))
		if htmx {
			req.Header.Set("HX-Request", "true")
		}
		response := httptest.NewRecorder()
		h.ListCloud(response, req)
		return response
	}

	full := request("/projects/edge/cloud", false)
	require.Equal(t, http.StatusOK, full.Code, full.Body.String())
	require.Contains(t, full.Body.String(), "<!DOCTYPE html>")
	require.Contains(t, full.Body.String(), "Networks → subnets")
	require.Contains(t, full.Body.String(), "Load balancers → backends")
	require.Contains(t, full.Body.String(), "Tenant fault scenarios")

	partial := request("/projects/edge/cloud?state=error", true)
	require.Equal(t, http.StatusOK, partial.Code, partial.Body.String())
	require.NotContains(t, partial.Body.String(), "<!DOCTYPE html>")
	require.Contains(t, partial.Body.String(), `role="alert"`)
	require.Contains(t, partial.Body.String(), "could not load this project topology")
}

func TestGraphPreviewAdapterRejectsCrossProjectRelationshipsAndDeletes(t *testing.T) {
	graph := newPreviewGraphConsole()
	first, err := graph.Snapshot("project-one")
	require.NoError(t, err)
	second, err := graph.Snapshot("project-two")
	require.NoError(t, err)

	err = graph.Create("project-two", GraphCreateInput{Kind: "backend", ParentID: second.LoadBalancers[0].ID, InstanceID: first.Instances[0].ID, Port: "8080"})
	require.ErrorIs(t, err, errGraphResourceNotFound)
	err = graph.Delete("project-two", "network", first.Networks[0].ID)
	require.ErrorIs(t, err, errGraphResourceNotFound)

	firstAfter, err := graph.Snapshot("project-one")
	require.NoError(t, err)
	require.Equal(t, first.Networks[0].ID, firstAfter.Networks[0].ID)
}

func TestCloudGraphHandlerRejectsAnotherOrganizationsProject(t *testing.T) {
	h, svc := newWebGraphTestHandler(t)
	first, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "first-org", Name: "First Org"})
	require.NoError(t, err)
	second, err := svc.CreateOrganization(domain.CreateOrganizationRequest{Slug: "second-org", Name: "Second Org"})
	require.NoError(t, err)
	project, err := svc.CreateProject(first.ID, domain.CreateProjectRequest{Slug: "private", Name: "Private"})
	require.NoError(t, err)
	snapshot, err := h.graph.Snapshot(project.ID)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodDelete, "/projects/private/cloud/network/"+snapshot.Networks[0].ID, strings.NewReader(""))
	req = mux.SetURLVars(req, map[string]string{"project": "private", "kind": "network", "id": snapshot.Networks[0].ID})
	req = req.WithContext(auth.WithOrg(req.Context(), &second.Organization))
	response := httptest.NewRecorder()
	h.DeleteGraphResource(response, req)

	require.Equal(t, http.StatusBadRequest, response.Code)
	after, err := h.graph.Snapshot(project.ID)
	require.NoError(t, err)
	require.Len(t, after.Networks, len(snapshot.Networks))
}
