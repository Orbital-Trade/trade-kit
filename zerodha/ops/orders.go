package ops

import (
	"encoding/json"
	"fmt"
)

// Order represents an order for the day.
type Order struct {
	OrderID         string  `json:"order_id"`
	Symbol          string  `json:"tradingsymbol"`
	Exchange        string  `json:"exchange"`
	TransactionType string  `json:"transaction_type"` // "BUY" or "SELL"
	OrderType       string  `json:"order_type"`       // "MARKET", "LIMIT", "SL", "SL-M"
	Quantity        int     `json:"quantity"`
	FilledQty       int     `json:"filled_quantity"`
	Price           float64 `json:"price"`
	TriggerPrice    float64 `json:"trigger_price"`
	Product         string  `json:"product"`   // "CNC", "MIS", "NRML"
	Validity        string  `json:"validity"`  // "DAY", "IOC"
	Status          string  `json:"status"`
	StatusMessage   string  `json:"status_message"`
	AveragePrice    float64 `json:"average_price"`
}

// GetOrders returns all orders for the day.
func GetOrders(c Caller) ([]Order, error) {
	data, err := c.Get("/orders", nil)
	if err != nil {
		return nil, fmt.Errorf("get_orders: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Order{}, nil
	}

	var resp struct {
		Status string  `json:"status"`
		Data   []Order `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("get_orders: parse: %w", err)
	}
	if resp.Data == nil {
		return []Order{}, nil
	}
	return resp.Data, nil
}

// CancelOrder cancels an order by ID.
func CancelOrder(c Caller, orderID string) error {
	_, err := c.Delete("/orders/regular/"+orderID, nil)
	if err != nil {
		return fmt.Errorf("cancel_order %s: %w", orderID, err)
	}
	return nil
}
