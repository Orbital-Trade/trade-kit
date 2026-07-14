package ops

import (
	"encoding/json"
	"fmt"
)

// Position represents an open net position.
type Position struct {
	Symbol       string  `json:"tradingsymbol"`
	Exchange     string  `json:"exchange"`
	Quantity     int     `json:"quantity"`
	AvgPrice     float64 `json:"average_price"`
	LastPrice    float64 `json:"last_price"`
	PnL          float64 `json:"pnl"`
	Unrealised   float64 `json:"unrealised"`
	Realised     float64 `json:"realised"`
	BuyQty       int     `json:"buy_quantity"`
	SellQty      int     `json:"sell_quantity"`
	Product      string  `json:"product"`
}

// GetPositions returns all net positions.
func GetPositions(c Caller) ([]Position, error) {
	data, err := c.Get("/portfolio/positions", nil)
	if err != nil {
		return nil, fmt.Errorf("get_positions: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Position{}, nil
	}

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Net []Position `json:"net"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("get_positions: parse: %w", err)
	}
	if resp.Data.Net == nil {
		return []Position{}, nil
	}
	return resp.Data.Net, nil
}
