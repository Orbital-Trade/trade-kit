package handler

import (
	"encoding/json"
	"net/http"
	"os/exec"
	"path/filepath"

	"trade-kit-sidecar/api"
)

// RunBacktest executes a backtest via the backtest binary and returns JSON results.
//
//	POST /v1/backtest
//	Body: {"strategy": "bounce", "symbol": "AAPL", "from": "2025-01-01", "to": "2025-06-30"}
func (h *Handlers) RunBacktest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Strategy string `json:"strategy"`
		Symbol   string `json:"symbol"`
		From     string `json:"from"`
		To       string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Strategy == "" || req.Symbol == "" || req.From == "" {
		api.WriteError(w, http.StatusBadRequest, "strategy, symbol, and from are required")
		return
	}

	bin := filepath.Join(h.baseDir, "backtest", "backtest")
	args := []string{"run", "--strategy", req.Strategy, "--symbol", req.Symbol, "--from", req.From, "--json"}
	if req.To != "" {
		args = append(args, "--to", req.To)
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Join(h.baseDir, "backtest")
	out, err := cmd.Output()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "backtest failed: "+err.Error())
		return
	}

	// Pass through the JSON output directly.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(out)
}
