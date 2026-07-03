// Package recipe manages strategy recipe execution as goroutines.
package recipe

import "time"

// RecipeSignal is a unified signal emitted by any strategy.
type RecipeSignal struct {
	ID          string    `json:"id"`
	RecipeID    string    `json:"recipe_id"`
	Symbol      string    `json:"symbol"`
	Action      string    `json:"action"` // "enter", "enter_short", "exit", "skip", "watch"
	Reason      string    `json:"reason"`
	Qty         int       `json:"qty"`
	LimitPrice  float64   `json:"limit_price,omitempty"`
	StopPrice   float64   `json:"stop_price,omitempty"`
	TargetPrice float64   `json:"target_price,omitempty"`
	Status      string    `json:"status"` // "pending", "filled", "expired", "rejected"
	EntryID     string    `json:"entry_id,omitempty"`
	StopID      string    `json:"stop_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// RecipeState is the status of a recipe, returned by the list endpoint.
type RecipeState struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	Status     string     `json:"status"` // "stopped", "running", "error"
	StartedAt  *time.Time `json:"started_at,omitempty"`
	Error      string     `json:"error,omitempty"`
	ScanCount  int        `json:"scan_count"`
	LastScanAt *time.Time `json:"last_scan_at,omitempty"`
	SignalCount int       `json:"signal_count"`
}

// BrokerExecutor is the interface the runner needs from the broker registry.
type BrokerExecutor interface {
	Buy(symbol string, qty int, limitPrice, stopPrice float64) (entryID, stopID string, err error)
	Sell(symbol string, qty int) (orderID string, err error)
	IsPaper() bool
}

// EventBroadcaster pushes SSE events to connected clients.
type EventBroadcaster interface {
	Broadcast(eventType string, data any)
}
