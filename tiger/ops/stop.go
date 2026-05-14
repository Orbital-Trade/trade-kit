package ops

// SetStopLoss — MCP tool: tiger_stop
//
// Places a GTC stop-market sell order (protective stop loss on a long position).
//
// Order type: STP (stop-market), NOT STP_LMT (stop-limit).
// Rationale: a stop-market order guarantees a fill when the stop price is
// triggered. A stop-limit can miss the fill in a fast-moving market and leave
// the position fully exposed — never acceptable for a protective stop.
// Calls Tiger REST method: place_order (order_type=STP, aux_price=stop).

import "fmt"

// SetStopLoss places a GTC stop-market sell to protect a long stock position.
// The stop triggers at stopPrice and exits at the next available market price.
func SetStopLoss(c Caller, symbol string, shares int, stopPrice float64) (OrderResult, error) {
	info := DetectMarket(symbol)
	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: info.Symbol, Action: "SELL", Type: "STP", Qty: shares, Price: stopPrice}, nil
	}
	p := info.OrderParams(c.Account())
	p["order_type"] = "STP" // stop-market
	p["action"] = "SELL"
	p["total_quantity"] = shares
	p["aux_price"] = stopPrice // Tiger uses aux_price as the stop trigger
	p["time_in_force"] = "GTC"
	data, err := c.Call("place_order", p)
	if err != nil {
		return OrderResult{}, fmt.Errorf("set_stop_loss %s: %w", info.Symbol, err)
	}
	return parseOrderID(data, info.Symbol, "SELL", "STP", shares, stopPrice)
}
