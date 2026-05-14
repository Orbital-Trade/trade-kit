package ops

import "testing"

func TestBuyMarket_paper(t *testing.T) {
	m := newMock(true)
	res, err := BuyMarket(m, "NOK", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("mode: want PAPER, got %s", res.Mode)
	}
	if res.Type != "MKT" {
		t.Errorf("order_type: want MKT, got %s", res.Type)
	}
	if m.called("place_order") != 0 {
		t.Error("paper mode must not call place_order")
	}
}

func TestBuyMarket_live(t *testing.T) {
	m := newMock(false).on("place_order", map[string]interface{}{"id": 111222333}, nil)
	res, err := BuyMarket(m, "NOK", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "111222333" {
		t.Errorf("order_id: want 111222333, got %s", res.OrderID)
	}
	if res.Mode != "LIVE" {
		t.Errorf("mode: want LIVE, got %s", res.Mode)
	}
	if res.Action != "BUY" || res.Type != "MKT" {
		t.Errorf("action/type: want BUY/MKT, got %s/%s", res.Action, res.Type)
	}
}

func TestBuyLimit_paper(t *testing.T) {
	m := newMock(true)
	res, err := BuyLimit(m, "NOK", 50, 4.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("order_type: want LMT, got %s", res.Type)
	}
	if res.Price != 4.50 {
		t.Errorf("price: want 4.50, got %f", res.Price)
	}
}

func TestBuyLimit_live(t *testing.T) {
	m := newMock(false).on("place_order", map[string]interface{}{"id": 444555666}, nil)
	res, err := BuyLimit(m, "NOK", 50, 4.50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "444555666" {
		t.Errorf("order_id: want 444555666, got %s", res.OrderID)
	}
}

func TestBuyLimit_zeroID(t *testing.T) {
	// API returning ID=0 means the order was rejected — must be an error.
	m := newMock(false).on("place_order", map[string]interface{}{"id": 0}, nil)
	_, err := BuyLimit(m, "NOK", 50, 4.50)
	if err == nil {
		t.Fatal("expected error for ID=0, got nil")
	}
}

func TestBuyMarket_apiError(t *testing.T) {
	m := newMock(false).onErr("place_order", "insufficient funds")
	_, err := BuyMarket(m, "NOK", 100)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
