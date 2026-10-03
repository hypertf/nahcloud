package domain

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	MaxFaultRules   = 100
	MaxFaultDelayMS = 2000
)

var AllowedFaultStatusCodes = map[int]bool{
	400: true, 408: true, 409: true, 423: true, 429: true,
	500: true, 502: true, 503: true, 504: true,
}

var ErrFaultRuleLimit = errors.New("fault rule limit reached")

type FaultRule struct {
	ID             string    `json:"id"`
	OrgID          string    `json:"org_id"`
	Name           string    `json:"name"`
	Enabled        bool      `json:"enabled"`
	Priority       int       `json:"priority"`
	Operation      *string   `json:"operation"`
	Route          *string   `json:"route"`
	Method         *string   `json:"method"`
	AfterMatches   uint64    `json:"after_matches"`
	EveryNth       uint64    `json:"every_nth"`
	FailurePercent int       `json:"failure_percent"`
	Seed           int64     `json:"seed"`
	MaxTriggers    *uint64   `json:"max_triggers"`
	StatusCode     *int      `json:"status_code"`
	DelayMS        int       `json:"delay_ms"`
	MatchCount     uint64    `json:"match_count"`
	TriggerCount   uint64    `json:"trigger_count"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateFaultRuleRequest struct {
	Name           string  `json:"name"`
	Enabled        *bool   `json:"enabled,omitempty"`
	Priority       *int    `json:"priority,omitempty"`
	Operation      *string `json:"operation,omitempty"`
	Route          *string `json:"route,omitempty"`
	Method         *string `json:"method,omitempty"`
	AfterMatches   *uint64 `json:"after_matches,omitempty"`
	EveryNth       *uint64 `json:"every_nth,omitempty"`
	FailurePercent *int    `json:"failure_percent,omitempty"`
	Seed           *int64  `json:"seed,omitempty"`
	MaxTriggers    *uint64 `json:"max_triggers,omitempty"`
	StatusCode     *int    `json:"status_code,omitempty"`
	DelayMS        *int    `json:"delay_ms,omitempty"`
}

// PatchField distinguishes an omitted PATCH property from an explicit null.
type PatchField[T any] struct {
	Set   bool
	Value *T
}

func (f *PatchField[T]) UnmarshalJSON(data []byte) error {
	f.Set = true
	if string(data) == "null" {
		f.Value = nil
		return nil
	}
	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	f.Value = &value
	return nil
}

type UpdateFaultRuleRequest struct {
	Name           PatchField[string] `json:"name"`
	Enabled        PatchField[bool]   `json:"enabled"`
	Priority       PatchField[int]    `json:"priority"`
	Operation      PatchField[string] `json:"operation"`
	Route          PatchField[string] `json:"route"`
	Method         PatchField[string] `json:"method"`
	AfterMatches   PatchField[uint64] `json:"after_matches"`
	EveryNth       PatchField[uint64] `json:"every_nth"`
	FailurePercent PatchField[int]    `json:"failure_percent"`
	Seed           PatchField[int64]  `json:"seed"`
	MaxTriggers    PatchField[uint64] `json:"max_triggers"`
	StatusCode     PatchField[int]    `json:"status_code"`
	DelayMS        PatchField[int]    `json:"delay_ms"`
}

type FaultDecision struct {
	RuleID     string
	MatchCount uint64
	StatusCode *int
	DelayMS    int
}
