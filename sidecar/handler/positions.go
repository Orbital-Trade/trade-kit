package handler

import (
	"net/http"

	"trade-kit-sidecar/api"
)

// GetPositions returns open positions for a broker.
func (h *Handlers) GetPositions(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	adapter, err := h.registry.Get(id)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, err.Error())
		return
	}
	if !adapter.Connected() {
		api.WriteError(w, http.StatusBadRequest, "broker not connected")
		return
	}

	positions, err := adapter.Positions()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.WriteJSON(w, http.StatusOK, positions)

	// Broadcast position update via SSE.
	h.broadcaster.Broadcast("position_update", map[string]interface{}{
		"broker_id": id,
		"positions": positions,
	})
}
