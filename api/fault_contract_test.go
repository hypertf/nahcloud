package api

import (
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
	return SetupRouter(NewHandler(svc, service.NewFaultService(sqlite.NewFaultRepository(db))), svc, "test")
}

func createFaultRule(t *testing.T, router http.Handler, token string, body any) domain.FaultRule {
	t.Helper()
	response := performRequest(t, router, http.MethodPost, "/v1/fault-rules", body, token)
	require.Equal(t, http.StatusCreated, response.Code, response.Body.String())
	var rule domain.FaultRule
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &rule))
	require.Len(t, rule.ID, 32)
	return rule
}

func TestFaultRuleCRUDMatchingAndTenantIsolation(t *testing.T) {
	router := newFaultTestRouter(t)
	one := createTestOrganization(t, router, "fault-one")
	two := createTestOrganization(t, router, "fault-two")
	rule := createFaultRule(t, router, one.APIKey.Token, map[string]any{
		"name": "org outage", "operation": "organizations.get", "route": "/v1/org", "method": "get", "status_code": 503,
	})
	require.Equal(t, one.ID, rule.OrgID)
	require.True(t, rule.Enabled)
	require.Equal(t, uint64(1), rule.EveryNth)
	require.Equal(t, 100, rule.FailurePercent)

	require.Equal(t, http.StatusUnauthorized, performRequest(t, router, http.MethodGet, "/v1/org", nil, "invalid-token").Code)
	response := performRequest(t, router, http.MethodGet, "/v1/org", nil, one.APIKey.Token)
	require.Equal(t, http.StatusServiceUnavailable, response.Code, response.Body.String())
	require.Equal(t, rule.ID, response.Header().Get("X-NahCloud-Fault-Rule"))
	require.JSONEq(t, `{"error":"injected fault","rule_id":"`+rule.ID+`","match_count":1}`, response.Body.String())
	require.Equal(t, http.StatusOK, performRequest(t, router, http.MethodGet, "/v1/org", nil, two.APIKey.Token).Code)

	response = performRequest(t, router, http.MethodGet, "/v1/fault-rules", nil, one.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code)
	var rules []domain.FaultRule
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &rules))
	require.Len(t, rules, 1)
	require.Equal(t, uint64(1), rules[0].MatchCount)
	require.JSONEq(t, `[]`, performRequest(t, router, http.MethodGet, "/v1/fault-rules", nil, two.APIKey.Token).Body.String())
	require.Equal(t, http.StatusNotFound, performRequest(t, router, http.MethodGet, "/v1/fault-rules/"+rule.ID, nil, two.APIKey.Token).Code)

	response = performRequest(t, router, http.MethodPatch, "/v1/fault-rules/"+rule.ID, map[string]any{"enabled": false, "status_code": nil, "delay_ms": 1}, one.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.Equal(t, http.StatusOK, performRequest(t, router, http.MethodGet, "/v1/org", nil, one.APIKey.Token).Code)
	response = performRequest(t, router, http.MethodPatch, "/v1/fault-rules/"+rule.ID, map[string]any{"enabled": true, "delay_ms": 15}, one.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	started := time.Now()
	require.Equal(t, http.StatusOK, performRequest(t, router, http.MethodGet, "/v1/org", nil, one.APIKey.Token).Code)
	require.GreaterOrEqual(t, time.Since(started), 10*time.Millisecond)
	response = performRequest(t, router, http.MethodPost, "/v1/fault-rules/"+rule.ID+"/reset", nil, one.APIKey.Token)
	require.Equal(t, http.StatusOK, response.Code, response.Body.String())
	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &rule))
	require.Zero(t, rule.MatchCount)
	require.Zero(t, rule.TriggerCount)
	require.Equal(t, http.StatusNoContent, performRequest(t, router, http.MethodPost, "/v1/fault-rules/reset", nil, one.APIKey.Token).Code)
	require.Equal(t, http.StatusNoContent, performRequest(t, router, http.MethodDelete, "/v1/fault-rules/"+rule.ID, nil, one.APIKey.Token).Code)
	require.Equal(t, http.StatusNotFound, performRequest(t, router, http.MethodGet, "/v1/fault-rules/"+rule.ID, nil, one.APIKey.Token).Code)
}

func TestFaultExemptionsAndTerraformStateParticipation(t *testing.T) {
	router := newFaultTestRouter(t)
	org := createTestOrganization(t, router, "fault-exempt")
	// A broad POST matcher cannot block fault management.
	createFaultRule(t, router, org.APIKey.Token, map[string]any{"method": "POST", "status_code": 500})
	require.Equal(t, http.StatusNoContent, performRequest(t, router, http.MethodPost, "/v1/fault-rules/reset", nil, org.APIKey.Token).Code)

	// TF-state is authenticated before interception and exposes named route metadata.
	createFaultRule(t, router, org.APIKey.Token, map[string]any{"priority": 0, "operation": "tfstate.get", "route": "/v1/tfstate/{id}", "method": "GET", "status_code": 423})
	req := httptest.NewRequest(http.MethodGet, "/v1/tfstate/demo", nil)
	req.SetBasicAuth("nah", org.APIKey.Token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, req)
	require.Equal(t, http.StatusLocked, response.Code, response.Body.String())

	// Organization reset is exempt and removes rules atomically.
	require.Equal(t, http.StatusNoContent, performRequest(t, router, http.MethodPost, "/v1/org/reset", nil, org.APIKey.Token).Code)
	require.JSONEq(t, `[]`, performRequest(t, router, http.MethodGet, "/v1/fault-rules", nil, org.APIKey.Token).Body.String())
	require.Equal(t, http.StatusOK, performRequest(t, router, http.MethodGet, "/buildz", nil, "").Code)
}

func TestFaultRuleRequestSafety(t *testing.T) {
	router := newFaultTestRouter(t)
	org := createTestOrganization(t, router, "fault-safety")
	for _, body := range []any{
		map[string]any{"status_code": 500},
		map[string]any{"method": "GET"},
		map[string]any{"method": "GET", "status_code": 501},
		map[string]any{"method": "GET", "delay_ms": 2001},
		map[string]any{"method": "GET", "status_code": 500, "unknown": true},
		map[string]any{"method": "GET", "status_code": 500, "id": "forged"},
	} {
		response := performRequest(t, router, http.MethodPost, "/v1/fault-rules", body, org.APIKey.Token)
		require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/fault-rules", strings.NewReader(strings.Repeat(" ", maxFaultRequestBodyBytes+1)))
	request.Header.Set("Authorization", "Bearer "+org.APIKey.Token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusBadRequest, response.Code)
}

func TestFaultRuleAPILimit(t *testing.T) {
	router := newFaultTestRouter(t)
	org := createTestOrganization(t, router, "fault-limit")
	for i := 0; i < domain.MaxFaultRules; i++ {
		createFaultRule(t, router, org.APIKey.Token, map[string]any{"method": "OPTIONS", "status_code": 500})
	}
	response := performRequest(t, router, http.MethodPost, "/v1/fault-rules", map[string]any{"method": "OPTIONS", "status_code": 500}, org.APIKey.Token)
	require.Equal(t, http.StatusTooManyRequests, response.Code, response.Body.String())
}
