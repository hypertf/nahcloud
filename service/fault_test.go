package service

import (
	"math"
	"testing"

	"github.com/hypertf/nahcloud/domain"
	"github.com/stretchr/testify/require"
)

func TestFaultRuleRejectsValuesOutsideSQLiteIntegerRange(t *testing.T) {
	tooLarge := uint64(math.MaxInt64) + 1
	err := validateFaultRule(&domain.FaultRule{Method: faultString("GET"), StatusCode: faultInt(503), EveryNth: tooLarge, FailurePercent: 100})
	require.ErrorContains(t, err, "must not exceed")
}

func faultString(value string) *string { return &value }
func faultInt(value int) *int          { return &value }
func faultUint(value uint64) *uint64   { return &value }

func TestFaultRuleValidationBoundaries(t *testing.T) {
	valid := domain.FaultRule{Operation: faultString("instances.create"), EveryNth: 1, FailurePercent: 100, StatusCode: faultInt(400), DelayMS: domain.MaxFaultDelayMS, Priority: 1000}
	require.NoError(t, validateFaultRule(&valid))
	for status := range domain.AllowedFaultStatusCodes {
		valid.StatusCode = faultInt(status)
		require.NoError(t, validateFaultRule(&valid))
	}

	cases := map[string]domain.FaultRule{
		"matcher required":  {EveryNth: 1, FailurePercent: 100, StatusCode: faultInt(500)},
		"effect required":   {Method: faultString("GET"), EveryNth: 1, FailurePercent: 100},
		"priority low":      {Method: faultString("GET"), Priority: -1, EveryNth: 1, FailurePercent: 100, StatusCode: faultInt(500)},
		"priority high":     {Method: faultString("GET"), Priority: 1001, EveryNth: 1, FailurePercent: 100, StatusCode: faultInt(500)},
		"every nth":         {Method: faultString("GET"), EveryNth: 0, FailurePercent: 100, StatusCode: faultInt(500)},
		"percent low":       {Method: faultString("GET"), EveryNth: 1, FailurePercent: 0, StatusCode: faultInt(500)},
		"percent high":      {Method: faultString("GET"), EveryNth: 1, FailurePercent: 101, StatusCode: faultInt(500)},
		"max triggers":      {Method: faultString("GET"), EveryNth: 1, FailurePercent: 100, MaxTriggers: faultUint(0), StatusCode: faultInt(500)},
		"status allowlist":  {Method: faultString("GET"), EveryNth: 1, FailurePercent: 100, StatusCode: faultInt(501)},
		"delay low":         {Method: faultString("GET"), EveryNth: 1, FailurePercent: 100, DelayMS: -1},
		"delay high":        {Method: faultString("GET"), EveryNth: 1, FailurePercent: 100, DelayMS: domain.MaxFaultDelayMS + 1},
		"route must be API": {Route: faultString("/buildz"), EveryNth: 1, FailurePercent: 100, StatusCode: faultInt(500)},
	}
	for name, rule := range cases {
		t.Run(name, func(t *testing.T) {
			err := validateFaultRule(&rule)
			require.Error(t, err)
			require.True(t, domain.IsInvalidInput(err))
		})
	}
}
