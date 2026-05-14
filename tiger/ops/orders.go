package ops

// GetOrders — MCP tool: tiger_orders
//
// Returns all open or pending stock orders.
// Calls Tiger REST method: orders.
//
// Tiger API quirks handled here:
//   - Response is double-encoded (client.Call unwraps the outer JSON string).
//   - After unwrapping, data may be a direct array or {"items":[...]} wrapper.
//   - Field names are camelCase: totalQuantity, filledQuantity, orderType, etc.
//   - aux_price is the stop trigger price (used for STP orders).

import (
	"encoding/json"
	"fmt"
)

// Order represents an open or pending stock order.
type Order struct {
	ID          string  `json:"id"`
	Symbol      string  `json:"symbol"`
	Action      string  `json:"action"`      // "BUY" or "SELL"
	OrderType   string  `json:"order_type"`  // "LMT", "MKT", "STP"
	Quantity    int     `json:"quantity"`
	LimitPrice  float64 `json:"limit_price"` // 0 for market/stop orders
	StopPrice   float64 `json:"stop_price"`  // aux_price — stop trigger price
	FilledQty   int     `json:"filled_qty"`
	Status      string  `json:"status"`
	TimeInForce string  `json:"time_in_force"` // "DAY" or "GTC"
}

// tigerOrder maps the raw Tiger API field names (camelCase) to Go fields.
type tigerOrder struct {
	ID          int64   `json:"id"`
	Symbol      string  `json:"symbol"`
	Action      string  `json:"action"`
	OrderType   string  `json:"orderType"`
	TotalQty    int     `json:"totalQuantity"`
	LimitPrice  float64 `json:"limitPrice"`
	AuxPrice    float64 `json:"auxPrice"`      // stop trigger price
	FilledQty   int     `json:"filledQuantity"`
	Status      string  `json:"status"`
	TimeInForce string  `json:"timeInForce"`
}

// GetOrders returns all open and pending stock orders for the account.
func GetOrders(c Caller) ([]Order, error) {
	data, err := c.Call("orders", map[string]interface{}{
		"account":  c.Account(),
		"sec_type": "STK",
		"lang":     "en_US",
	})
	if err != nil {
		return nil, fmt.Errorf("get_orders: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Order{}, nil
	}

	raw, err := parseOrders(data)
	if err != nil {
		return nil, fmt.Errorf("get_orders: parse response: %w", err)
	}

	out := make([]Order, 0, len(raw))
	for _, r := range raw {
		if r.ID == 0 {
			continue
		}
		out = append(out, Order{
			ID:          fmt.Sprintf("%d", r.ID),
			Symbol:      r.Symbol,
			Action:      r.Action,
			OrderType:   r.OrderType,
			Quantity:    r.TotalQty,
			LimitPrice:  r.LimitPrice,
			StopPrice:   r.AuxPrice,
			FilledQty:   r.FilledQty,
			Status:      r.Status,
			TimeInForce: r.TimeInForce,
		})
	}
	return out, nil
}

// parseOrders handles both response shapes Tiger may return:
//   - Direct array:            [{...}, ...]
//   - Wrapped in items object: {"items":[{...}], ...}
func parseOrders(data json.RawMessage) ([]tigerOrder, error) {
	// Try direct array first.
	var direct []tigerOrder
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}

	// Try items wrapper — Tiger wraps list responses in {"items":[...]}.
	var wrapper struct {
		Items []tigerOrder `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("neither array nor {items:[]} format: %w", err)
	}
	return wrapper.Items, nil
}
