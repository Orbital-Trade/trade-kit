package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"

	"trade-kit-sidecar/api"
)

// GetJournal returns trade history from the journal.
//
//	GET /v1/journal?symbol=AAPL&days=30
func (h *Handlers) GetJournal(w http.ResponseWriter, r *http.Request) {
	bin := filepath.Join(h.baseDir, "journal", "journal")

	args := []string{"list"}
	if sym := r.URL.Query().Get("symbol"); sym != "" {
		args = append(args, "--symbol", sym)
	}
	if days := r.URL.Query().Get("days"); days != "" {
		args = append(args, "--days", days)
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Join(h.baseDir, "journal")
	out, err := cmd.Output()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "journal list: "+err.Error())
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{
		"output": strings.TrimSpace(string(out)),
	})
}

// GetJournalPnL returns P&L summary from the journal.
//
//	GET /v1/journal/pnl?symbol=AAPL
func (h *Handlers) GetJournalPnL(w http.ResponseWriter, r *http.Request) {
	bin := filepath.Join(h.baseDir, "journal", "journal")

	args := []string{"pnl"}
	if sym := r.URL.Query().Get("symbol"); sym != "" {
		args = append(args, "--symbol", sym)
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Join(h.baseDir, "journal")
	out, err := cmd.Output()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "journal pnl: "+err.Error())
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]string{
		"output": strings.TrimSpace(string(out)),
	})
}

// AddJournalEntry records a trade in the journal.
//
//	POST /v1/journal
//	Body: {"side": "BUY", "symbol": "AAPL", "qty": 10, "price": 185.50, "strategy": "daytrader", "note": "demo"}
func (h *Handlers) AddJournalEntry(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Side     string  `json:"side"`
		Symbol   string  `json:"symbol"`
		Qty      int     `json:"qty"`
		Price    float64 `json:"price"`
		Strategy string  `json:"strategy"`
		Note     string  `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.WriteError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Side == "" || req.Symbol == "" || req.Qty <= 0 || req.Price <= 0 {
		api.WriteError(w, http.StatusBadRequest, "side, symbol, qty, and price are required")
		return
	}

	bin := filepath.Join(h.baseDir, "journal", "journal")
	args := []string{"add", strings.ToUpper(req.Side), req.Symbol,
		fmt.Sprintf("%d", req.Qty), fmt.Sprintf("%.4f", req.Price)}
	if req.Strategy != "" {
		args = append(args, "--strategy", req.Strategy)
	}
	if req.Note != "" {
		args = append(args, "--note", req.Note)
	}

	cmd := exec.Command(bin, args...)
	cmd.Dir = filepath.Join(h.baseDir, "journal")
	out, err := cmd.Output()
	if err != nil {
		api.WriteError(w, http.StatusInternalServerError, "journal add: "+err.Error())
		return
	}

	api.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"ok":     true,
		"output": strings.TrimSpace(string(out)),
	})
}
