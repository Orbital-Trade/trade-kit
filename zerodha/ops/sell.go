package ops

import (
	"fmt"
	"strconv"
)

// SellMarket places a market sell order.
// Default exchange: NSE, default product: CNC (delivery).
func SellMarket(c Caller, symbol string, qty int) (OrderResult, error) {
	exchange, sym := parseExchange(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: sym, Action: "SELL", Type: "MKT", Qty: qty}, nil
	}

	data, err := c.PostForm("/orders/regular", map[string]string{
		"tradingsymbol":    sym,
		"exchange":         exchange,
		"transaction_type": "SELL",
		"order_type":       "MARKET",
		"quantity":         strconv.Itoa(qty),
		"product":          "CNC",
		"validity":         "DAY",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_market %s: %w", sym, err)
	}
	return parseOrderResponse(data, sym, "SELL", "MKT", qty, 0)
}

// SellLimit places a limit sell order.
// Default exchange: NSE, default product: CNC (delivery).
func SellLimit(c Caller, symbol string, qty int, limitPrice float64) (OrderResult, error) {
	exchange, sym := parseExchange(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: sym, Action: "SELL", Type: "LMT", Qty: qty, Price: limitPrice}, nil
	}

	data, err := c.PostForm("/orders/regular", map[string]string{
		"tradingsymbol":    sym,
		"exchange":         exchange,
		"transaction_type": "SELL",
		"order_type":       "LIMIT",
		"quantity":         strconv.Itoa(qty),
		"product":          "CNC",
		"price":            fmt.Sprintf("%.2f", limitPrice),
		"validity":         "DAY",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_limit %s: %w", sym, err)
	}
	return parseOrderResponse(data, sym, "SELL", "LMT", qty, limitPrice)
}
