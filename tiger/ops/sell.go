package ops

// SellMarket / SellLimit — MCP tool: tiger_sell
//
// Places a stock sell order. SellLimit (GTC) is the default exit strategy:
// set a target and let the order work. SellMarket is for urgent exits where
// fill certainty matters more than price.
// Calls Tiger REST method: place_order (sec_type=STK).

import "fmt"

// SellMarket places a DAY market sell order. Fill is guaranteed; price is not.
func SellMarket(c Caller, symbol string, shares int) (OrderResult, error) {
	info := DetectMarket(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: info.Symbol, Action: "SELL", Type: "MKT", Qty: shares}, nil
	}
	p := info.OrderParams(c.Account())
	p["order_type"] = "MKT"
	p["action"] = "SELL"
	p["total_quantity"] = shares
	p["time_in_force"] = "DAY"
	data, err := c.Call("place_order", p)
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_market %s: %w", info.Symbol, err)
	}
	return parseOrderID(data, info.Symbol, "SELL", "MKT", shares, 0)
}

// SellLimit places a GTC limit sell order. Typically used for take-profit exits.
// The order remains active until filled, cancelled, or the position is closed.
func SellLimit(c Caller, symbol string, shares int, limitPrice float64) (OrderResult, error) {
	info := DetectMarket(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: info.Symbol, Action: "SELL", Type: "LMT", Qty: shares, Price: limitPrice}, nil
	}
	p := info.OrderParams(c.Account())
	p["order_type"] = "LMT"
	p["action"] = "SELL"
	p["total_quantity"] = shares
	p["limit_price"] = limitPrice
	p["time_in_force"] = "GTC"
	data, err := c.Call("place_order", p)
	if err != nil {
		return OrderResult{}, fmt.Errorf("sell_limit %s: %w", info.Symbol, err)
	}
	return parseOrderID(data, info.Symbol, "SELL", "LMT", shares, limitPrice)
}
