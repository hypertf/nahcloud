package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"strings"

	"github.com/hypertf/nahcloud/domain"
)

type FaultRepository interface {
	Create(*domain.FaultRule) error
	List(orgID string) ([]*domain.FaultRule, error)
	Get(orgID, id string) (*domain.FaultRule, error)
	Update(*domain.FaultRule) error
	Delete(orgID, id string) error
	Reset(orgID, id string) (*domain.FaultRule, error)
	ResetAll(orgID string) error
	Evaluate(orgID, operation, route, method string) (*domain.FaultDecision, error)
}

type FaultService struct{ repo FaultRepository }

func NewFaultService(repo FaultRepository) *FaultService { return &FaultService{repo: repo} }

func (s *FaultService) Create(orgID string, req domain.CreateFaultRuleRequest) (*domain.FaultRule, error) {
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, err
	}
	rule := &domain.FaultRule{
		ID: hex.EncodeToString(idBytes), OrgID: orgID, Name: req.Name, Enabled: true,
		EveryNth: 1, FailurePercent: 100,
		Operation: cleanMatcher(req.Operation), Route: cleanMatcher(req.Route), Method: upperMatcher(req.Method),
		MaxTriggers: req.MaxTriggers, StatusCode: req.StatusCode,
	}
	if req.Enabled != nil {
		rule.Enabled = *req.Enabled
	}
	if req.Priority != nil {
		rule.Priority = *req.Priority
	}
	if req.AfterMatches != nil {
		rule.AfterMatches = *req.AfterMatches
	}
	if req.EveryNth != nil {
		rule.EveryNth = *req.EveryNth
	}
	if req.FailurePercent != nil {
		rule.FailurePercent = *req.FailurePercent
	}
	if req.Seed != nil {
		rule.Seed = *req.Seed
	}
	if req.DelayMS != nil {
		rule.DelayMS = *req.DelayMS
	}
	if err := validateFaultRule(rule); err != nil {
		return nil, err
	}
	if err := s.repo.Create(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func cleanMatcher(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func upperMatcher(value *string) *string {
	value = cleanMatcher(value)
	if value == nil {
		return nil
	}
	upper := strings.ToUpper(*value)
	return &upper
}

func validateFaultRule(rule *domain.FaultRule) error {
	if len(rule.Name) > 128 {
		return domain.InvalidInputError("fault rule name is too long", nil)
	}
	if rule.Priority < 0 || rule.Priority > 1000 {
		return domain.InvalidInputError("priority must be between 0 and 1000", nil)
	}
	if rule.Operation == nil && rule.Route == nil && rule.Method == nil {
		return domain.InvalidInputError("at least one matcher is required", nil)
	}
	for field, value := range map[string]*string{"operation": rule.Operation, "route": rule.Route, "method": rule.Method} {
		if value != nil && len(*value) > 256 {
			return domain.InvalidInputError(field+" is too long", nil)
		}
	}
	if rule.Route != nil && !strings.HasPrefix(*rule.Route, "/v1/") {
		return domain.InvalidInputError("route must be an exact /v1 mux template", nil)
	}
	if rule.EveryNth < 1 {
		return domain.InvalidInputError("every_nth must be at least 1", nil)
	}
	if rule.AfterMatches > math.MaxInt64 || rule.EveryNth > math.MaxInt64 {
		return domain.InvalidInputError("fault counters must not exceed 9223372036854775807", nil)
	}
	if rule.FailurePercent < 1 || rule.FailurePercent > 100 {
		return domain.InvalidInputError("failure_percent must be between 1 and 100", nil)
	}
	if rule.MaxTriggers != nil && *rule.MaxTriggers < 1 {
		return domain.InvalidInputError("max_triggers must be null or at least 1", nil)
	}
	if rule.MaxTriggers != nil && *rule.MaxTriggers > math.MaxInt64 {
		return domain.InvalidInputError("max_triggers must not exceed 9223372036854775807", nil)
	}
	if rule.StatusCode != nil && !domain.AllowedFaultStatusCodes[*rule.StatusCode] {
		return domain.InvalidInputError("status_code is not allowlisted", nil)
	}
	if rule.DelayMS < 0 || rule.DelayMS > domain.MaxFaultDelayMS {
		return domain.InvalidInputError(fmt.Sprintf("delay_ms must be between 0 and %d", domain.MaxFaultDelayMS), nil)
	}
	if rule.StatusCode == nil && rule.DelayMS == 0 {
		return domain.InvalidInputError("at least one fault effect is required", nil)
	}
	return nil
}

func (s *FaultService) List(orgID string) ([]*domain.FaultRule, error)  { return s.repo.List(orgID) }
func (s *FaultService) Get(orgID, id string) (*domain.FaultRule, error) { return s.repo.Get(orgID, id) }

func (s *FaultService) Update(orgID, id string, req domain.UpdateFaultRuleRequest) (*domain.FaultRule, error) {
	rule, err := s.repo.Get(orgID, id)
	if err != nil {
		return nil, err
	}
	if req.Name.Set {
		if req.Name.Value == nil {
			return nil, domain.InvalidInputError("name cannot be null", nil)
		}
		rule.Name = *req.Name.Value
	}
	if req.Enabled.Set {
		if req.Enabled.Value == nil {
			return nil, domain.InvalidInputError("enabled cannot be null", nil)
		}
		rule.Enabled = *req.Enabled.Value
	}
	if req.Priority.Set {
		if req.Priority.Value == nil {
			return nil, domain.InvalidInputError("priority cannot be null", nil)
		}
		rule.Priority = *req.Priority.Value
	}
	if req.Operation.Set {
		rule.Operation = cleanMatcher(req.Operation.Value)
	}
	if req.Route.Set {
		rule.Route = cleanMatcher(req.Route.Value)
	}
	if req.Method.Set {
		rule.Method = upperMatcher(req.Method.Value)
	}
	if req.AfterMatches.Set {
		if req.AfterMatches.Value == nil {
			return nil, domain.InvalidInputError("after_matches cannot be null", nil)
		}
		rule.AfterMatches = *req.AfterMatches.Value
	}
	if req.EveryNth.Set {
		if req.EveryNth.Value == nil {
			return nil, domain.InvalidInputError("every_nth cannot be null", nil)
		}
		rule.EveryNth = *req.EveryNth.Value
	}
	if req.FailurePercent.Set {
		if req.FailurePercent.Value == nil {
			return nil, domain.InvalidInputError("failure_percent cannot be null", nil)
		}
		rule.FailurePercent = *req.FailurePercent.Value
	}
	if req.Seed.Set {
		if req.Seed.Value == nil {
			return nil, domain.InvalidInputError("seed cannot be null", nil)
		}
		rule.Seed = *req.Seed.Value
	}
	if req.MaxTriggers.Set {
		rule.MaxTriggers = req.MaxTriggers.Value
	}
	if req.StatusCode.Set {
		rule.StatusCode = req.StatusCode.Value
	}
	if req.DelayMS.Set {
		if req.DelayMS.Value == nil {
			return nil, domain.InvalidInputError("delay_ms cannot be null", nil)
		}
		rule.DelayMS = *req.DelayMS.Value
	}
	if err := validateFaultRule(rule); err != nil {
		return nil, err
	}
	if err := s.repo.Update(rule); err != nil {
		return nil, err
	}
	return rule, nil
}

func (s *FaultService) Delete(orgID, id string) error { return s.repo.Delete(orgID, id) }
func (s *FaultService) Reset(orgID, id string) (*domain.FaultRule, error) {
	return s.repo.Reset(orgID, id)
}
func (s *FaultService) ResetAll(orgID string) error { return s.repo.ResetAll(orgID) }
func (s *FaultService) Evaluate(orgID, operation, route, method string) (*domain.FaultDecision, error) {
	return s.repo.Evaluate(orgID, operation, route, method)
}
