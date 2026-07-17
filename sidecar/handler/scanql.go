package handler

import (
	"encoding/json"
	"net/http"

	"trade-kit-sidecar/api"
	"trade-kit-sidecar/scanql"
)

// ValidateScanQL validates ScanQL syntax without running it.
func (h *Handlers) ValidateScanQL(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ScanQL string `json:"scanql"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	_, err := scanql.Parse(req.ScanQL)
	if err != nil {
		api.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}
	api.WriteJSON(w, http.StatusOK, map[string]bool{"valid": true})
}

// StartCustomRecipe parses ScanQL and starts it as a recipe.
func (h *Handlers) StartCustomRecipe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name   string `json:"name"`
		ScanQL string `json:"scanql"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	plan, err := scanql.Parse(req.ScanQL)
	if err != nil {
		api.WriteError(w, http.StatusBadRequest, "parse error: "+err.Error())
		return
	}

	if req.Name != "" {
		plan.Name = req.Name
	}

	if err := h.runner.StartCustom(plan.Name, scanql.Run(plan)); err != nil {
		api.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	api.WriteOK(w)
}

// ScanQLExamples returns example .scan files.
func (h *Handlers) ScanQLExamples(w http.ResponseWriter, r *http.Request) {
	examples := []map[string]string{
		{
			"name":        "rsi_bounce",
			"description": "RSI oversold bounce — enters when RSI drops below 25",
			"scanql": "SCAN rsi_bounce\n  EVERY 300s\n  SYMBOLS AAPL, NVDA, TSLA, RKLB\n  FETCH quote, rsi(14)\n  WHERE rsi <= 25\n    AND volume >= 500000\n  ENTER LONG\n    STOP 5%\n    TARGET 3R\n    BUDGET 150",
		},
		{
			"name":        "gap_scalp",
			"description": "Gap-up day trade — enters on 3-20% pre-market gap with volume",
			"scanql": "SCAN gap_scalp\n  EVERY 60s\n  SYMBOLS LUNR, RKLB, ASTS, IONQ, NVDA\n  FETCH quote, rvol\n  WHERE gap_pct BETWEEN 3.0 AND 20.0\n    AND rvol >= 1.5\n    AND volume >= 500000\n  ENTER LONG\n    STOP 2%\n    TARGET 3R\n    BUDGET 200",
		},
		{
			"name":        "vix_momentum",
			"description": "QQQ/VIX momentum — trades TQQQ when QQQ is up with low VIX",
			"scanql": "SCAN vix_momentum\n  EVERY 30s\n  SYMBOLS QQQ\n  FETCH quote\n  WHERE change_pct >= 0.3\n  ENTER LONG TQQQ 2 SHARES\n    STOP 5%\n    TARGET 6%\n    EXIT BY 12:30 ET",
		},
	}
	api.WriteJSON(w, http.StatusOK, examples)
}
