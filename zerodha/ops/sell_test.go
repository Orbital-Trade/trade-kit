package ops

import "testing"

func TestSellMarketPaper(t *testing.T) {
	m := newMock(true)
	res, err := SellMarket(m, "RELIANCE", 10)
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
	res, err := SellLimit(m, "TCS", 10, 3600.00)
	if err != nil {
		t.Fatalf("SellLimit: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("Type = %q, want LMT", res.Type)
	}
}

func TestSellMarketLive(t *testing.T) {
	m := newMock(false)
	m.setResponse("/orders/regular", map[string]interface{}{
		"status": "success",
		"data":   map[string]interface{}{"order_id": "sell-order-456"},
	})

	res, err := SellMarket(m, "INFY", 5)
	if err != nil {
		t.Fatalf("SellMarket: %v", err)
	}
	if res.OrderID != "sell-order-456" {
		t.Errorf("OrderID = %q, want sell-order-456", res.OrderID)
	}
	if res.Mode != "LIVE" {
		t.Errorf("Mode = %q, want LIVE", res.Mode)
	}
}
