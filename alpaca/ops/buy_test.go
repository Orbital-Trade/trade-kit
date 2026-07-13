package ops

import "testing"

func TestBuyMarketPaper(t *testing.T) {
	m := newMock(true)
	res, err := BuyMarket(m, "AAPL", 10)
	if err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", res.Mode)
	}
	if res.Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", res.Symbol)
	}
	if res.Type != "MKT" {
		t.Errorf("Type = %q, want MKT", res.Type)
	}
}

func TestBuyLimitPaper(t *testing.T) {
	m := newMock(true)
	res, err := BuyLimit(m, "AAPL", 10, 150.00)
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
	m.setResponse("/v2/orders", map[string]interface{}{"id": "order-abc-123"})

	res, err := BuyMarket(m, "AAPL", 10)
	if err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if res.OrderID != "order-abc-123" {
		t.Errorf("OrderID = %q, want order-abc-123", res.OrderID)
	}
	if res.Mode != "LIVE" {
		t.Errorf("Mode = %q, want LIVE", res.Mode)
	}
}

func TestBuyWithStopsPaper(t *testing.T) {
	m := newMock(true)
	res, err := BuyWithStops(m, "TSLA", 5, 200.00, 190.00, 220.00)
	if err != nil {
		t.Fatalf("BuyWithStops: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("Type = %q, want LMT", res.Type)
	}
}
