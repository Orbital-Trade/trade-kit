package ops

// CancelOrder — MCP tool: tiger_cancel
//
// Cancels an open order by ID.
// Calls Tiger REST method: cancel_order.

import "fmt"

// CancelResult confirms the outcome of a cancellation request.
type CancelResult struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"` // "CANCELLED" or "PAPER_CANCELLED"
}

// CancelOrder cancels the open order with the given ID.
// In paper mode the call is skipped and PAPER_CANCELLED is returned.
func CancelOrder(c Caller, orderID string) (CancelResult, error) {
	if c.IsPaper() {
		return CancelResult{OrderID: orderID, Status: "PAPER_CANCELLED"}, nil
	}
	_, err := c.Call("cancel_order", map[string]interface{}{
		"account": c.Account(),
		"id":      orderID,
		"lang":    "en_US",
	})
	if err != nil {
		return CancelResult{}, fmt.Errorf("cancel_order %s: %w", orderID, err)
	}
	return CancelResult{OrderID: orderID, Status: "CANCELLED"}, nil
}
