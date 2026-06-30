package ops

import (
	"testing"
)

func TestGetOrders(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/trading/real/orders", []map[string]interface{}{
		{
			"OrderID":        55667788,
			"InstrumentID":   1001,
			"IsBuy":          true,
			"Amount":         300.00,
			"Units":          2.0,
			"Rate":           150.00,
			"StopLossRate":   145.00,
			"TakeProfitRate": 165.00,
			"Status":         "Pending",
		},
	})

	orders, err := GetOrders(m)
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}
	o := orders[0]
	if o.OrderID != "55667788" {
		t.Errorf("OrderID = %q, want 55667788", o.OrderID)
	}
	if !o.IsBuy {
		t.Error("IsBuy = false, want true")
	}
	if o.Rate != 150.00 {
		t.Errorf("Rate = %v, want 150.00", o.Rate)
	}
	if o.Status != "Pending" {
		t.Errorf("Status = %q, want Pending", o.Status)
	}
}

func TestGetOrdersEmpty(t *testing.T) {
	m := newMock(true)
	m.setResponse("/api/v1/trading/demo/orders", []map[string]interface{}{})

	orders, err := GetOrders(m)
	if err != nil {
		t.Fatalf("GetOrders: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("len(orders) = %d, want 0", len(orders))
	}
}
