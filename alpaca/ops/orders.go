package ops

import (
	"encoding/json"
	"fmt"
)

// Order represents an open or pending order.
type Order struct {
	ID             string  `json:"id"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"` // "buy" or "sell"
	Type           string  `json:"type"` // "market", "limit", "stop", "stop_limit", "trailing_stop"
	Qty            string  `json:"qty"`
	FilledQty      string  `json:"filled_qty"`
	FilledAvgPrice string  `json:"filled_avg_price"`
	LimitPrice     string  `json:"limit_price"`
	StopPrice      string  `json:"stop_price"`
	TimeInForce    string  `json:"time_in_force"` // "day", "gtc", "ioc", "fok"
	Status         string  `json:"status"`
	CreatedAt      string  `json:"created_at"`
}

// GetOrders returns all open/pending orders.
func GetOrders(c Caller) ([]Order, error) {
	data, err := c.Get("/v2/orders", map[string]string{
		"status": "open",
	})
	if err != nil {
		return nil, fmt.Errorf("get_orders: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Order{}, nil
	}

	var orders []Order
	if err := json.Unmarshal(data, &orders); err != nil {
		return nil, fmt.Errorf("get_orders: parse: %w", err)
	}
	return orders, nil
}

// CancelOrder cancels an order by ID.
func CancelOrder(c Caller, orderID string) error {
	_, err := c.Delete("/v2/orders/"+orderID, nil)
	if err != nil {
		return fmt.Errorf("cancel_order %s: %w", orderID, err)
	}
	return nil
}
