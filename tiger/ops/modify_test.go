package ops

import (
	"encoding/json"
	"testing"
)

func TestModifyOrder_paperMode(t *testing.T) {
	// Paper mode: no API call, returns PAPER_MODIFIED immediately.
	m := newMock(true) // paper=true
	res, err := ModifyOrder(m, "123456789", ModifyParams{LimitPrice: 4.60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "PAPER_MODIFIED" {
		t.Errorf("status: want PAPER_MODIFIED, got %s", res.Status)
	}
	if res.OrderID != "123456789" {
		t.Errorf("order_id: want 123456789, got %s", res.OrderID)
	}
	// Paper mode must not call any API.
	if m.called("orders") > 0 || m.called("modify_order") > 0 {
		t.Error("paper mode must not call the API")
	}
}

func TestModifyOrder_live_success(t *testing.T) {
	// Live mode: fetches orders, finds the order, calls modify_order.
	m := newMock(false).
		on("orders", realTigerOrders, nil).
		on("modify_order", map[string]interface{}{"id": 123456789}, nil)

	res, err := ModifyOrder(m, "123456789", ModifyParams{LimitPrice: 4.60})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "MODIFIED" {
		t.Errorf("status: want MODIFIED, got %s", res.Status)
	}
	if res.Mode != "LIVE" {
		t.Errorf("mode: want LIVE, got %s", res.Mode)
	}
	if m.called("orders") != 1 {
		t.Errorf("expected 1 orders call, got %d", m.called("orders"))
	}
	if m.called("modify_order") != 1 {
		t.Errorf("expected 1 modify_order call, got %d", m.called("modify_order"))
	}
}

func TestModifyOrder_orderNotFound(t *testing.T) {
	// Order ID doesn't exist in the orders list.
	m := newMock(false).on("orders", realTigerOrders, nil)

	_, err := ModifyOrder(m, "999999999", ModifyParams{LimitPrice: 4.60})
	if err == nil {
		t.Fatal("expected error for unknown order ID, got nil")
	}
}

func TestModifyOrder_ordersApiError(t *testing.T) {
	m := newMock(false).onErr("orders", "session expired")
	_, err := ModifyOrder(m, "123456789", ModifyParams{LimitPrice: 4.60})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestModifyOrder_modifyApiError(t *testing.T) {
	m := newMock(false).
		on("orders", realTigerOrders, nil).
		onErr("modify_order", "order already filled")

	_, err := ModifyOrder(m, "123456789", ModifyParams{LimitPrice: 4.60})
	if err == nil {
		t.Fatal("expected error from modify_order API, got nil")
	}
}

func TestModifyOrder_stopPrice(t *testing.T) {
	// Modifying a stop order's aux_price.
	m := newMock(false).
		on("orders", realTigerOrders, nil).
		on("modify_order", map[string]interface{}{"id": 987654321}, nil)

	res, err := ModifyOrder(m, "987654321", ModifyParams{StopPrice: 4.00})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "MODIFIED" {
		t.Errorf("status: want MODIFIED, got %s", res.Status)
	}
}

func TestModifyOrder_nullOrders(t *testing.T) {
	// If orders returns null (empty), order will not be found.
	m := newMock(false)
	m.responses["orders"] = append(m.responses["orders"], json.RawMessage("null"))
	m.errs["orders"] = append(m.errs["orders"], nil)

	_, err := ModifyOrder(m, "123456789", ModifyParams{LimitPrice: 4.60})
	if err == nil {
		t.Fatal("expected error for null orders, got nil")
	}
}
