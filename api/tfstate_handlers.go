package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hypertf/nahcloud/domain"
)

// TFStateGet handles GET /v1/tfstate/{state_id}
func (h *Handler) TFStateGet(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]

	state, err := h.service.GetTFState(org.ID, id)
	if err != nil {
		if domain.IsNotFound(err) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		h.writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(state))
}

// TFStatePost handles POST /v1/tfstate/{state_id}
func (h *Handler) TFStatePost(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]

	// Enforce lock if present
	rawLock, lockInfo, err := h.service.GetTFStateLock(org.ID, id)
	if err != nil && !domain.IsNotFound(err) {
		h.writeError(w, err)
		return
	}
	if err == nil {
		provided := r.URL.Query().Get("ID")
		if lockInfo == nil || provided == "" || provided != lockInfo.ID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusLocked) // 423
			w.Write([]byte(rawLock))
			return
		}
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, domain.InternalError("failed to read request body"))
		return
	}
	if err := h.service.SetTFState(org.ID, id, string(body)); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// TFStateDelete handles DELETE /v1/tfstate/{state_id}
func (h *Handler) TFStateDelete(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]

	// Enforce lock if present
	rawLock, lockInfo, err := h.service.GetTFStateLock(org.ID, id)
	if err != nil && !domain.IsNotFound(err) {
		h.writeError(w, err)
		return
	}
	if err == nil {
		provided := r.URL.Query().Get("ID")
		if lockInfo == nil || provided == "" || provided != lockInfo.ID {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusLocked) // 423
			w.Write([]byte(rawLock))
			return
		}
	}

	if err := h.service.DeleteTFState(org.ID, id); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// TFStateLock handles LOCK /v1/tfstate/{state_id}
func (h *Handler) TFStateLock(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, domain.InternalError("failed to read request body"))
		return
	}
	// Validate lock JSON minimally to extract ID
	var li domain.TFStateLock
	if err := json.Unmarshal(body, &li); err != nil || li.ID == "" {
		h.writeError(w, domain.InvalidInputError("invalid lock payload: missing or invalid ID", nil))
		return
	}

	// Try to place the lock
	locked, existing, err := h.service.TryLockTFState(org.ID, id, string(body))
	if err != nil {
		h.writeError(w, err)
		return
	}
	if locked {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusLocked) // 423
		w.Write([]byte(existing))
		return
	}
	w.WriteHeader(http.StatusOK)
}

// TFStateUnlock handles UNLOCK /v1/tfstate/{state_id}
func (h *Handler) TFStateUnlock(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		h.writeError(w, err)
		return
	}
	vars := mux.Vars(r)
	id := vars["id"]

	provided := r.URL.Query().Get("ID")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, domain.InvalidInputError("failed to read unlock payload", nil))
		return
	}
	if len(body) > 0 {
		var requested domain.TFStateLock
		if err := json.Unmarshal(body, &requested); err != nil || requested.ID == "" {
			h.writeError(w, domain.InvalidInputError("invalid unlock payload: missing or invalid ID", nil))
			return
		}
		provided = requested.ID
	}
	if provided == "" {
		h.writeError(w, domain.InvalidInputError("unlock payload must include an ID", nil))
		return
	}

	// Get current lock
	rawLock, lockInfo, err := h.service.GetTFStateLock(org.ID, id)
	if err != nil {
		if domain.IsNotFound(err) {
			w.WriteHeader(http.StatusOK)
			return
		}
		h.writeError(w, err)
		return
	}
	if lockInfo != nil && provided == lockInfo.ID {
		if _, _, err := h.service.UnlockTFState(org.ID, id); err != nil {
			h.writeError(w, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	// Mismatch
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict) // 409
	w.Write([]byte(rawLock))
}
