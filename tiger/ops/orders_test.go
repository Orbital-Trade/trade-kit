package ops

import (
	"encoding/json"
	"testing"
)

// realTigerOrders mirrors what the actual Tiger API returns after
// client.Call unwraps the outer JSON string. Fields are camelCase.
var realTigerOrders = []map[string]interface{}{
	{
		"id":              123456789,
		"symbol":          "NOK",
		"action":          "BUY",
		"orderType":       "LMT",
		"totalQuantity":   100,
		"limitPrice":      4.50,
		"auxPrice":        0.0,
		"filledQuantity":  0,
		"status":          "Submitted",
		"timeInForce":     "DAY",
	},
	{
		"id":              987654321,
		"symbol":          "NOK",
		"action":          "SELL",
		"orderType":       "STP",
		"totalQuantity":   100,
		"limitPrice":      0.0,
		"auxPrice":        4.20,
		"filledQuantity":  0,
		"status":          "Submitted",
		"timeInForce":     "GTC",
	},
}

func TestGetOrders_directArray(t *testing.T) {
	// Direct array response (no wrapper).
	m := newMock(false).on("orders", realTigerOrders, nil)

	orders, err := GetOrders(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders, got %d", len(orders))
	}

	buy := orders[0]
	if buy.ID != "123456789" {
		t.Errorf("id: want 123456789, got %s", buy.ID)
	}
	if buy.Symbol != "NOK" {
		t.Errorf("symbol: want NOK, got %s", buy.Symbol)
	}
	if buy.OrderType != "LMT" {
		t.Errorf("order_type (from orderType): want LMT, got %s", buy.OrderType)
	}
	if buy.Quantity != 100 {
		t.Errorf("quantity (from totalQuantity): want 100, got %d", buy.Quantity)
	}
	if buy.LimitPrice != 4.50 {
		t.Errorf("limit_price (from limitPrice): want 4.50, got %f", buy.LimitPrice)
	}
	if buy.StopPrice != 0 {
		t.Errorf("stop_price (from auxPrice): want 0, got %f", buy.StopPrice)
	}
	if buy.TimeInForce != "DAY" {
		t.Errorf("time_in_force (from timeInForce): want DAY, got %s", buy.TimeInForce)
	}

	stop := orders[1]
	if stop.OrderType != "STP" {
		t.Errorf("order_type: want STP, got %s", stop.OrderType)
	}
	// auxPrice becomes StopPrice
	if stop.StopPrice != 4.20 {
		t.Errorf("stop_price (from auxPrice): want 4.20, got %f", stop.StopPrice)
	}
	if stop.TimeInForce != "GTC" {
		t.Errorf("time_in_force: want GTC, got %s", stop.TimeInForce)
	}
}

func TestGetOrders_itemsWrapper(t *testing.T) {
	// Tiger wraps orders in {"items":[...]} after double-decode.
	wrapped := map[string]interface{}{
		"items": realTigerOrders,
	}
	m := newMock(false).on("orders", wrapped, nil)

	orders, err := GetOrders(m)
	if err != nil {
		t.Fatalf("unexpected error for items-wrapped response: %v", err)
	}
	if len(orders) != 2 {
		t.Fatalf("expected 2 orders from items wrapper, got %d", len(orders))
	}
}

func TestGetOrders_empty(t *testing.T) {
	m := newMock(false).on("orders", []interface{}{}, nil)
	orders, err := GetOrders(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected empty, got %d", len(orders))
	}
}

func TestGetOrders_nullData(t *testing.T) {
	// Tiger normalises "" to null for empty list responses.
	m := newMock(false)
	m.responses["orders"] = append(m.responses["orders"], json.RawMessage("null"))
	m.errs["orders"] = append(m.errs["orders"], nil)

	orders, err := GetOrders(m)
	if err != nil {
		t.Fatalf("expected empty result for null data, got error: %v", err)
	}
	if len(orders) != 0 {
		t.Errorf("expected 0 orders, got %d", len(orders))
	}
}

func TestGetOrders_apiError(t *testing.T) {
	m := newMock(false).onErr("orders", "session expired")
	_, err := GetOrders(m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetOrders_parseError(t *testing.T) {
	m := newMock(false)
	m.responses["orders"] = append(m.responses["orders"], []byte(`{not valid json`))
	m.errs["orders"] = append(m.errs["orders"], nil)

	_, err := GetOrders(m)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestParseOrders_directArray(t *testing.T) {
	data, _ := json.Marshal(realTigerOrders)
	items, err := parseOrders(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].OrderType != "LMT" {
		t.Errorf("orderType: want LMT, got %s", items[0].OrderType)
	}
	if items[1].AuxPrice != 4.20 {
		t.Errorf("auxPrice: want 4.20, got %f", items[1].AuxPrice)
	}
}

func TestParseOrders_itemsWrapper(t *testing.T) {
	wrapped, _ := json.Marshal(map[string]interface{}{"items": realTigerOrders})
	items, err := parseOrders(wrapped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestParseOrders_invalid(t *testing.T) {
	_, err := parseOrders(json.RawMessage(`"not an object or array"`))
	if err == nil {
		t.Fatal("expected error for string input, got nil")
	}
}
