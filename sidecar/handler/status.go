package handler

import (
	"net/http"
	"time"

	"trade-kit-sidecar/api"
)

type statusResponse struct {
	Version       string         `json:"version"`
	Uptime        string         `json:"uptime"`
	PaperMode     bool           `json:"paper_mode"`
	Brokers       []brokerStatus `json:"brokers"`
	RecipesRunning int           `json:"recipes_running"`
}

type brokerStatus struct {
	ID        string `json:"id"`
	Connected bool   `json:"connected"`
}

// startTime is set when the handler package is initialized.
var startTime = time.Now()

// GetStatus returns server health and connection status.
func (h *Handlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	brokers := h.registry.List()
	bs := make([]brokerStatus, len(brokers))
	for i, b := range brokers {
		bs[i] = brokerStatus{ID: b.ID, Connected: b.Connected}
	}

	api.WriteJSON(w, http.StatusOK, statusResponse{
		Version:        h.srv.Version(),
		Uptime:         time.Since(startTime).Round(time.Second).String(),
		PaperMode:      h.registry.IsPaper(),
		Brokers:        bs,
		RecipesRunning: 0,
	})
}

// Kill stops all brokers and shuts down the server.
func (h *Handlers) Kill(w http.ResponseWriter, r *http.Request) {
	// Disconnect all brokers.
	for _, b := range h.registry.List() {
		if b.Connected {
			adapter, _ := h.registry.Get(b.ID)
			if adapter != nil {
				adapter.Disconnect()
			}
		}
	}

	api.WriteOK(w)

	// Trigger graceful shutdown after response is sent.
	go h.srv.Shutdown()
}
