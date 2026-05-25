// Package queue manages the persistent order execution queue.
// Orders are stored in queue.json and executed by the daemon when
// the target market window opens.
package queue

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Order types
const (
	TypeBuy    = "buy"
	TypeSell   = "sell"
	TypeStop   = "stop"   // place a new stop order
	TypeTarget = "target" // place a new target (take-profit) order
	TypeModify = "modify" // modify an existing open order
	TypeCancel = "cancel" // cancel an existing open order
	TypeExec   = "exec"   // run an arbitrary shell command
)

// Execution windows
const (
	WindowNextOpen  = "next_open"  // US regular session open (9:30 AM ET / 21:30 SGT)
	WindowPreMarket = "pre_market" // US pre-market (4:00 AM ET / 16:00 SGT)
	WindowPreOpen   = "pre_open"   // 10 min before US open (9:20 AM ET / 21:20 SGT)
	WindowMorning   = "morning"    // SGT morning scan (7:00 AM SGT / 23:00 UTC prev day)
	WindowNow       = "now"        // execute immediately on next daemon tick
)

// Statuses
const (
	StatusPending   = "pending"
	StatusSubmitted = "submitted"
	StatusFilled    = "filled"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Order is a pending execution instruction.
type Order struct {
	ID         string     `json:"id"`
	Type       string     `json:"type"`
	Symbol     string     `json:"symbol,omitempty"`
	Qty        int        `json:"qty,omitempty"`
	Limit      float64    `json:"limit,omitempty"`
	Stop       float64    `json:"stop,omitempty"`
	Target     float64    `json:"target,omitempty"`
	OrderID    string     `json:"order_id,omitempty"`  // for modify/cancel
	Cmd        string     `json:"cmd,omitempty"`       // for exec type
	Daily      bool       `json:"daily,omitempty"`     // re-queue daily after execution
	Window     string     `json:"window"`
	ExecuteAt  time.Time  `json:"execute_at"`
	Note       string     `json:"note,omitempty"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	ExecutedAt *time.Time `json:"executed_at,omitempty"`
	Result     string     `json:"result,omitempty"`
	Error      string     `json:"error,omitempty"`
}

// Summary returns a one-line human-readable description.
func (o Order) Summary() string {
	switch o.Type {
	case TypeBuy:
		if o.Limit > 0 {
			return fmt.Sprintf("BUY %s %d @ limit $%.2f", o.Symbol, o.Qty, o.Limit)
		}
		return fmt.Sprintf("BUY %s %d (market)", o.Symbol, o.Qty)
	case TypeSell:
		if o.Limit > 0 {
			return fmt.Sprintf("SELL %s %d @ limit $%.2f", o.Symbol, o.Qty, o.Limit)
		}
		return fmt.Sprintf("SELL %s %d (market)", o.Symbol, o.Qty)
	case TypeStop:
		return fmt.Sprintf("STOP %s %d @ $%.2f", o.Symbol, o.Qty, o.Stop)
	case TypeTarget:
		return fmt.Sprintf("TARGET %s %d @ $%.2f", o.Symbol, o.Qty, o.Target)
	case TypeModify:
		parts := ""
		if o.Stop > 0 {
			parts += fmt.Sprintf(" stop→$%.2f", o.Stop)
		}
		if o.Limit > 0 {
			parts += fmt.Sprintf(" limit→$%.2f", o.Limit)
		}
		if o.Qty > 0 {
			parts += fmt.Sprintf(" qty→%d", o.Qty)
		}
		return fmt.Sprintf("MODIFY %s%s", o.OrderID[:12], parts)
	case TypeCancel:
		return fmt.Sprintf("CANCEL order %s", o.OrderID[:12])
	case TypeExec:
		cmd := o.Cmd
		if len(cmd) > 40 {
			cmd = cmd[:37] + "..."
		}
		daily := ""
		if o.Daily {
			daily = " [daily]"
		}
		return fmt.Sprintf("EXEC %s%s", cmd, daily)
	}
	return o.Type
}

// Queue manages the persistent order list.
type Queue struct {
	mu     sync.Mutex
	path   string
	orders []Order
}

func Open(path string) (*Queue, error) {
	q := &Queue{path: path}
	return q, q.load()
}

func (q *Queue) Add(o Order) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	o.ID = newID()
	o.Status = StatusPending
	o.CreatedAt = time.Now().UTC()
	o.ExecuteAt = computeExecuteAt(o.Window)
	q.orders = append(q.orders, o)
	return q.flush()
}

func (q *Queue) Pending() []Order {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []Order
	for _, o := range q.orders {
		if o.Status == StatusPending {
			out = append(out, o)
		}
	}
	return out
}

func (q *Queue) All() []Order {
	q.mu.Lock()
	defer q.mu.Unlock()
	out := make([]Order, len(q.orders))
	copy(out, q.orders)
	return out
}

func (q *Queue) Cancel(id string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i := range q.orders {
		if q.orders[i].ID == id && q.orders[i].Status == StatusPending {
			q.orders[i].Status = StatusCancelled
			return q.flush()
		}
	}
	return fmt.Errorf("no pending order with id %s", id)
}

func (q *Queue) SetResult(id, status, result, errMsg string) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now().UTC()
	for i := range q.orders {
		if q.orders[i].ID == id {
			q.orders[i].Status = status
			q.orders[i].ExecutedAt = &now
			q.orders[i].Result = result
			q.orders[i].Error = errMsg
			return q.flush()
		}
	}
	return fmt.Errorf("order %s not found", id)
}

func (q *Queue) load() error {
	data, err := os.ReadFile(q.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &q.orders)
}

func (q *Queue) flush() error {
	data, err := json.MarshalIndent(q.orders, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(q.path, data, 0644)
}

// computeExecuteAt returns the UTC time when an order in the given window should fire.
// Uses the America/New_York timezone to handle EST/EDT transitions automatically.
func computeExecuteAt(window string) time.Time {
	now := time.Now().UTC()
	loc := etLocation()

	switch window {
	case WindowPreMarket:
		return nextSessionTime(now, loc, 4, 0)
	case WindowPreOpen:
		return nextSessionTime(now, loc, 9, 20)
	case WindowMorning:
		// 7:00 AM SGT = 23:00 UTC previous day
		sgLoc, err := time.LoadLocation("Asia/Singapore")
		if err != nil {
			// Fallback: SGT is always UTC+8 (no DST)
			sgLoc = time.FixedZone("SGT", 8*60*60)
		}
		sgtNow := now.In(sgLoc)
		target := time.Date(sgtNow.Year(), sgtNow.Month(), sgtNow.Day(), 7, 0, 0, 0, sgLoc)
		if !sgtNow.Before(target) {
			target = target.Add(24 * time.Hour)
		}
		for w := target.Weekday(); w == time.Saturday || w == time.Sunday; w = target.Weekday() {
			target = target.Add(24 * time.Hour)
		}
		return target.UTC()
	case WindowNow:
		return now
	default: // WindowNextOpen
		return nextSessionTime(now, loc, 9, 30)
	}
}

// etLocation returns the America/New_York location for proper EST/EDT handling.
// Falls back to fixed UTC-5 (EST) if the timezone database is unavailable.
func etLocation() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.FixedZone("EST", -5*60*60)
	}
	return loc
}

// nextSessionTime returns the next occurrence of hour:min in the given location, skipping weekends.
func nextSessionTime(now time.Time, loc *time.Location, hour, min int) time.Time {
	local := now.In(loc)
	target := time.Date(local.Year(), local.Month(), local.Day(), hour, min, 0, 0, loc)
	if !local.Before(target) {
		target = target.Add(24 * time.Hour)
	}
	for w := target.Weekday(); w == time.Saturday || w == time.Sunday; w = target.Weekday() {
		target = target.Add(24 * time.Hour)
	}
	return target.UTC()
}

func newID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
