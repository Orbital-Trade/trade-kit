package handler

import (
	"net/http"

	"trade-kit-sidecar/api"
)

// GetOrders returns pending orders for a broker.
func (h *Handlers) GetOrders(w http.ResponseWriter, r *http.Request) {
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

	orders, err := adapter.Orders()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	api.WriteJSON(w, http.StatusOK, orders)
}
