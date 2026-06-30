package handler

import (
	"encoding/json"
	"net/http"

	"trade-kit-sidecar/api"
)

// ListBrokers returns all registered brokers with status.
func (h *Handlers) ListBrokers(w http.ResponseWriter, r *http.Request) {
	api.WriteJSON(w, http.StatusOK, h.registry.List())
}

// ConnectBroker connects a broker using provided credentials.
func (h *Handlers) ConnectBroker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adapter, err := h.registry.Get(id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	var creds map[string]string
	if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if err := adapter.Connect(creds); err != nil {
		api.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		})
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok": true,
	})
}

// TestBroker verifies a broker connection is alive.
func (h *Handlers) TestBroker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adapter, err := h.registry.Get(id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	if !adapter.Connected() {
		api.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
			"error":     "not connected",
		})
		return
	}

	if err := adapter.Test(); err != nil {
		api.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"connected": true,
	})
}

// DisconnectBroker disconnects a broker.
func (h *Handlers) DisconnectBroker(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adapter, err := h.registry.Get(id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, err.Error())
		return
	}

	adapter.Disconnect()
	api.WriteOK(w)
}
