package ops

// BuyMarket / BuyLimit — MCP tool: tiger_buy
//
// Places a stock buy order. BuyLimit is preferred: it gives price control and
// avoids market-order slippage. BuyMarket guarantees a fill at any price.
// Calls Tiger REST method: place_order (sec_type=STK).

import (
	"encoding/json"
	"fmt"
)

// BuyMarket places a DAY market buy order. Fill is guaranteed; price is not.
func BuyMarket(c Caller, symbol string, shares int) (OrderResult, error) {
	info := DetectMarket(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: info.Symbol, Action: "BUY", Type: "MKT", Qty: shares}, nil
	}
	p := info.OrderParams(c.Account())
	p["order_type"] = "MKT"
	p["action"] = "BUY"
	p["total_quantity"] = shares
	p["time_in_force"] = "DAY"
	data, err := c.Call("place_order", p)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_market %s: %w", info.Symbol, err)
	}
	return parseOrderID(data, info.Symbol, "BUY", "MKT", shares, 0)
}

// BuyLimit places a DAY limit buy order. Order fills only at limitPrice or better.
func BuyLimit(c Caller, symbol string, shares int, limitPrice float64) (OrderResult, error) {
	info := DetectMarket(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: info.Symbol, Action: "BUY", Type: "LMT", Qty: shares, Price: limitPrice}, nil
	}
	p := info.OrderParams(c.Account())
	p["order_type"] = "LMT"
	p["action"] = "BUY"
	p["total_quantity"] = shares
	p["limit_price"] = limitPrice
	p["time_in_force"] = "DAY"
	data, err := c.Call("place_order", p)
	if err != nil {
		return OrderResult{}, fmt.Errorf("buy_limit %s: %w", info.Symbol, err)
	}
	return parseOrderID(data, info.Symbol, "BUY", "LMT", shares, limitPrice)
}

// parseOrderID extracts the order ID from a place_order response.
// Returns an error if the API returns ID=0, which indicates the order was not accepted.
func parseOrderID(data json.RawMessage, symbol, action, orderType string, qty int, price float64) (OrderResult, error) {
	var res struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(data, &res); err != nil {
		return OrderResult{}, fmt.Errorf("place_order %s: parse response: %w", symbol, err)
	}
	if res.ID == 0 {
		return OrderResult{}, fmt.Errorf("place_order %s: API returned order ID 0 — order likely rejected", symbol)
	}
	return OrderResult{
		OrderID: fmt.Sprintf("%d", res.ID),
		Mode:    "LIVE",
		Symbol:  symbol,
		Action:  action,
		Type:    orderType,
		Qty:     qty,
		Price:   price,
	}, nil
}
