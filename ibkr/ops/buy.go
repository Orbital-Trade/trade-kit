package ops

import (
	"encoding/json"
	"fmt"
)

// BuyMarket places a market buy order.
func BuyMarket(c Caller, symbol string, qty int) (OrderResult, error) {
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "BUY", Type: "MKT", Qty: qty}, nil
	}

	conid, err := c.ResolveConID(symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_market %s: %w", symbol, err)
	}

	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/iserver/account/%s/orders", accountID)
	body := map[string]interface{}{
		"orders": []map[string]interface{}{
			{
				"conid":     conid,
				"orderType": "MKT",
				"side":      "BUY",
				"quantity":  qty,
				"tif":       "DAY",
			},
		},
	}

	data, err := c.Post(path, body)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_market %s: %w", symbol, err)
	}
	return parseOrderResult(data, symbol, "BUY", "MKT", qty, 0)
}

// BuyLimit places a limit buy order.
func BuyLimit(c Caller, symbol string, qty int, limitPrice float64) (OrderResult, error) {
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "BUY", Type: "LMT", Qty: qty, Price: limitPrice}, nil
	}

	conid, err := c.ResolveConID(symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_limit %s: %w", symbol, err)
	}

	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/iserver/account/%s/orders", accountID)
	body := map[string]interface{}{
		"orders": []map[string]interface{}{
			{
				"conid":     conid,
				"orderType": "LMT",
				"side":      "BUY",
				"quantity":  qty,
				"price":     limitPrice,
				"tif":       "DAY",
			},
		},
	}

	data, err := c.Post(path, body)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_limit %s: %w", symbol, err)
	}
	return parseOrderResult(data, symbol, "BUY", "LMT", qty, limitPrice)
}

func parseOrderResult(data json.RawMessage, symbol, action, orderType string, qty int, price float64) (OrderResult, error) {
	// IBKR returns an array of order reply objects.
	var replies []struct {
		OrderID   string `json:"order_id"`
		OrderStat string `json:"order_status"`
	}
	if err := json.Unmarshal(data, &replies); err == nil && len(replies) > 0 && replies[0].OrderID != "" {
		return OrderResult{
			OrderID: replies[0].OrderID,
			Mode:    "LIVE",
			Symbol:  symbol,
			Action:  action,
			Type:    orderType,
			Qty:     qty,
			Price:   price,
		}, nil
	}

	// Fallback: try single object.
	var single struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(data, &single); err == nil && single.OrderID != "" {
		return OrderResult{
			OrderID: single.OrderID,
			Mode:    "LIVE",
			Symbol:  symbol,
			Action:  action,
			Type:    orderType,
			Qty:     qty,
			Price:   price,
		}, nil
	}

	return OrderResult{}, fmt.Errorf("place_order %s: no order ID returned", symbol)
}
