package ops

// ModifyOrder — MCP tool: tiger_modify
//
// Modifies an existing open order in-place: price, quantity, or TIF.
// Calls Tiger REST method: modify_order.
//
// Tiger modify_order requires the full order context (action, order_type,
// symbol, currency, sec_type) plus the fields being changed. This function
// fetches the live order first so the caller only needs to specify what changes.
//
// Modifiable fields: limit_price, aux_price (stop trigger), total_quantity, time_in_force.

import (
	"fmt"
	"strconv"
)

// ModifyParams holds the fields the caller wants to change.
// Zero values mean "keep original".
type ModifyParams struct {
	LimitPrice  float64 // new limit price (0 = no change)
	StopPrice   float64 // new stop/aux price (0 = no change)
	Quantity    int     // new quantity (0 = no change)
	TimeInForce string  // new TIF: "DAY" or "GTC" ("" = no change)
}

// ModifyResult describes the outcome of a modify request.
type ModifyResult struct {
	OrderID string `json:"order_id"`
	Mode    string `json:"mode"`
	Status  string `json:"status"` // "MODIFIED" or "PAPER_MODIFIED"
}

// ModifyOrder modifies the open order with the given ID.
// It fetches the current order state, applies only the non-zero fields
// from params, then calls modify_order.
// In paper mode the call is skipped and PAPER_MODIFIED is returned.
func ModifyOrder(c Caller, orderID string, params ModifyParams) (ModifyResult, error) {
	if c.IsPaper() {
		return ModifyResult{OrderID: orderID, Mode: "PAPER", Status: "PAPER_MODIFIED"}, nil
	}

	// Fetch current order to get the required context fields.
	orders, err := GetOrders(c)
	if err != nil {
		return ModifyResult{}, fmt.Errorf("modify_order %s: fetch orders: %w", orderID, err)
	}

	var cur *Order
	for i := range orders {
		if orders[i].ID == orderID {
			cur = &orders[i]
			break
		}
	}
	if cur == nil {
		return ModifyResult{}, fmt.Errorf("modify_order %s: order not found (may already be filled or cancelled)", orderID)
	}

	// Tiger modify_order requires the numeric id as an int64.
	idInt, err := strconv.ParseInt(orderID, 10, 64)
	if err != nil {
		return ModifyResult{}, fmt.Errorf("modify_order: invalid order ID %q: %w", orderID, err)
	}

	// Build the biz_content. Start from current order values;
	// override only the fields the caller specified.
	limitPrice := cur.LimitPrice
	if params.LimitPrice > 0 {
		limitPrice = params.LimitPrice
	}
	auxPrice := cur.StopPrice
	if params.StopPrice > 0 {
		auxPrice = params.StopPrice
	}
	qty := cur.Quantity
	if params.Quantity > 0 {
		qty = params.Quantity
	}
	tif := cur.TimeInForce
	if params.TimeInForce != "" {
		tif = params.TimeInForce
	}

	biz := map[string]interface{}{
		"account":        c.Account(),
		"id":             idInt,
		"symbol":         cur.Symbol,
		"currency":       "USD",
		"sec_type":       "STK",
		"action":         cur.Action,
		"order_type":     cur.OrderType,
		"total_quantity": qty,
		"time_in_force":  tif,
		"lang":           "en_US",
	}
	if limitPrice > 0 {
		biz["limit_price"] = limitPrice
	}
	if auxPrice > 0 {
		biz["aux_price"] = auxPrice
	}

	_, err = c.Call("modify_order", biz)
	if err != nil {
		return ModifyResult{}, fmt.Errorf("modify_order %s: %w", orderID, err)
	}
	return ModifyResult{OrderID: orderID, Mode: "LIVE", Status: "MODIFIED"}, nil
}
