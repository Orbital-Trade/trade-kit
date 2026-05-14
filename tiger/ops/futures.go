package ops

// Futures operations — MCP tools: tiger_futures_entry, tiger_futures_close,
// tiger_futures_update_stop.
//
// All futures stops use order_type=STP (stop-market). See stop.go for the
// rationale. A stop-limit on a futures position is never acceptable.
//
// FuturesEntry always places a matching protective stop immediately after the
// entry order. If the stop placement fails for any reason, an emergency market
// close is attempted automatically before returning an error.

import (
	"fmt"
	"os"
)

// FuturesEntryResult holds both order IDs from a paired entry + stop placement.
type FuturesEntryResult struct {
	EntryOrderID string  `json:"entry_order_id"`
	StopOrderID  string  `json:"stop_order_id"`
	Mode         string  `json:"mode"`
	Symbol       string  `json:"symbol"`
	Direction    string  `json:"direction"` // "LONG" or "SHORT"
	Contracts    int     `json:"contracts"`
	EntryPrice   float64 `json:"entry_price"`
	StopPrice    float64 `json:"stop_price"`
}

// FuturesEntry places a DAY limit entry order followed immediately by a GTC
// stop-market protective stop. direction must be "LONG" or "SHORT".
//
// Emergency close: if the stop placement fails after entry is submitted, a
// market close is sent automatically to prevent an unprotected position.
func FuturesEntry(c Caller, symbol, direction string, contracts int, entryPrice, stopPrice float64) (FuturesEntryResult, error) {
	action, stopAction := actionsFor(direction)

	if c.IsPaper() {
		return FuturesEntryResult{
			EntryOrderID: "PAPER", StopOrderID: "PAPER", Mode: "PAPER",
			Symbol: symbol, Direction: direction, Contracts: contracts,
			EntryPrice: entryPrice, StopPrice: stopPrice,
		}, nil
	}

	contractCode, err := c.ResolveFuturesContract(symbol)
	if err != nil {
		return FuturesEntryResult{}, fmt.Errorf("futures_entry: resolve %s: %w", symbol, err)
	}

	// Place entry (DAY limit).
	entryData, err := c.Call("place_order", map[string]interface{}{
		"account":        c.Account(),
		"symbol":         contractCode,
		"sec_type":       "FUT",
		"currency":       "USD",
		"order_type":     "LMT",
		"action":         action,
		"total_quantity": contracts,
		"limit_price":    entryPrice,
		"time_in_force":  "DAY",
		"outside_rth":    false,
		"lang":           "en_US",
	})
	if err != nil {
		return FuturesEntryResult{}, fmt.Errorf("futures_entry %s: entry order: %w", symbol, err)
	}
	entryRes, err := parseOrderID(entryData, symbol, action, "LMT", contracts, entryPrice)
	if err != nil {
		return FuturesEntryResult{}, err
	}

	// Place protective stop (GTC stop-market).
	stopData, err := c.Call("place_order", map[string]interface{}{
		"account":        c.Account(),
		"symbol":         contractCode,
		"sec_type":       "FUT",
		"currency":       "USD",
		"order_type":     "STP",
		"action":         stopAction,
		"total_quantity": contracts,
		"aux_price":      stopPrice,
		"time_in_force":  "GTC",
		"outside_rth":    false,
		"lang":           "en_US",
	})
	if err != nil {
		// Emergency close: entry was submitted but stop failed — position is
		// unprotected. Attempt a market close before returning the error.
		fmt.Fprintf(os.Stderr, "[CRITICAL] stop order failed after entry %s — sending emergency market close\n", entryRes.OrderID)
		_, closeErr := FuturesClose(c, symbol, direction, contracts)
		if closeErr != nil {
			return FuturesEntryResult{}, fmt.Errorf(
				"CRITICAL: stop AND emergency close both failed — MANUAL INTERVENTION REQUIRED. entry=%s stopErr=%v closeErr=%v",
				entryRes.OrderID, err, closeErr,
			)
		}
		return FuturesEntryResult{}, fmt.Errorf("futures_entry %s: stop failed (position emergency-closed): %w", symbol, err)
	}

	stopRes, err := parseOrderID(stopData, symbol, stopAction, "STP", contracts, stopPrice)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[CRITICAL] stop response parse failed after entry %s — sending emergency market close\n", entryRes.OrderID)
		FuturesClose(c, symbol, direction, contracts) //nolint:errcheck
		return FuturesEntryResult{}, fmt.Errorf("futures_entry %s: stop parse failed (position emergency-closed): %w", symbol, err)
	}

	return FuturesEntryResult{
		EntryOrderID: entryRes.OrderID,
		StopOrderID:  stopRes.OrderID,
		Mode:         "LIVE",
		Symbol:       symbol,
		Direction:    direction,
		Contracts:    contracts,
		EntryPrice:   entryPrice,
		StopPrice:    stopPrice,
	}, nil
}

