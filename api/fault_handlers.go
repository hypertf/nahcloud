package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/hypertf/nahcloud/domain"
)

const maxFaultRequestBodyBytes = 64 << 10

func (h *Handler) PutFaultScenario(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var req domain.PutFaultScenarioRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxFaultRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		h.writeError(w, domain.InvalidInputError("invalid fault scenario JSON", nil))
		return
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		h.writeError(w, domain.InvalidInputError("fault scenario body must contain one JSON value", nil))
		return
	}
	scenario, err := h.faults.Put(org.ID, req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scenario)
}

func (h *Handler) GetFaultScenario(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	scenario, err := h.faults.Get(org.ID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scenario)
}

func (h *Handler) ResetFaultScenario(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	scenario, err := h.faults.Reset(org.ID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, scenario)
}

// FaultMiddleware evaluates faults after authentication and before resource handlers.
func (h *Handler) FaultMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.faults == nil || strings.HasPrefix(r.URL.Path, "/v1/faults") {
			next.ServeHTTP(w, r)
			return
		}
		org := OrgFromContext(r.Context())
		if org == nil {
			next.ServeHTTP(w, r)
			return
		}
		decision, err := h.faults.Evaluate(org.ID, r.Method, r.URL.Path)
		if err != nil {
			h.writeError(w, err)
			return
		}
		if decision == nil {
			next.ServeHTTP(w, r)
			return
		}
		if decision.LatencyMS > 0 {
			timer := time.NewTimer(time.Duration(decision.LatencyMS) * time.Millisecond)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("X-NahCloud-Fault-Rule", decision.RuleID)
		w.Header().Set("X-NahCloud-Fault-Call", strconv.FormatUint(decision.Call, 10))
		if decision.Status != 0 {
			h.writeJSON(w, decision.Status, map[string]any{"error": "injected fault", "rule_id": decision.RuleID, "call": decision.Call})
			return
		}
		next.ServeHTTP(w, r)
	})
}
