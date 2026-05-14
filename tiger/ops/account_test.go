package ops

import (
	"encoding/json"
	"testing"
)

// realTigerAccount mirrors what the actual Tiger API returns after
// client.Call unwraps the outer JSON string. Fields are flat camelCase —
// no "summary" sub-object.
var realTigerAccount = map[string]interface{}{
	"netLiquidation":     1234.56,
	"cashValue":          500.00,
	"buyingPower":        1000.00,
	"grossPositionValue": 734.56,
}

func TestGetAccount_directArray(t *testing.T) {
	// Direct array response (no wrapper).
	m := newMock(false).on("assets", []map[string]interface{}{realTigerAccount}, nil)

	acct, err := GetAccount(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acct.NetLiquidation != 1234.56 {
		t.Errorf("net_liquidation: want 1234.56, got %f", acct.NetLiquidation)
	}
	if acct.Cash != 500.00 {
		t.Errorf("cash (from cashValue): want 500.00, got %f", acct.Cash)
	}
	if acct.BuyingPower != 1000.00 {
		t.Errorf("buying_power: want 1000.00, got %f", acct.BuyingPower)
	}
	if acct.GrossPositionValue != 734.56 {
		t.Errorf("gross_position_value: want 734.56, got %f", acct.GrossPositionValue)
	}
}

func TestGetAccount_itemsWrapper(t *testing.T) {
	// Tiger wraps account data in {"items":[...]} after double-decode.
	wrapped := map[string]interface{}{
		"items": []map[string]interface{}{realTigerAccount},
	}
	m := newMock(false).on("assets", wrapped, nil)

	acct, err := GetAccount(m)
	if err != nil {
		t.Fatalf("unexpected error for items-wrapped response: %v", err)
	}
	if acct.NetLiquidation != 1234.56 {
		t.Errorf("net_liquidation from items wrapper: want 1234.56, got %f", acct.NetLiquidation)
	}
	if acct.Cash != 500.00 {
		t.Errorf("cash from items wrapper: want 500.00, got %f", acct.Cash)
	}
}

func TestGetAccount_empty(t *testing.T) {
	// Empty items array — account has no data.
	m := newMock(false).on("assets", []interface{}{}, nil)
	acct, err := GetAccount(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if acct.NetLiquidation != 0 || acct.Cash != 0 {
		t.Error("expected zero-value account for empty response")
	}
}

func TestGetAccount_nullData(t *testing.T) {
	// Tiger returns "" for empty responses; client.Call normalises to JSON null.
	m := newMock(false)
	m.responses["assets"] = append(m.responses["assets"], json.RawMessage("null"))
	m.errs["assets"] = append(m.errs["assets"], nil)

	acct, err := GetAccount(m)
	if err != nil {
		t.Fatalf("expected empty result for null data, got error: %v", err)
	}
	if acct.NetLiquidation != 0 {
		t.Errorf("expected zero account for null data, got %f", acct.NetLiquidation)
	}
}

func TestGetAccount_apiError(t *testing.T) {
	m := newMock(false).onErr("assets", "connection timeout")
	_, err := GetAccount(m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestParseAccount_directArray(t *testing.T) {
	data, _ := json.Marshal([]map[string]interface{}{realTigerAccount})
	items, err := parseAccount(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].NetLiquidation != 1234.56 {
		t.Errorf("netLiquidation: want 1234.56, got %f", items[0].NetLiquidation)
	}
	if items[0].CashValue != 500.00 {
		t.Errorf("cashValue: want 500.00, got %f", items[0].CashValue)
	}
}

func TestParseAccount_itemsWrapper(t *testing.T) {
	wrapped, _ := json.Marshal(map[string]interface{}{"items": []map[string]interface{}{realTigerAccount}})
	items, err := parseAccount(wrapped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
}

func TestParseAccount_invalid(t *testing.T) {
	_, err := parseAccount(json.RawMessage(`"not an object or array"`))
	if err == nil {
		t.Fatal("expected error for string input, got nil")
	}
}
