package ops

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// BuyMarket places a market buy order.
// Default exchange: NSE, default product: CNC (delivery).
// Use "BSE:SYMBOL" prefix to route to BSE.
func BuyMarket(c Caller, symbol string, qty int) (OrderResult, error) {
	exchange, sym := parseExchange(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: sym, Action: "BUY", Type: "MKT", Qty: qty}, nil
	}

	data, err := c.PostForm("/orders/regular", map[string]string{
		"tradingsymbol":    sym,
		"exchange":         exchange,
		"transaction_type": "BUY",
		"order_type":       "MARKET",
		"quantity":         strconv.Itoa(qty),
		"product":          "CNC",
		"validity":         "DAY",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_market %s: %w", sym, err)
	}
	return parseOrderResponse(data, sym, "BUY", "MKT", qty, 0)
}

// BuyLimit places a limit buy order.
// Default exchange: NSE, default product: CNC (delivery).
func BuyLimit(c Caller, symbol string, qty int, limitPrice float64) (OrderResult, error) {
	exchange, sym := parseExchange(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: sym, Action: "BUY", Type: "LMT", Qty: qty, Price: limitPrice}, nil
	}

	data, err := c.PostForm("/orders/regular", map[string]string{
		"tradingsymbol":    sym,
		"exchange":         exchange,
		"transaction_type": "BUY",
		"order_type":       "LIMIT",
		"quantity":         strconv.Itoa(qty),
		"product":          "CNC",
		"price":            fmt.Sprintf("%.2f", limitPrice),
		"validity":         "DAY",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_limit %s: %w", sym, err)
	}
	return parseOrderResponse(data, sym, "BUY", "LMT", qty, limitPrice)
}

// BuyIntraday places an intraday (MIS) market buy order.
func BuyIntraday(c Caller, symbol string, qty int) (OrderResult, error) {
	exchange, sym := parseExchange(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: sym, Action: "BUY", Type: "MKT", Qty: qty}, nil
	}

	data, err := c.PostForm("/orders/regular", map[string]string{
		"tradingsymbol":    sym,
		"exchange":         exchange,
		"transaction_type": "BUY",
		"order_type":       "MARKET",
		"quantity":         strconv.Itoa(qty),
		"product":          "MIS",
		"validity":         "DAY",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_intraday %s: %w", sym, err)
	}
	return parseOrderResponse(data, sym, "BUY", "MKT", qty, 0)
}

// parseExchange splits "BSE:RELIANCE" into ("BSE", "RELIANCE").
// Defaults to "NSE" if no prefix.
func parseExchange(symbol string) (string, string) {
	if idx := strings.Index(symbol, ":"); idx > 0 {
		return strings.ToUpper(symbol[:idx]), strings.ToUpper(symbol[idx+1:])
	}
	return "NSE", strings.ToUpper(symbol)
}

// parseOrderResponse extracts the order_id from Kite's response envelope.
// Kite returns: {"status": "success", "data": {"order_id": "..."}}
func parseOrderResponse(data json.RawMessage, symbol, action, orderType string, qty int, price float64) (OrderResult, error) {
	var resp struct {
		Status string `json:"status"`
		Data   struct {
			OrderID string `json:"order_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return OrderResult{}, fmt.Errorf("place_order %s: parse: %w", symbol, err)
	}
	if resp.Data.OrderID == "" {
		return OrderResult{}, fmt.Errorf("place_order %s: no order ID returned", symbol)
	}
	return OrderResult{
		OrderID: resp.Data.OrderID,
		Mode:    "LIVE",
		Symbol:  symbol,
		Action:  action,
		Type:    orderType,
		Qty:     qty,
		Price:   price,
	}, nil
}
