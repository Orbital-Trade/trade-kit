package ops

// SetTakeProfit — MCP tool: tiger_take_profit
//
// Places a GTC limit sell order to lock in profit on a long position.
// Delegates to SellLimit — separated into its own file so it maps to its
// own MCP tool with an unambiguous name.
// Calls Tiger REST method: place_order (order_type=LMT, time_in_force=GTC).

// SetTakeProfit places a GTC limit sell to exit a long position at a profit target.
func SetTakeProfit(c Caller, symbol string, shares int, limitPrice float64) (OrderResult, error) {
	return SellLimit(c, symbol, shares, limitPrice)
}
