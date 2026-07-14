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

func TestSellMarketLive(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST123"
	m.conids["AAPL"] = 265598
	m.setResponse("/v1/api/iserver/account/TEST123/orders", []map[string]interface{}{
		{"order_id": "sell-order-789", "order_status": "Submitted"},
	})

	res, err := SellMarket(m, "AAPL", 10)
	if err != nil {
		t.Fatalf("SellMarket: %v", err)
	}
	if res.OrderID != "sell-order-789" {
		t.Errorf("OrderID = %q, want sell-order-789", res.OrderID)
	}
	if res.Mode != "LIVE" {
		t.Errorf("Mode = %q, want LIVE", res.Mode)
	}
}

func TestSellLimitLive(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST123"
	m.conids["MSFT"] = 272093
	m.setResponse("/v1/api/iserver/account/TEST123/orders", []map[string]interface{}{
		{"order_id": "sell-lmt-321", "order_status": "Submitted"},
	})

	res, err := SellLimit(m, "MSFT", 5, 420.00)
	if err != nil {
		t.Fatalf("SellLimit: %v", err)
	}
	if res.OrderID != "sell-lmt-321" {
		t.Errorf("OrderID = %q, want sell-lmt-321", res.OrderID)
	}
	if res.Price != 420.00 {
		t.Errorf("Price = %v, want 420.00", res.Price)
	}
}
