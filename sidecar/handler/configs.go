package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"trade-kit-sidecar/api"
)

// configPaths maps recipe IDs to their config file paths relative to trade-kit root.
var configPaths = map[string]string{
	"daytrader": "daytrader/daytrader.json",
	"bounce":    "bounce/bounce.json",
	"earnings":  "earnings/earnings.json",
	"index":     "index/index.json",
	"alert":     "alert/alert.json",
	"notifier":  "notifier/notifier.json",
	"controller": "controller/controller.json",
	"backtest":  "backtest/backtest.json",
}

// GetConfig returns a recipe's config JSON.
//
//	GET /v1/configs/{id}
func (h *Handlers) GetConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rel, ok := configPaths[id]
	if !ok {
		api.WriteError(w, http.StatusNotFound, "unknown config: "+id)
		return
	}

	path := filepath.Join(h.baseDir, rel)
	data, err := os.ReadFile(path)
	if err != nil {
		api.WriteError(w, http.StatusNotFound, "config not found: "+rel)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// UpdateConfig writes a recipe's config JSON.
//
//	PUT /v1/configs/{id}
//	Body: the full JSON config to write
func (h *Handlers) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	rel, ok := configPaths[id]
	if !ok {
		api.WriteError(w, http.StatusNotFound, "unknown config: "+id)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "read body: "+err.Error())
		return
	}

	// Validate it's valid JSON.
	var check json.RawMessage
	if err := json.Unmarshal(body, &check); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Pretty-print before writing.
	var pretty json.RawMessage
	if err := json.Unmarshal(body, &pretty); err == nil {
		indented, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			body = append(indented, '\n')
		}
	}

	path := filepath.Join(h.baseDir, rel)
	if err := os.WriteFile(path, body, 0644); err != nil {
		api.WriteError(w, http.StatusInternalServerError, "write config: "+err.Error())
		return
	}

	api.WriteOK(w)
}
