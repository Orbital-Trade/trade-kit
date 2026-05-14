package broker

import (
	"fmt"

	"tiger-cli/client"
	"tiger-cli/ops"
)

// LiveBroker executes orders against the Tiger Brokers API.
type LiveBroker struct {
	caller ops.Caller
}

func NewLive() (*LiveBroker, error) {
	c, err := client.New(false) // false = live, not paper
	if err != nil {
		return nil, fmt.Errorf("tiger client: %w", err)
	}
	return &LiveBroker{caller: c}, nil
}

// CheckGate validates that adding a new position of newCost USD doesn't breach
// the playbook capital rules. Returns nil if OK, descriptive error if blocked.
//
//   Rule 1: max 10 concurrent positions
//   Rule 2: new position ≤ 30% of net liquidation
//   Rule 3: cash reserve ≥ 15% of net liquidation after trade
//
// Fails open (returns nil) if Tiger API is unreachable — better a missed check
// than a missed trade on a connectivity hiccup.
func (l *LiveBroker) CheckGate(newCost float64) error {
	positions, err := ops.GetPositions(l.caller)
	if err != nil {
		return nil // fail open
	}
	account, err := ops.GetAccount(l.caller)
	if err != nil {
		return nil
	}
	if account.NetLiquidation <= 0 {
		return nil
	}

	// Rule 1: max 10 concurrent positions
	if len(positions) >= 10 {
		return fmt.Errorf("max positions reached (%d/10) — close one before opening new", len(positions))
	}

	// Rule 2: position size ≤ 30% of net liquidation
	posRatio := newCost / account.NetLiquidation
	if posRatio > 0.30 {
		return fmt.Errorf("position size %.0f%% of NAV exceeds 30%% limit ($%.0f of $%.0f net liq)",
			posRatio*100, newCost, account.NetLiquidation)
	}

	// Rule 3: maintain ≥15% cash reserve after trade
	cashAfter := account.Cash - newCost
	cashRatio := cashAfter / account.NetLiquidation
	if cashRatio < 0.15 {
		shortfall := 0.15*account.NetLiquidation - cashAfter
		return fmt.Errorf("cash reserve would drop to %.0f%% (min 15%%) — need $%.0f more cash",
			cashRatio*100, shortfall)
	}

	return nil
}

func (l *LiveBroker) Buy(symbol string, qty int, limitPrice, stopPrice float64) (entryID, stopID string, err error) {
	entry, err := ops.BuyLimit(l.caller, symbol, qty, limitPrice)
	if err != nil {
		return "", "", fmt.Errorf("buy_limit %s: %w", symbol, err)
	}
	entryID = entry.OrderID

	if stopPrice > 0 {
		stop, err := ops.SetStopLoss(l.caller, symbol, qty, stopPrice)
		if err != nil {
			return entryID, "", fmt.Errorf("stop %s: %w", symbol, err)
		}
		stopID = stop.OrderID
	}
	return entryID, stopID, nil
}

func (l *LiveBroker) Sell(symbol string, qty int) (string, error) {
	result, err := ops.SellMarket(l.caller, symbol, qty)
	if err != nil {
		return "", fmt.Errorf("sell_market %s: %w", symbol, err)
	}
	return result.OrderID, nil
}

func (l *LiveBroker) Mode() string { return "live" }
