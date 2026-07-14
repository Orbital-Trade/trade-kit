package ops

import "testing"

func TestBuyMarketPaper(t *testing.T) {
	m := newMock(true)
	res, err := BuyMarket(m, "RELIANCE", 10)
	if err != nil {
		t.Fatalf("BuyMarket: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", res.Mode)
	}
	if res.Symbol != "RELIANCE" {
		t.Errorf("Symbol = %q, want RELIANCE", res.Symbol)
	}
	if res.Type != "MKT" {
		t.Errorf("Type = %q, want MKT", res.Type)
	}
}

func TestBuyLimitPaper(t *testing.T) {
	m := newMock(true)
	res, err := BuyLimit(m, "TCS", 5, 3500.00)
	if err != nil {
		t.Fatalf("BuyLimit: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("Type = %q, want LMT", res.Type)
	}
	if res.Price != 3500.00 {
		t.Errorf("Price = %v, want 3500.00", res.Price)
	}
}

func TestBuyMarketLive(t *testing.T) {
	m := newMock(false)
	m.setResponse("/orders/regular", map[string]interface{}{
		"status": "success",
		"data":   map[string]interface{}{"order_id": "order-abc-123"},
	})

	res, err := BuyMarket(m, "INFY", 10)
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

func TestBuyIntradayPaper(t *testing.T) {
	m := newMock(true)
	res, err := BuyIntraday(m, "RELIANCE", 20)
	if err != nil {
		t.Fatalf("BuyIntraday: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", res.Mode)
	}
}

func TestParseExchange(t *testing.T) {
	tests := []struct {
		input      string
		wantExch   string
		wantSymbol string
	}{
		{"RELIANCE", "NSE", "RELIANCE"},
		{"BSE:RELIANCE", "BSE", "RELIANCE"},
		{"NFO:NIFTY23JUNFUT", "NFO", "NIFTY23JUNFUT"},
		{"reliance", "NSE", "RELIANCE"},
		{"bse:reliance", "BSE", "RELIANCE"},
	}
	for _, tt := range tests {
		exch, sym := parseExchange(tt.input)
		if exch != tt.wantExch {
			t.Errorf("parseExchange(%q) exchange = %q, want %q", tt.input, exch, tt.wantExch)
		}
		if sym != tt.wantSymbol {
			t.Errorf("parseExchange(%q) symbol = %q, want %q", tt.input, sym, tt.wantSymbol)
		}
	}
}
