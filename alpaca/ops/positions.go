package ops

import (
	"encoding/json"
	"fmt"
)

// Position represents an open position.
type Position struct {
	Symbol        string  `json:"symbol"`
	Qty           float64 `json:"qty,string"`
	Side          string  `json:"side"` // "long" or "short"
	AvgEntryPrice float64 `json:"avg_entry_price,string"`
	CurrentPrice  float64 `json:"current_price,string"`
	MarketValue   float64 `json:"market_value,string"`
	UnrealizedPL  float64 `json:"unrealized_pl,string"`
	UnrealizedPct float64 `json:"unrealized_plpc,string"`
	AssetClass    string  `json:"asset_class"`
}

// GetPositions returns all open positions.
func GetPositions(c Caller) ([]Position, error) {
	data, err := c.Get("/v2/positions", nil)
	if err != nil {
		return nil, fmt.Errorf("get_positions: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Position{}, nil
	}

	var positions []Position
	if err := json.Unmarshal(data, &positions); err != nil {
		return nil, fmt.Errorf("get_positions: parse: %w", err)
	}
	return positions, nil
}

// ClosePosition closes a position by symbol.
func ClosePosition(c Caller, symbol string) error {
	_, err := c.Delete("/v2/positions/"+symbol, nil)
	if err != nil {
		return fmt.Errorf("close_position %s: %w", symbol, err)
	}
	return nil
}
