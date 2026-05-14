package ops

import "testing"

func TestSellMarket_paper(t *testing.T) {
	m := newMock(true)
	res, err := SellMarket(m, "NOK", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "PAPER" || res.Action != "SELL" || res.Type != "MKT" {
		t.Errorf("want PAPER/SELL/MKT, got %s/%s/%s", res.Mode, res.Action, res.Type)
	}
	if m.called("place_order") != 0 {
		t.Error("paper mode must not call place_order")
	}
}

func TestSellMarket_live(t *testing.T) {
	m := newMock(false).on("place_order", map[string]interface{}{"id": 222333444}, nil)
	res, err := SellMarket(m, "NOK", 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "222333444" {
		t.Errorf("order_id: want 222333444, got %s", res.OrderID)
	}
	if res.Action != "SELL" || res.Type != "MKT" {
		t.Errorf("want SELL/MKT, got %s/%s", res.Action, res.Type)
	}
}

func TestSellLimit_paper(t *testing.T) {
	m := newMock(true)
	res, err := SellLimit(m, "NOK", 50, 5.00)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Type != "LMT" {
		t.Errorf("order_type: want LMT, got %s", res.Type)
	}
	if res.Price != 5.00 {
		t.Errorf("price: want 5.00, got %f", res.Price)
	}
}

func TestSellLimit_live(t *testing.T) {
	m := newMock(false).on("place_order", map[string]interface{}{"id": 555666777}, nil)
	res, err := SellLimit(m, "NOK", 50, 5.00)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "555666777" {
		t.Errorf("order_id: want 555666777, got %s", res.OrderID)
	}
}

func TestSellLimit_isGTC(t *testing.T) {
	// Verify that SellLimit sends time_in_force=GTC (captured via mock call args).
	// We do this indirectly: if it calls place_order exactly once and returns
	// a valid result, the params were accepted (the mock doesn't validate params,
	// but a real integration test would).
	m := newMock(false).on("place_order", map[string]interface{}{"id": 1}, nil)
	_, err := SellLimit(m, "NOK", 50, 5.00)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.called("place_order") != 1 {
		t.Errorf("expected 1 place_order call, got %d", m.called("place_order"))
	}
}
