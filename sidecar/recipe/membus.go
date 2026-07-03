package recipe

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

// MemBus is an in-memory signal bus for a single recipe run.
type MemBus struct {
	mu       sync.RWMutex
	signals  []RecipeSignal
	onChange func(RecipeSignal)
}

// NewMemBus creates a new in-memory signal bus.
func NewMemBus(onChange func(RecipeSignal)) *MemBus {
	return &MemBus{onChange: onChange}
}

// Add records a new signal and triggers the onChange callback.
func (b *MemBus) Add(sig RecipeSignal) {
	b.mu.Lock()
	if sig.ID == "" {
		sig.ID = generateID(sig.RecipeID, sig.Symbol)
	}
	if sig.CreatedAt.IsZero() {
		sig.CreatedAt = time.Now()
	}
	if sig.Status == "" {
		sig.Status = "pending"
	}
	b.signals = append(b.signals, sig)
	b.mu.Unlock()

	if b.onChange != nil {
		b.onChange(sig)
	}
}

// Update changes the status of a signal by ID and triggers onChange.
func (b *MemBus) Update(id, status, entryID, stopID string) {
	b.mu.Lock()
	for i := range b.signals {
		if b.signals[i].ID == id {
			b.signals[i].Status = status
			if entryID != "" {
				b.signals[i].EntryID = entryID
			}
			if stopID != "" {
				b.signals[i].StopID = stopID
			}
			sig := b.signals[i]
			b.mu.Unlock()
			if b.onChange != nil {
				b.onChange(sig)
			}
			return
		}
	}
	b.mu.Unlock()
}

// All returns a copy of all signals.
func (b *MemBus) All() []RecipeSignal {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]RecipeSignal, len(b.signals))
	copy(out, b.signals)
	return out
}

// Pending returns signals with status "pending".
func (b *MemBus) Pending() []RecipeSignal {
	b.mu.RLock()
	defer b.mu.RUnlock()
	var out []RecipeSignal
	for _, s := range b.signals {
		if s.Status == "pending" {
			out = append(out, s)
		}
	}
	return out
}

func generateID(recipeID, symbol string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%s-%d", recipeID, symbol, time.Now().UnixNano())))
	return fmt.Sprintf("%x", h[:3])
}
