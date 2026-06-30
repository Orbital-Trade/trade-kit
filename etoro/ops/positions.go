package ops

// GetPositions — returns open positions via eToro Trading API.
//
// Uses demo or real trading endpoints depending on client mode.
// Demo: GET /api/v1/trading/demo/portfolio
// Real: GET /api/v1/trading/real/portfolio

import (
	"encoding/json"
	"fmt"
)

// Position represents a single open position.
type Position struct {
	PositionID    string  `json:"position_id"`
	InstrumentID  int     `json:"instrument_id"`
	Symbol        string  `json:"symbol"`
	IsBuy         bool    `json:"is_buy"`
	Amount        float64 `json:"amount"`         // invested amount
	Units         float64 `json:"units"`           // number of units/shares
	OpenRate      float64 `json:"open_rate"`       // entry price
	CurrentRate   float64 `json:"current_rate"`    // current price
	StopLoss      float64 `json:"stop_loss"`
	TakeProfit    float64 `json:"take_profit"`
	PnL           float64 `json:"pnl"`
	PnLPct        float64 `json:"pnl_pct"`
}

// etoroPosition maps the raw eToro API response fields.
type etoroPosition struct {
	PositionID      json.Number `json:"PositionID"`
	InstrumentID    int         `json:"InstrumentID"`
	IsBuy           bool        `json:"IsBuy"`
	Amount          float64     `json:"Amount"`
	Units           float64     `json:"Units"`
	OpenRate        float64     `json:"OpenRate"`
	CurrentRate     float64     `json:"CurrentRate"`
	StopLossRate    float64     `json:"StopLossRate"`
	TakeProfitRate  float64     `json:"TakeProfitRate"`
	NetProfit       float64     `json:"NetProfit"`
}

// GetPositions returns all open positions for the account.
func GetPositions(c Caller) ([]Position, error) {
	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	data, err := c.Get(prefix+"/portfolio", nil)
	if err != nil {
		return nil, fmt.Errorf("get_positions: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Position{}, nil
	}

	var raw []etoroPosition
	if err := json.Unmarshal(data, &raw); err != nil {
		// Try wrapped response.
		var wrapper struct {
			Positions []etoroPosition `json:"positions"`
		}
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return nil, fmt.Errorf("get_positions: parse: %w", err)
		}
		raw = wrapper.Positions
	}

	out := make([]Position, 0, len(raw))
	for _, r := range raw {
		var pnlPct float64
		if r.Amount > 0 {
			pnlPct = r.NetProfit / r.Amount * 100
		}
		out = append(out, Position{
			PositionID:   r.PositionID.String(),
			InstrumentID: r.InstrumentID,
			IsBuy:        r.IsBuy,
			Amount:       r.Amount,
			Units:        r.Units,
			OpenRate:     r.OpenRate,
			CurrentRate:  r.CurrentRate,
			StopLoss:     r.StopLossRate,
			TakeProfit:   r.TakeProfitRate,
			PnL:          r.NetProfit,
			PnLPct:       pnlPct,
		})
	}

	// Resolve symbols for display.
	for i := range out {
		name, err := resolveSymbolName(c, out[i].InstrumentID)
		if err == nil {
			out[i].Symbol = name
		}
	}

	return out, nil
}

// resolveSymbolName looks up a symbol name from an instrument ID via cache.
// Falls back to the instrument ID as string if lookup fails.
func resolveSymbolName(c Caller, instrID int) (string, error) {
	cacheMu.RLock()
	for _, inst := range cache {
		if inst.ID == instrID {
			cacheMu.RUnlock()
			return inst.Symbol, nil
		}
	}
	cacheMu.RUnlock()
	return fmt.Sprintf("ID:%d", instrID), nil
}
