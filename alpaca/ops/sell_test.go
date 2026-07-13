package ops

import "testing"

func TestSellMarketPaper(t *testing.T) {
	m := newMock(true)
	res, err := SellMarket(m, "AAPL", 10)
	if err != nil {
		t.Fatalf("SellMarket: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", res.Mode)
	}
	if res.Action != "SELL" {
		t.Errorf("Action = %q, want SELL", res.Action)
	}
}

func TestSellLimitPaper(t *testing.T) {
	m := newMock(true)
	res, err := SellLimit(m, "AAPL", 10, 160.00)
	if err != nil {
		t.Fatalf("SellLimit: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("Type = %q, want LMT", res.Type)
	}
}
