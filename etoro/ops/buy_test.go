package ops

import (
	"testing"
)

func TestBuyMarketPaper(t *testing.T) {
	m := newMock(true)

	res, err := BuyMarket(m, "AAPL", 200)
	if err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if res.OrderID != "PAPER" {
		t.Errorf("OrderID = %q, want PAPER", res.OrderID)
	}
	if res.Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", res.Mode)
	}
	if res.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", res.Symbol)
	}
	if res.Action != "BUY" {
		t.Errorf("Action = %q, want BUY", res.Action)
	}
	if res.Type != "MKT" {
		t.Errorf("Type = %q, want MKT", res.Type)
	}
}

func TestBuyLimitPaper(t *testing.T) {
	m := newMock(true)

	res, err := BuyLimit(m, "AAPL", 200, 150.00)
	if err != nil {
		t.Fatalf("BuyLimit: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("Type = %q, want LMT", res.Type)
	}
	if res.Price != 150.00 {
		t.Errorf("Price = %v, want 150.00", res.Price)
	}
}

func TestBuyMarketLive(t *testing.T) {
	m := newMock(false)
	// Set up instrument resolution.
	m.setResponse("/api/v1/asset-explorer/instruments", []map[string]interface{}{
		{"InstrumentID": 1001, "SymbolFull": "AAPL", "InstrumentDisplayName": "Apple", "IsActive": true},
	})
	// Set up order placement.
	m.setResponse("/api/v1/trading/real/orders", map[string]interface{}{
		"OrderID":    99887766,
		"PositionID": 0,
	})

	res, err := BuyMarket(m, "AAPL", 200)
	if err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if res.OrderID != "99887766" {
		t.Errorf("OrderID = %q, want 99887766", res.OrderID)
	}
	if res.Mode != "LIVE" {
		t.Errorf("Mode = %q, want LIVE", res.Mode)
	}
	if res.InstrID != 1001 {
		t.Errorf("InstrID = %d, want 1001", res.InstrID)
	}
}

func TestBuyWithStopsPaper(t *testing.T) {
	m := newMock(true)

	res, err := BuyWithStops(m, "TSLA", 500, 0, 180.00, 250.00)
	if err != nil {
		t.Fatalf("BuyWithStops: %v", err)
	}
	if res.Type != "MKT" {
		t.Errorf("Type = %q, want MKT (no limit price)", res.Type)
	}

	res, err = BuyWithStops(m, "TSLA", 500, 200.00, 180.00, 250.00)
	if err != nil {
		t.Fatalf("BuyWithStops: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("Type = %q, want LMT (limit price set)", res.Type)
	}
}
