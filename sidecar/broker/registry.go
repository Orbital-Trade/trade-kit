package broker

import (
	"fmt"
	"sync"
)

// Registry manages all broker adapters.
type Registry struct {
	mu      sync.RWMutex
	brokers map[string]BrokerAdapter
	paper   bool
}

// NewRegistry creates a registry with all three broker adapters pre-registered.
func NewRegistry() *Registry {
	r := &Registry{
		brokers: make(map[string]BrokerAdapter),
		paper:   true, // default to paper/demo mode
	}
	r.brokers["tiger"] = NewTigerAdapter()
	r.brokers["moomoo"] = NewMoomooAdapter()
	r.brokers["etoro"] = NewEtoroAdapter()
	return r
}

// Get returns a broker adapter by ID.
func (r *Registry) Get(id string) (BrokerAdapter, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	b, ok := r.brokers[id]
	if !ok {
		return nil, fmt.Errorf("unknown broker: %s", id)
	}
	return b, nil
}

// List returns the status of all registered brokers.
func (r *Registry) List() []BrokerStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]BrokerStatus, 0, len(r.brokers))
	for _, b := range r.brokers {
		s := BrokerStatus{
			ID:        b.ID(),
			Name:      b.Name(),
			Connected: b.Connected(),
		}
		if b.Connected() {
			if acct, err := b.Account(); err == nil {
				s.Account = &acct
			}
		}
		out = append(out, s)
	}
	return out
}

// SetPaperMode propagates paper mode to all broker adapters.
func (r *Registry) SetPaperMode(paper bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.paper = paper
	for _, b := range r.brokers {
		b.SetPaper(paper)
	}
}

// IsPaper returns the global paper mode setting.
func (r *Registry) IsPaper() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.paper
}
