package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
)

const maxFaultRequestBodyBytes = 64 << 10

func (h *Handler) decodeFaultJSON(w http.ResponseWriter, r *http.Request, value any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxFaultRequestBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return domain.InvalidInputError("invalid fault rule JSON", nil)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return domain.InvalidInputError("body must contain one JSON value", nil)
	}
	return nil
}

func (h *Handler) CreateFaultRule(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var req domain.CreateFaultRuleRequest
	if err := h.decodeFaultJSON(w, r, &req); err != nil {
		h.writeError(w, err)
		return
	}
	rule, err := h.faults.Create(org.ID, req)
	if errors.Is(err, domain.ErrFaultRuleLimit) {
		h.writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "fault rule limit reached"})
		return
	}
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, rule)
}

func (h *Handler) ListFaultRules(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	rules, err := h.faults.List(org.ID)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rules)
}

func (h *Handler) GetFaultRule(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	rule, err := h.faults.Get(org.ID, mux.Vars(r)["id"])
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) UpdateFaultRule(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	var req domain.UpdateFaultRuleRequest
	if err := h.decodeFaultJSON(w, r, &req); err != nil {
		h.writeError(w, err)
		return
	}
	rule, err := h.faults.Update(org.ID, mux.Vars(r)["id"], req)
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) DeleteFaultRule(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if err := h.faults.Delete(org.ID, mux.Vars(r)["id"]); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ResetFaultRule(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	rule, err := h.faults.Reset(org.ID, mux.Vars(r)["id"])
	if err != nil {
		h.writeError(w, err)
		return
	}
	h.writeJSON(w, http.StatusOK, rule)
}

func (h *Handler) ResetAllFaultRules(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if err := h.faults.ResetAll(org.ID); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) FaultMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.faults == nil || strings.HasPrefix(r.URL.Path, "/v1/fault-rules") || r.URL.Path == "/v1/org/reset" {
			next.ServeHTTP(w, r)
			return
		}
		org := OrgFromContext(r.Context())
		if org == nil {
			next.ServeHTTP(w, r)
			return
		}
		operation, routeTemplate := "", ""
		if route := mux.CurrentRoute(r); route != nil {
			operation = route.GetName()
			routeTemplate, _ = route.GetPathTemplate()
		}
		decision, err := h.faults.Evaluate(org.ID, operation, routeTemplate, r.Method)
		if err != nil {
			h.writeError(w, err)
			return
		}
		if decision == nil {
			next.ServeHTTP(w, r)
			return
		}
		if decision.DelayMS > 0 {
			timer := time.NewTimer(time.Duration(decision.DelayMS) * time.Millisecond)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-r.Context().Done():
				return
			}
		}
		w.Header().Set("X-NahCloud-Fault-Rule", decision.RuleID)
		w.Header().Set("X-NahCloud-Fault-Match", strconv.FormatUint(decision.MatchCount, 10))
		if decision.StatusCode != nil {
			h.writeJSON(w, *decision.StatusCode, map[string]any{"error": "injected fault", "rule_id": decision.RuleID, "match_count": decision.MatchCount})
			return
		}
		next.ServeHTTP(w, r)
	})
}
