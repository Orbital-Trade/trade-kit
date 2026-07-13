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

	data, err := c.Post("/v2/orders", map[string]interface{}{
		"symbol":        symbol,
		"qty":           qty,
		"side":          "buy",
		"type":          "market",
		"time_in_force": "day",
	})
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

	data, err := c.Post("/v2/orders", map[string]interface{}{
		"symbol":        symbol,
		"qty":           qty,
		"side":          "buy",
		"type":          "limit",
		"limit_price":   limitPrice,
		"time_in_force": "day",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_limit %s: %w", symbol, err)
	}
	return parseOrderResult(data, symbol, "BUY", "LMT", qty, limitPrice)
}

// BuyWithStops places a bracket order: entry + stop-loss + take-profit.
func BuyWithStops(c Caller, symbol string, qty int, limitPrice, stopPrice, takeProfit float64) (OrderResult, error) {
	if c.IsPaper() {
		tp := "MKT"
		if limitPrice > 0 {
			tp = "LMT"
		}
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: "BUY", Type: tp, Qty: qty, Price: limitPrice}, nil
	}

	order := map[string]interface{}{
		"symbol":        symbol,
		"qty":           qty,
		"side":          "buy",
		"time_in_force": "day",
	}

	if limitPrice > 0 {
		order["type"] = "limit"
		order["limit_price"] = limitPrice
	} else {
		order["type"] = "market"
	}

	// Bracket order for stop-loss and take-profit.
	if stopPrice > 0 && takeProfit > 0 {
		order["order_class"] = "bracket"
		order["stop_loss"] = map[string]interface{}{"stop_price": stopPrice}
		order["take_profit"] = map[string]interface{}{"limit_price": takeProfit}
	} else if stopPrice > 0 {
		order["order_class"] = "oto"
		order["stop_loss"] = map[string]interface{}{"stop_price": stopPrice}
	}

	data, err := c.Post("/v2/orders", order)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy %s: %w", symbol, err)
	}

	tp := "MKT"
	if limitPrice > 0 {
		tp = "LMT"
	}
	return parseOrderResult(data, symbol, "BUY", tp, qty, limitPrice)
}

func parseOrderResult(data json.RawMessage, symbol, action, orderType string, qty int, price float64) (OrderResult, error) {
	var res struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return OrderResult{}, fmt.Errorf("place_order %s: parse: %w", symbol, err)
	}
	if res.ID == "" {
		return OrderResult{}, fmt.Errorf("place_order %s: no order ID returned", symbol)
	}
	return OrderResult{
		OrderID: res.ID,
		Mode:    "LIVE",
		Symbol:  symbol,
		Action:  action,
		Type:    orderType,
		Qty:     qty,
		Price:   price,
	}, nil
}
