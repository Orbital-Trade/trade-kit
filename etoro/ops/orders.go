package ops

// GetOrders — returns open/pending orders via eToro Trading API.
//
// Demo: GET /api/v1/trading/demo/orders
// Real: GET /api/v1/trading/real/orders

import (
	"encoding/json"
	"fmt"
)

// Order represents an open or pending order.
type Order struct {
	OrderID      string  `json:"order_id"`
	InstrumentID int     `json:"instrument_id"`
	Symbol       string  `json:"symbol"`
	IsBuy        bool    `json:"is_buy"`
	Amount       float64 `json:"amount"`
	Units        float64 `json:"units"`
	Rate         float64 `json:"rate"`          // limit price
	StopLoss     float64 `json:"stop_loss"`
	TakeProfit   float64 `json:"take_profit"`
	Status       string  `json:"status"`
}

// etoroOrder maps raw eToro API response fields.
type etoroOrder struct {
	OrderID      json.Number `json:"OrderID"`
	InstrumentID int         `json:"InstrumentID"`
	IsBuy        bool        `json:"IsBuy"`
	Amount       float64     `json:"Amount"`
	Units        float64     `json:"Units"`
	Rate         float64     `json:"Rate"`
	StopLossRate float64     `json:"StopLossRate"`
	TakeProfitRate float64   `json:"TakeProfitRate"`
	Status       string      `json:"Status"`
}

// GetOrders returns all open and pending orders for the account.
func GetOrders(c Caller) ([]Order, error) {
	prefix := "/api/v1/trading/demo"
	if !c.IsPaper() {
		prefix = "/api/v1/trading/real"
	}

	data, err := c.Get(prefix+"/orders", nil)
	if err != nil {
		return nil, fmt.Errorf("get_orders: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Order{}, nil
	}

	var raw []etoroOrder
	if err := json.Unmarshal(data, &raw); err != nil {
		var wrapper struct {
			Orders []etoroOrder `json:"orders"`
		}
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return nil, fmt.Errorf("get_orders: parse: %w", err)
		}
		raw = wrapper.Orders
	}

	out := make([]Order, 0, len(raw))
	for _, r := range raw {
		ord := Order{
			OrderID:      r.OrderID.String(),
			InstrumentID: r.InstrumentID,
			IsBuy:        r.IsBuy,
			Amount:       r.Amount,
			Units:        r.Units,
			Rate:         r.Rate,
			StopLoss:     r.StopLossRate,
			TakeProfit:   r.TakeProfitRate,
			Status:       r.Status,
		}
		name, err := resolveSymbolName(c, r.InstrumentID)
		if err == nil {
			ord.Symbol = name
		}
		out = append(out, ord)
	}

	return out, nil
}