// FuturesClose closes an open futures position with a DAY market order.
// direction is the direction of the existing position ("LONG" or "SHORT");
// the close order uses the opposite side.
func FuturesClose(c Caller, symbol, direction string, contracts int) (OrderResult, error) {
	_, closeAction := actionsFor(direction)

	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: closeAction, Type: "MKT", Qty: contracts}, nil
	}

	contractCode, err := c.ResolveFuturesContract(symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("futures_close: resolve %s: %w", symbol, err)
	}

	data, err := c.Call("place_order", map[string]interface{}{
		"account":        c.Account(),
		"symbol":         contractCode,
		"sec_type":       "FUT",
		"currency":       "USD",
		"order_type":     "MKT",
		"action":         closeAction,
		"total_quantity": contracts,
		"time_in_force":  "DAY",
		"lang":           "en_US",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("futures_close %s: %w", symbol, err)
	}
	return parseOrderID(data, symbol, closeAction, "MKT", contracts, 0)
}

// FuturesUpdateStop cancels the previous stop order and places a new
// stop-market at newStopPrice. Used for trailing stop management.
// If the cancel fails it is logged and execution continues — the new stop
// is more important than confirming the old one is gone.
func FuturesUpdateStop(c Caller, symbol, direction string, contracts int, newStopPrice float64, oldStopID string) (OrderResult, error) {
	_, stopAction := actionsFor(direction)

	if c.IsPaper() {
		return OrderResult{OrderID: "PAPER", Mode: "PAPER", Symbol: symbol, Action: stopAction, Type: "STP", Qty: contracts, Price: newStopPrice}, nil
	}

	// Cancel old stop (best-effort).
	if oldStopID != "" {
		_, err := c.Call("cancel_order", map[string]interface{}{
			"account": c.Account(),
			"id":      oldStopID,
			"lang":    "en_US",
		})
		if err != nil {
			// Non-fatal: the old stop may have already filled or been cancelled.
			fmt.Fprintf(os.Stderr, "[WARN] futures_update_stop: cancel %s: %v\n", oldStopID, err)
		}
	}

	contractCode, err := c.ResolveFuturesContract(symbol)
	if err != nil {
		return OrderResult{}, fmt.Errorf("futures_update_stop: resolve %s: %w", symbol, err)
	}

	data, err := c.Call("place_order", map[string]interface{}{
		"account":        c.Account(),
		"symbol":         contractCode,
		"sec_type":       "FUT",
		"currency":       "USD",
		"order_type":     "STP",
		"action":         stopAction,
		"total_quantity": contracts,
		"aux_price":      newStopPrice,
		"time_in_force":  "GTC",
		"lang":           "en_US",
	})
	if err != nil {
		return OrderResult{}, fmt.Errorf("futures_update_stop %s: %w", symbol, err)
	}
	return parseOrderID(data, symbol, stopAction, "STP", contracts, newStopPrice)
}

// actionsFor returns the entry and protective-stop actions for a direction.
// LONG entry: BUY / stop: SELL. SHORT entry: SELL / stop: BUY.
func actionsFor(direction string) (entry, stop string) {
	if direction == "SHORT" {
		return "SELL", "BUY"
	}
	return "BUY", "SELL"
}
