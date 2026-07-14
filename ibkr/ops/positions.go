package ops

import (
	"encoding/json"
	"fmt"
)

// Position represents an open position.
type Position struct {
	ConID        int     `json:"conid"`
	Symbol       string  `json:"contractDesc"`
	Side         string  `json:"side"`
	Qty          float64 `json:"position"`
	AvgCost      float64 `json:"avgCost"`
	MktPrice     float64 `json:"mktPrice"`
	MktValue     float64 `json:"mktValue"`
	UnrealizedPL float64 `json:"unrealizedPnl"`
	RealizedPL   float64 `json:"realizedPnl"`
}

// GetPositions returns all open positions.
func GetPositions(c Caller) ([]Position, error) {
	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/portfolio/%s/positions/0", accountID)
	data, err := c.Get(path, nil)
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

	// Derive side from quantity.
	for i := range positions {
		if positions[i].Qty > 0 {
			positions[i].Side = "long"
		} else if positions[i].Qty < 0 {
			positions[i].Side = "short"
		}
	}

	return positions, nil
}

// ClosePosition closes a position by selling/covering the full quantity.
func ClosePosition(c Caller, symbol string, qty int) error {
	conid, err := c.ResolveConID(symbol)
	if err != nil {
		return fmt.Errorf("close_position %s: %w", symbol, err)
	}
	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/iserver/account/%s/orders", accountID)

	order := map[string]interface{}{
		"orders": []map[string]interface{}{
			{
				"conid":     conid,
				"orderType": "MKT",
				"side":      "SELL",
				"quantity":  qty,
				"tif":       "DAY",
			},
		},
	}

	_, err = c.Post(path, order)
	if err != nil {
		return fmt.Errorf("close_position %s: %w", symbol, err)
	}
	return nil
}
