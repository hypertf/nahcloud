package service

import (
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/stretchr/testify/require"
)

type stubFaultRepository struct{}

func (stubFaultRepository) Put(_ string, req domain.PutFaultScenarioRequest) (*domain.FaultScenario, error) {
	return &domain.FaultScenario{Rules: req.Rules}, nil
}
func (stubFaultRepository) Get(string) (*domain.FaultScenario, error)   { return nil, nil }
func (stubFaultRepository) Reset(string) (*domain.FaultScenario, error) { return nil, nil }
func (stubFaultRepository) Evaluate(string, string, string) (*domain.FaultDecision, error) {
	return nil, nil
}

func TestFaultValidationSafetyBoundaries(t *testing.T) {
	svc := NewFaultService(stubFaultRepository{})
	valid := domain.FaultRule{ID: "safe", Operation: "get", Route: "/v1/projects/*", Status: 400, LatencyMS: domain.MaxFaultLatencyMS, EveryN: domain.MaxFaultEveryN}
	scenario, err := svc.Put("org", domain.PutFaultScenarioRequest{Enabled: true, Rules: []domain.FaultRule{valid}})
	require.NoError(t, err)
	require.Equal(t, "GET", scenario.Rules[0].Operation)

	cases := []domain.FaultRule{
		{ID: "status-low", Operation: "GET", Route: "/v1/org", Status: 399, EveryN: 1},
		{ID: "status-high", Operation: "GET", Route: "/v1/org", Status: 600, EveryN: 1},
		{ID: "latency", Operation: "GET", Route: "/v1/org", LatencyMS: domain.MaxFaultLatencyMS + 1, EveryN: 1},
		{ID: "frequency", Operation: "GET", Route: "/v1/org", Status: 500, EveryN: 0},
		{ID: "management", Operation: "GET", Route: "/v1/faults/scenario", Status: 500, EveryN: 1},
		{ID: "empty", Operation: "GET", Route: "/v1/org", EveryN: 1},
	}
	for _, rule := range cases {
		t.Run(rule.ID, func(t *testing.T) {
			_, err := svc.Put("org", domain.PutFaultScenarioRequest{Rules: []domain.FaultRule{rule}})
			require.Error(t, err)
			require.True(t, domain.IsInvalidInput(err))
		})
	}
}
