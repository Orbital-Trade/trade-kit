// Package signal defines the shared signal bus used by all strategy bots.
//
// signals.json is the universal queue. Any source can write to it:
//   - earnings-bot scan     (pre-earnings setups)
//   - bounce-bot scan       (RSI oversold setups)
//   - trade-kit scanner  (API-generated setups)
//   - Human manual entry    (edit the file directly)
//
// The bot's run loop monitors this file and executes pending signals
// that are still fresh (price within tolerance) and not expired.
package signal

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Status values for a signal's lifecycle.
const (
	StatusPending  = "pending"  // waiting to be executed
	StatusActive   = "active"   // order placed, position open
	StatusFilled   = "filled"   // position closed with P&L
	StatusExpired  = "expired"  // past expires_at without being acted on
	StatusRejected = "rejected" // price moved too far from signal price
	StatusDeclined = "declined" // user said N in semi mode
)

// Signal is a trade instruction that any source can write into the signal bus.
type Signal struct {
	ID          string     `json:"id"`
	Symbol      string     `json:"symbol"`
	Strategy    string     `json:"strategy"`              // "earnings" | "bounce" | "daytrader" | "manual"
	Action      string     `json:"action"`                // "enter" | "exit"
	EntryLimit  float64    `json:"entry_limit"`           // limit price for entry
	Stop        float64    `json:"stop"`                  // protective stop price
	Target      float64    `json:"target,omitempty"`      // 0 = strategy-defined exit
	Qty         int        `json:"qty"`                   // shares
	Reason      string     `json:"reason"`                // human-readable rationale
	ExpiresAt   time.Time  `json:"expires_at"`            // signal invalid after this time
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FilledAt    *time.Time `json:"filled_at,omitempty"`
	FilledPrice float64    `json:"filled_price,omitempty"`
	EntryOrderID string    `json:"entry_order_id,omitempty"`
	StopOrderID  string    `json:"stop_order_id,omitempty"`
	Notes       string     `json:"notes,omitempty"` // rejection reason, slippage info, etc.
}

// IsPending returns true if the signal is waiting to be acted on.
func (s Signal) IsPending() bool { return s.Status == StatusPending }

// IsExpired returns true if the signal is past its expiry time.
func (s Signal) IsExpired() bool { return time.Now().UTC().After(s.ExpiresAt) }

// GenerateID creates a deterministic ID for a signal so duplicate signals
// (same symbol + strategy + day) are not added twice.
func GenerateID(strategy, symbol string, t time.Time) string {
	key := fmt.Sprintf("%s-%s-%s", strategy, symbol, t.Format("2006-01-02"))
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash[:6])
}

// Bus manages reading and writing the signals.json file.
type Bus struct {
	mu      sync.Mutex
	path    string
	signals []Signal
}

// String returns the file path (used in fmt.Printf).
func (b *Bus) String() string { return b.path }

// Open loads (or creates) the signal bus at path.
func Open(path string) (*Bus, error) {
	b := &Bus{path: path}
	if err := b.load(); err != nil {
		return nil, err
	}
	return b, nil
}

// Pending returns all pending signals for the given strategy.
// Pass "" to get all pending signals regardless of strategy.
func (b *Bus) Pending(strategy string) []Signal {
	b.mu.Lock()
	defer b.mu.Unlock()
	var out []Signal
	for _, s := range b.signals {
		if s.Status == StatusPending && (strategy == "" || s.Strategy == strategy) {
			out = append(out, s)
		}
	}
	return out
}

// All returns a snapshot of every signal in the bus.
func (b *Bus) All() []Signal {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]Signal, len(b.signals))
	copy(out, b.signals)
	return out
}

// Add appends a new signal unless an identical ID already exists.
// Returns (true, nil) if the signal was added; (false, nil) if already present.
func (b *Bus) Add(sig Signal) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.signals {
		if s.ID == sig.ID {
			return false, nil // already exists
		}
	}
	sig.CreatedAt = time.Now().UTC()
	sig.UpdatedAt = sig.CreatedAt
	b.signals = append(b.signals, sig)
	return true, b.flush()
}

// Reload re-reads the file from disk — picks up signals written by other processes.
func (b *Bus) Reload() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.load()
}

// Update applies fn to the signal with the given ID and saves.
func (b *Bus) Update(id string, fn func(*Signal)) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	for i := range b.signals {
		if b.signals[i].ID == id {
			fn(&b.signals[i])
			b.signals[i].UpdatedAt = time.Now().UTC()
			return b.flush()
		}
	}
	return fmt.Errorf("signal %s not found", id)
}

func (b *Bus) load() error {
	data, err := os.ReadFile(b.path)
	if os.IsNotExist(err) {
		b.signals = nil
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &b.signals)
}

func (b *Bus) flush() error {
	data, err := json.MarshalIndent(b.signals, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(b.path, data, 0644)
}
