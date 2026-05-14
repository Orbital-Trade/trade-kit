package ops

import "testing"

func TestCancelOrder_paper(t *testing.T) {
	m := newMock(true) // paper=true
	res, err := CancelOrder(m, "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "PAPER_CANCELLED" {
		t.Errorf("status: want PAPER_CANCELLED, got %s", res.Status)
	}
	if res.OrderID != "123456" {
		t.Errorf("order_id: want 123456, got %s", res.OrderID)
	}
	// Paper mode must not call the API.
	if m.called("cancel_order") != 0 {
		t.Error("paper mode called cancel_order — it must not")
	}
}

func TestCancelOrder_live(t *testing.T) {
	m := newMock(false).on("cancel_order", map[string]interface{}{"result": "ok"}, nil)
	res, err := CancelOrder(m, "789012")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "CANCELLED" {
		t.Errorf("status: want CANCELLED, got %s", res.Status)
	}
	if m.called("cancel_order") != 1 {
		t.Errorf("expected 1 cancel_order call, got %d", m.called("cancel_order"))
	}
}

func TestCancelOrder_apiError(t *testing.T) {
	m := newMock(false).onErr("cancel_order", "order not found")
	_, err := CancelOrder(m, "000")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
