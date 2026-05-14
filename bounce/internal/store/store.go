package store

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

const defaultPath = "bounce-trades.json"

type Trade struct {
	Symbol       string     `json:"symbol"`
	EntryPrice   float64    `json:"entry_price"`
	EntryRSI     float64    `json:"entry_rsi"`
	StopPrice    float64    `json:"stop_price"`
	TargetRSI    float64    `json:"target_rsi"`
	Qty          int        `json:"qty"`
	EntryOrderID string     `json:"entry_order_id"`
	StopOrderID  string     `json:"stop_order_id"`
	Status       string     `json:"status"` // open | stopped | exited | expired
	EntryAt      time.Time  `json:"entry_at"`
	ExpiresAt    time.Time  `json:"expires_at"` // forced exit after max_hold_days
	ExitAt       *time.Time `json:"exit_at,omitempty"`
	ExitPrice    float64    `json:"exit_price,omitempty"`
	PnL          float64    `json:"pnl,omitempty"`
}

type Store struct {
	mu     sync.Mutex
	path   string
	trades []Trade
}

func New(path string) (*Store, error) {
	if path == "" {
		path = defaultPath
	}
	s := &Store{path: path}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) FindOpen(symbol string) *Trade {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.trades {
		if s.trades[i].Symbol == symbol && s.trades[i].Status == "open" {
			return &s.trades[i]
		}
	}
	return nil
}

func (s *Store) Save(t Trade) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trades = append(s.trades, t)
	return s.flush()
}

func (s *Store) Close(symbol, status string, exitPrice float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.trades {
		if s.trades[i].Symbol == symbol && s.trades[i].Status == "open" {
			now := time.Now().UTC()
			s.trades[i].Status = status
			s.trades[i].ExitAt = &now
			s.trades[i].ExitPrice = exitPrice
			s.trades[i].PnL = (exitPrice - s.trades[i].EntryPrice) * float64(s.trades[i].Qty)
			return s.flush()
		}
	}
	return fmt.Errorf("no open trade for %s", symbol)
}

func (s *Store) All() []Trade {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Trade, len(s.trades))
	copy(out, s.trades)
	return out
}

func (s *Store) load() error {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.trades)
}

func (s *Store) flush() error {
	data, err := json.MarshalIndent(s.trades, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}
