package domain

import "time"

const (
	MaxFaultRules     = 64
	MaxFaultLatencyMS = 5000
	MaxFaultEveryN    = 1000000
)

// FaultScenario is an explicitly enabled, organization-scoped fault plan.
type FaultScenario struct {
	OrgID     string      `json:"org_id"`
	Name      string      `json:"name"`
	Enabled   bool        `json:"enabled"`
	Seed      int64       `json:"seed"`
	Rules     []FaultRule `json:"rules"`
	UpdatedAt time.Time   `json:"updated_at"`
}

// FaultRule deterministically injects a status and/or latency into matching calls.
type FaultRule struct {
	ID            string `json:"id"`
	Operation     string `json:"operation"`
	Route         string `json:"route"`
	Status        int    `json:"status,omitempty"`
	LatencyMS     int    `json:"latency_ms,omitempty"`
	EveryN        int    `json:"every_n"`
	CallCount     uint64 `json:"call_count"`
	InjectedCount uint64 `json:"injected_count"`
}

// PutFaultScenarioRequest replaces the calling organization's complete scenario.
type PutFaultScenarioRequest struct {
	Name    string      `json:"name"`
	Enabled bool        `json:"enabled"`
	Seed    int64       `json:"seed"`
	Rules   []FaultRule `json:"rules"`
}

// FaultDecision is the result of evaluating one request.
type FaultDecision struct {
	RuleID    string
	Call      uint64
	Status    int
	LatencyMS int
}
