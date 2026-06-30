package ops

// Instrument lookup — resolves ticker symbols to eToro instrument IDs.
//
// eToro uses internal numeric instrument IDs, not ticker symbols, for all
// trading operations. This file provides symbol → ID resolution via the
// Asset Explorer API, with a local cache to minimize API calls.

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

// Instrument represents an eToro tradeable asset.
type Instrument struct {
	ID           int    `json:"InstrumentID"`
	Symbol       string `json:"SymbolFull"`
	Name         string `json:"InstrumentDisplayName"`
	Exchange     string `json:"ExchangeID"`
	Type         string `json:"InstrumentTypeID"`
	IsActive     bool   `json:"IsActive"`
}

// instrumentCache stores resolved symbol → instrument mappings.
var (
	cacheMu sync.RWMutex
	cache   = make(map[string]Instrument)
)

// ResolveInstrument looks up an eToro instrument ID from a ticker symbol.
// Results are cached for the lifetime of the process.
func ResolveInstrument(c Caller, symbol string) (Instrument, error) {
	sym := strings.ToUpper(strings.TrimSpace(symbol))

	// Check cache first.
	cacheMu.RLock()
	if inst, ok := cache[sym]; ok {
		cacheMu.RUnlock()
		return inst, nil
	}
	cacheMu.RUnlock()

	// Search via Asset Explorer API.
	data, err := c.Get("/api/v1/asset-explorer/instruments", map[string]string{
		"query": sym,
	})
	if err != nil {
		return Instrument{}, fmt.Errorf("resolve_instrument %s: %w", sym, err)
	}

	var results []Instrument
	if err := json.Unmarshal(data, &results); err != nil {
		// Try wrapped response.
		var wrapper struct {
			Instruments []Instrument `json:"instruments"`
		}
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return Instrument{}, fmt.Errorf("resolve_instrument %s: parse: %w", sym, err)
		}
		results = wrapper.Instruments
	}

	// Find exact symbol match.
	for _, inst := range results {
		if strings.EqualFold(inst.Symbol, sym) {
			cacheMu.Lock()
			cache[sym] = inst
			cacheMu.Unlock()
			return inst, nil
		}
	}

	// Fall back to first active result.
	for _, inst := range results {
		if inst.IsActive {
			cacheMu.Lock()
			cache[sym] = inst
			cacheMu.Unlock()
			return inst, nil
		}
	}

	if len(results) > 0 {
		cacheMu.Lock()
		cache[sym] = results[0]
		cacheMu.Unlock()
		return results[0], nil
	}

	return Instrument{}, fmt.Errorf("resolve_instrument %s: no matching instrument found", sym)
}

// SearchInstruments searches for instruments by name or symbol.
func SearchInstruments(c Caller, query string) ([]Instrument, error) {
	data, err := c.Get("/api/v1/asset-explorer/instruments", map[string]string{
		"query": query,
	})
	if err != nil {
		return nil, fmt.Errorf("search_instruments: %w", err)
	}

	var results []Instrument
	if err := json.Unmarshal(data, &results); err != nil {
		var wrapper struct {
			Instruments []Instrument `json:"instruments"`
		}
		if err2 := json.Unmarshal(data, &wrapper); err2 != nil {
			return nil, fmt.Errorf("search_instruments: parse: %w", err)
		}
		results = wrapper.Instruments
	}

	return results, nil
}
