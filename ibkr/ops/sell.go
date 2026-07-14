package ops

import "fmt"

// SellMarket places a market sell order.
func SellMarket(c Caller, symbol string, qty int) (OrderResult, error) {
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "SELL", Type: "MKT", Qty: qty}, nil
	}

	conid, err := c.ResolveConID(symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_market %s: %w", symbol, err)
	}

	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/iserver/account/%s/orders", accountID)
	body := map[string]interface{}{
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

	data, err := c.Post(path, body)
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_market %s: %w", symbol, err)
	}
	return parseOrderResult(data, symbol, "SELL", "MKT", qty, 0)
}

// SellLimit places a GTC limit sell order.
func SellLimit(c Caller, symbol string, qty int, limitPrice float64) (OrderResult, error) {
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "SELL", Type: "LMT", Qty: qty, Price: limitPrice}, nil
	}

	conid, err := c.ResolveConID(symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_limit %s: %w", symbol, err)
	}

	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/iserver/account/%s/orders", accountID)
	body := map[string]interface{}{
		"orders": []map[string]interface{}{
			{
				"conid":     conid,
				"orderType": "LMT",
				"side":      "SELL",
				"quantity":  qty,
				"price":     limitPrice,
				"tif":       "GTC",
			},
		},
	}

	data, err := c.Post(path, body)
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_limit %s: %w", symbol, err)
	}
	return parseOrderResult(data, symbol, "SELL", "LMT", qty, limitPrice)
}
