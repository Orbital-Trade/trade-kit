package ops

import (
	"encoding/json"
	"fmt"
)

// Order represents an open or pending order.
type Order struct {
	OrderID   string  `json:"orderId"`
	ConID     int     `json:"conid"`
	Symbol    string  `json:"ticker"`
	Side      string  `json:"side"`      // "BUY" or "SELL"
	OrderType string  `json:"orderType"` // "MKT", "LMT", "STP", etc.
	Qty       float64 `json:"totalSize"`
	FilledQty float64 `json:"filledQuantity"`
	Price     float64 `json:"price"`
	Status    string  `json:"status"`
	TimeInForce string `json:"timeInForce"`
}

// GetOrders returns all open/pending orders.
func GetOrders(c Caller) ([]Order, error) {
	data, err := c.Get("/v1/api/iserver/account/orders", nil)
	if err != nil {
		return nil, fmt.Errorf("get_orders: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []Order{}, nil
	}

	// IBKR wraps orders in {"orders": [...]}
	var wrapper struct {
		Orders []Order `json:"orders"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Try direct array fallback.
		var orders []Order
		if err2 := json.Unmarshal(data, &orders); err2 != nil {
			return nil, fmt.Errorf("get_orders: parse: %w", err)
		}
		return orders, nil
	}
	return wrapper.Orders, nil
}

// CancelOrder cancels an order by ID.
func CancelOrder(c Caller, orderID string) error {
	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/iserver/account/%s/order/%s", accountID, orderID)
	_, err := c.Delete(path, nil)
	if err != nil {
		return fmt.Errorf("cancel_order %s: %w", orderID, err)
	}
	return nil
}
