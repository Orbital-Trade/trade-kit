package ops

import "testing"

func TestSetStopLoss_paper(t *testing.T) {
	m := newMock(true)
	res, err := SetStopLoss(m, "NOK", 100, 4.20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("mode: want PAPER, got %s", res.Mode)
	}
	if res.Type != "STP" {
		t.Errorf("order_type: want STP, got %s", res.Type)
	}
	if res.Price != 4.20 {
		t.Errorf("price: want 4.20, got %f", res.Price)
	}
	if res.Action != "SELL" {
		t.Errorf("action: want SELL, got %s", res.Action)
	}
	if m.called("place_order") != 0 {
		t.Error("paper mode must not call place_order")
	}
}

func TestSetStopLoss_live(t *testing.T) {
	m := newMock(false).on("place_order", map[string]interface{}{"id": 333444555}, nil)
	res, err := SetStopLoss(m, "NOK", 100, 4.20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "333444555" {
		t.Errorf("order_id: want 333444555, got %s", res.OrderID)
	}
	if res.Type != "STP" {
		t.Errorf("order_type: want STP, got %s", res.Type)
	}
}

func TestSetStopLoss_apiError(t *testing.T) {
	m := newMock(false).onErr("place_order", "position not found")
	_, err := SetStopLoss(m, "NOK", 100, 4.20)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
