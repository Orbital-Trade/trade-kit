package ops

import (
	"encoding/json"
	"fmt"
)

// Holding represents a long-term DEMAT holding.
type Holding struct {
	Symbol          string  `json:"tradingsymbol"`
	Exchange        string  `json:"exchange"`
	ISIN            string  `json:"isin"`
	Quantity        int     `json:"quantity"`
	T1Quantity      int     `json:"t1_quantity"`
	AveragePrice    float64 `json:"average_price"`
	LastPrice       float64 `json:"last_price"`
	PnL             float64 `json:"pnl"`
	ClosePrice      float64 `json:"close_price"`
	DayChange       float64 `json:"day_change"`
	DayChangePct    float64 `json:"day_change_percentage"`
}

// GetHoldings returns all DEMAT holdings.
func GetHoldings(c Caller) ([]Holding, error) {
	data, err := c.Get("/portfolio/holdings", nil)
	if err != nil {
		return nil, fmt.Errorf("get_holdings: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Holding{}, nil
	}

	var resp struct {
		Status string    `json:"status"`
		Data   []Holding `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("get_holdings: parse: %w", err)
	}
	if resp.Data == nil {
		return []Holding{}, nil
	}
	return resp.Data, nil
}
