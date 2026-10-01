package service

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/hypertf/nahcloud/domain"
)

// FaultRepository persists scenarios and atomically advances matching counters.
type FaultRepository interface {
	Put(orgID string, req domain.PutFaultScenarioRequest) (*domain.FaultScenario, error)
	Get(orgID string) (*domain.FaultScenario, error)
	Reset(orgID string) (*domain.FaultScenario, error)
	Evaluate(orgID, operation, route string) (*domain.FaultDecision, error)
}

// FaultService owns validation and deterministic fault evaluation.
type FaultService struct{ repo FaultRepository }

var faultRuleIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func NewFaultService(repo FaultRepository) *FaultService { return &FaultService{repo: repo} }

func (s *FaultService) Put(orgID string, req domain.PutFaultScenarioRequest) (*domain.FaultScenario, error) {
	if len(req.Name) > 128 {
		return nil, domain.InvalidInputError("fault scenario name is too long", nil)
	}
	if req.Seed < 0 {
		return nil, domain.InvalidInputError("fault scenario seed cannot be negative", nil)
	}
	if len(req.Rules) > domain.MaxFaultRules {
		return nil, domain.InvalidInputError("too many fault rules", map[string]interface{}{"max": domain.MaxFaultRules})
	}
	seen := make(map[string]bool, len(req.Rules))
	for i := range req.Rules {
		rule := &req.Rules[i]
		rule.Operation = strings.ToUpper(rule.Operation)
		if len(rule.ID) > 64 || !faultRuleIDPattern.MatchString(rule.ID) || seen[rule.ID] {
			return nil, domain.InvalidInputError("fault rule IDs must be unique 1-64 character alphanumeric identifiers", nil)
		}
		seen[rule.ID] = true
		if rule.Operation == "" || len(rule.Operation) > 16 {
			return nil, domain.InvalidInputError("fault rule operation is required and must be at most 16 characters", nil)
		}
		if !validFaultRoute(rule.Route) {
			return nil, domain.InvalidInputError("fault rule route must be an exact /v1 path or end in *", nil)
		}
		if strings.HasPrefix(rule.Route, "/v1/faults") {
			return nil, domain.InvalidInputError("fault management routes cannot be targeted", nil)
		}
		if rule.Status != 0 && (rule.Status < http.StatusBadRequest || rule.Status > 599) {
			return nil, domain.InvalidInputError("fault status must be zero or between 400 and 599", nil)
		}
		if rule.LatencyMS < 0 || rule.LatencyMS > domain.MaxFaultLatencyMS {
			return nil, domain.InvalidInputError(fmt.Sprintf("fault latency_ms must be between 0 and %d", domain.MaxFaultLatencyMS), nil)
		}
		if rule.Status == 0 && rule.LatencyMS == 0 {
			return nil, domain.InvalidInputError("fault rule must inject a status or latency", nil)
		}
		if rule.EveryN < 1 || rule.EveryN > domain.MaxFaultEveryN {
			return nil, domain.InvalidInputError(fmt.Sprintf("fault every_n must be between 1 and %d", domain.MaxFaultEveryN), nil)
		}
		rule.CallCount, rule.InjectedCount = 0, 0
	}
	return s.repo.Put(orgID, req)
}

func validFaultRoute(route string) bool {
	if !strings.HasPrefix(route, "/v1/") || len(route) > 256 || strings.ContainsAny(route, "?#") {
		return false
	}
	return !strings.Contains(strings.TrimSuffix(route, "*"), "*")
}

func (s *FaultService) Get(orgID string) (*domain.FaultScenario, error) {
	return s.repo.Get(orgID)
}

func (s *FaultService) Reset(orgID string) (*domain.FaultScenario, error) {
	return s.repo.Reset(orgID)
}

func (s *FaultService) Evaluate(orgID, operation, route string) (*domain.FaultDecision, error) {
	return s.repo.Evaluate(orgID, operation, route)
}
