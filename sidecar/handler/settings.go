package handler

import (
	"encoding/json"
	"net/http"

	"trade-kit-sidecar/api"
)

// SetPaperMode toggles paper/demo mode for all brokers.
func (h *Handlers) SetPaperMode(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	h.registry.SetPaperMode(body.Enabled)
	api.WriteJSON(w, http.StatusOK, map[string]bool{
		"ok":         true,
		"paper_mode": body.Enabled,
	})
}
