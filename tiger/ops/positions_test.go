package ops

import (
	"encoding/json"
	"errors"
	"testing"
)

// realTigerPositions mirrors what the actual Tiger API returns after
// client.Call unwraps the outer JSON string. Verified from live debug output.
var realTigerPositions = []map[string]interface{}{
	{
		"symbol":        "EXTR",
		"position":      9,
		"positionQty":   9.0,
		"averageCost":   21.9775,
		"latestPrice":   23.2605,
		"marketValue":   209.3445,
		"unrealizedPnl": 11.55,
		"realizedPnl":   0.0,
	},
	{
		"symbol":        "NOK",
		"position":      10,
		"positionQty":   10.0,
		"averageCost":   11.2395,
		"latestPrice":   13.13,
		"marketValue":   131.3,
		"unrealizedPnl": 18.91,
		"realizedPnl":   10.38,
	},
	{
		"symbol":        "QUBT",
		"position":      15,
		"positionQty":   15.0,
		"averageCost":   9.5954,
		"latestPrice":   10.4,
		"marketValue":   156.0,
		"unrealizedPnl": 12.07,
		"realizedPnl":   0.0,
	},
}

func TestGetPositions_directArray(t *testing.T) {
	// Direct array response (no wrapper).
	m := newMock(false).on("positions", realTigerPositions, nil)

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(positions) != 3 {
		t.Fatalf("expected 3 positions, got %d", len(positions))
	}

	// Verify field mapping against real API names (camelCase → our snake_case).
	extr := positions[0]
	if extr.Symbol != "EXTR" {
		t.Errorf("symbol: want EXTR, got %s", extr.Symbol)
	}
	if extr.Shares != 9 {
		t.Errorf("shares (from 'position'): want 9, got %d", extr.Shares)
	}
	if extr.AvgCost != 21.9775 {
		t.Errorf("avg_cost (from 'averageCost'): want 21.9775, got %f", extr.AvgCost)
	}
	if extr.MarketPrice != 23.2605 {
		t.Errorf("market_price (from 'latestPrice'): want 23.2605, got %f", extr.MarketPrice)
	}
	if extr.MarketValue != 209.3445 {
		t.Errorf("market_value: want 209.3445, got %f", extr.MarketValue)
	}
	if extr.UnrealizedPnL != 11.55 {
		t.Errorf("unrealized_pnl: want 11.55, got %f", extr.UnrealizedPnL)
	}

	nok := positions[1]
	if nok.RealizedPnL != 10.38 {
		t.Errorf("realized_pnl: want 10.38, got %f", nok.RealizedPnL)
	}
}

func TestGetPositions_itemsWrapper(t *testing.T) {
	// Tiger wraps positions in {"items":[...]} after double-decode.
	wrapped := map[string]interface{}{
		"items": realTigerPositions,
		"summary": map[string]interface{}{
			"netLiquidation": 1000.0,
		},
	}
	m := newMock(false).on("positions", wrapped, nil)

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("unexpected error for items-wrapped response: %v", err)
	}
	if len(positions) != 3 {
		t.Fatalf("expected 3 positions from items wrapper, got %d", len(positions))
	}
}

func TestGetPositions_positionQtyFallback(t *testing.T) {
	// When 'position' is 0 but 'positionQty' has the value, use positionQty.
	m := newMock(false).on("positions", []map[string]interface{}{
		{
			"symbol":      "NOK",
			"position":    0,
			"positionQty": 10.0,
			"averageCost": 11.24,
			"latestPrice": 13.13,
		},
	}, nil)

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("expected 1 position, got %d", len(positions))
	}
	if positions[0].Shares != 10 {
		t.Errorf("shares from positionQty fallback: want 10, got %d", positions[0].Shares)
	}
}

func TestGetPositions_empty(t *testing.T) {
	m := newMock(false).on("positions", []interface{}{}, nil)
	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("expected empty slice, got %d", len(positions))
	}
}

func TestGetPositions_nullData(t *testing.T) {
	// Tiger returns "" for empty lists; client.Call normalises to JSON null.
	m := newMock(false)
	m.responses["positions"] = append(m.responses["positions"], json.RawMessage("null"))
	m.errs["positions"] = append(m.errs["positions"], nil)

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("expected empty result for null data, got error: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("expected 0 positions, got %d", len(positions))
	}
}

func TestGetPositions_apiError(t *testing.T) {
	m := newMock(false).onErr("positions", "API error 4001: unauthorized")
	_, err := GetPositions(m)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetPositions_parseError(t *testing.T) {
	m := newMock(false)
	m.responses["positions"] = append(m.responses["positions"], []byte(`{not valid json`))
	m.errs["positions"] = append(m.errs["positions"], nil)

	_, err := GetPositions(m)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestParsePositions_directArray(t *testing.T) {
	data, _ := json.Marshal(realTigerPositions)
	items, err := parsePositions(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].Symbol != "EXTR" {
		t.Errorf("symbol: want EXTR, got %s", items[0].Symbol)
	}
}

func TestParsePositions_itemsWrapper(t *testing.T) {
	wrapped, _ := json.Marshal(map[string]interface{}{"items": realTigerPositions})
	items, err := parsePositions(wrapped)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestParsePositions_invalid(t *testing.T) {
	_, err := parsePositions(json.RawMessage(`"not an object or array"`))
	if err == nil {
		t.Fatal("expected error for string input, got nil")
	}
}

// Suppress "declared and not used" — errors import is used in the API error test.
var _ = errors.New
