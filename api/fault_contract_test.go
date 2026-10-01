package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hypertf/nahcloud/domain"
	"github.com/hypertf/nahcloud/service"
	"github.com/hypertf/nahcloud/storage/sqlite"
	"github.com/stretchr/testify/require"
)

func newFaultTestRouter(t *testing.T) http.Handler {
	db, err := sqlite.NewDB(t.TempDir() + "/nahcloud.db")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	svc := service.NewService(sqlite.NewOrganizationRepository(db), sqlite.NewAPIKeyRepository(db), sqlite.NewSessionRepository(db), sqlite.NewProjectRepository(db), sqlite.NewInstanceRepository(db), sqlite.NewMetadataRepository(db), sqlite.NewBucketRepository(db), sqlite.NewObjectRepository(db))
	faults := service.NewFaultService(sqlite.NewFaultRepository(db))
	return SetupRouter(NewHandler(svc, faults), svc, "test")
}

func TestFaultAPIContractAndTenantIsolation(t *testing.T) {
	router := newFaultTestRouter(t)
	one := createTestOrganization(t, router, "fault-one")
	two := createTestOrganization(t, router, "fault-two")
	req := domain.PutFaultScenarioRequest{Name: "outage", Enabled: true, Seed: 17, Rules: []domain.FaultRule{{
		ID: "org-read", Operation: "GET", Route: "/v1/org", Status: 503, EveryN: 1,
	}}}
	response := performRequest(t, router, http.MethodPut, "/v1/faults/scenario", req, one.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())

	response = performRequest(t, router, http.MethodGet, "/v1/org", nil, one.APIKey.Token)
	require.Equal(t, http.StatusServiceUnavailable, response.Code)
	require.Equal(t, "org-read", response.Header().Get("X-NahCloud-Fault-Rule"))
	require.JSONEq(t, `{"error":"injected fault","rule_id":"org-read","call":1}`, response.Body.String())
	require.Equal(t, http.StatusOK, performRequest(t, router, http.MethodGet, "/v1/org", nil, two.APIKey.Token).Code)

	response = performRequest(t, router, http.MethodGet, "/v1/faults/scenario", nil, one.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	var scenario domain.FaultScenario
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &scenario))
	require.Equal(t, one.ID, scenario.OrgID)
	require.Equal(t, uint64(1), scenario.Rules[0].CallCount)
	require.Equal(t, uint64(1), scenario.Rules[0].InjectedCount)

	response = performRequest(t, router, http.MethodPost, "/v1/faults/scenario/reset", nil, one.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &scenario))
	require.Zero(t, scenario.Rules[0].CallCount)
	require.Equal(t, http.StatusNotFound, performRequest(t, router, http.MethodGet, "/v1/faults/scenario", nil, two.APIKey.Token).Code)
}

func TestFaultAPIRejectsUnsafeBodies(t *testing.T) {
	router := newFaultTestRouter(t)
	org := createTestOrganization(t, router, "fault-safety")

	request := httptest.NewRequest(http.MethodPut, "/v1/faults/scenario", strings.NewReader(`{"enabled":false,"unknown":true}`))
	request.Header.Set("Authorization", "Bearer "+org.APIKey.Token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)

	request = httptest.NewRequest(http.MethodPut, "/v1/faults/scenario", bytes.NewReader(bytes.Repeat([]byte(" "), maxFaultRequestBodyBytes+1)))
	request.Header.Set("Authorization", "Bearer "+org.APIKey.Token)
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestFaultAPIAppliesBoundedLatencyWithoutReplacingResponse(t *testing.T) {
	router := newFaultTestRouter(t)
	org := createTestOrganization(t, router, "fault-latency")
	req := domain.PutFaultScenarioRequest{Enabled: true, Rules: []domain.FaultRule{{
		ID: "slow-org", Operation: "GET", Route: "/v1/org", LatencyMS: 15, EveryN: 1,
	}}}
	require.Equal(t, http.StatusOK, performRequest(t, router, http.MethodPut, "/v1/faults/scenario", req, org.APIKey.Token).Code)

	started := time.Now()
	response := performRequest(t, router, http.MethodGet, "/v1/org", nil, org.APIKey.Token)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Millisecond)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, "slow-org", response.Header().Get("X-NahCloud-Fault-Rule"))
}
