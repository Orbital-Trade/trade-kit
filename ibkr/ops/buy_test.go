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
	m.accountID = "TEST123"
	m.conids["AAPL"] = 265598
	m.setResponse("/v1/api/iserver/account/TEST123/orders", []map[string]interface{}{
		{"order_id": "order-abc-123", "order_status": "Submitted"},
	})

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

func TestBuyLimitLive(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST123"
	m.conids["TSLA"] = 76792991
	m.setResponse("/v1/api/iserver/account/TEST123/orders", []map[string]interface{}{
		{"order_id": "order-def-456", "order_status": "Submitted"},
	})

	res, err := BuyLimit(m, "TSLA", 5, 200.00)
	if err != nil {
		t.Fatalf("BuyLimit: %v", err)
	}
	if res.OrderID != "order-def-456" {
		t.Errorf("OrderID = %q, want order-def-456", res.OrderID)
	}
	if res.Price != 200.00 {
		t.Errorf("Price = %v, want 200.00", res.Price)
	}
}
