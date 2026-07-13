package ops

import "fmt"

// SellMarket places a market sell order.
func SellMarket(c Caller, symbol string, qty int) (OrderResult, error) {
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "SELL", Type: "MKT", Qty: qty}, nil
	}

	data, err := c.Post("/v2/orders", map[string]interface{}{
		"symbol":        symbol,
		"qty":           qty,
		"side":          "sell",
		"type":          "market",
		"time_in_force": "day",
	})
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

	data, err := c.Post("/v2/orders", map[string]interface{}{
		"symbol":        symbol,
		"qty":           qty,
		"side":          "sell",
		"type":          "limit",
		"limit_price":   limitPrice,
		"time_in_force": "gtc",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_limit %s: %w", symbol, err)
	}
	return parseOrderResult(data, symbol, "SELL", "LMT", qty, limitPrice)
}
