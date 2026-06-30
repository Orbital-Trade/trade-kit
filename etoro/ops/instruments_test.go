package ops

import (
	"testing"
)

func TestResolveInstrument(t *testing.T) {
	// Clear cache from prior tests.
	cacheMu.Lock()
	cache = make(map[string]Instrument)
	cacheMu.Unlock()

	m := newMock(false)
	m.setResponse("/api/v1/asset-explorer/instruments", []map[string]interface{}{
		{"InstrumentID": 1001, "SymbolFull": "AAPL", "InstrumentDisplayName": "Apple Inc", "IsActive": true},
		{"InstrumentID": 1002, "SymbolFull": "AAPLQ", "InstrumentDisplayName": "Apple Options", "IsActive": true},
	})

	inst, err := ResolveInstrument(m, "AAPL")
	if err != nil {
		t.Fatalf("ResolveInstrument: %v", err)
	}
	if inst.ID != 1001 {
		t.Errorf("ID = %d, want 1001", inst.ID)
	}
	if inst.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", inst.Symbol)
	}

	// Second call should use cache (mock won't be consulted).
	inst2, err := ResolveInstrument(m, "AAPL")
	if err != nil {
		t.Fatalf("ResolveInstrument cached: %v", err)
	}
	if inst2.ID != 1001 {
		t.Errorf("cached ID = %d, want 1001", inst2.ID)
	}
}

func TestResolveInstrumentNotFound(t *testing.T) {
	cacheMu.Lock()
	cache = make(map[string]Instrument)
	cacheMu.Unlock()

	m := newMock(false)
	m.setResponse("/api/v1/asset-explorer/instruments", []map[string]interface{}{})

	_, err := ResolveInstrument(m, "ZZZZZ")
	if err == nil {
		t.Fatal("expected error for unknown instrument, got nil")
	}
}

func TestSearchInstruments(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/asset-explorer/instruments", []map[string]interface{}{
		{"InstrumentID": 2001, "SymbolFull": "TSLA", "InstrumentDisplayName": "Tesla Inc", "IsActive": true},
		{"InstrumentID": 2002, "SymbolFull": "TM", "InstrumentDisplayName": "Toyota Motor", "IsActive": true},
	})

	results, err := SearchInstruments(m, "tesla")
	if err != nil {
		t.Fatalf("SearchInstruments: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("len(results) = %d, want 2", len(results))
	}
}
