package ops

import "testing"

func TestGetPositions(t *testing.T) {
	m := newMock(false)
	m.setResponse("/v2/positions", []map[string]interface{}{
		{
			"symbol": "AAPL", "qty": "10", "side": "long",
			"avg_entry_price": "150.00", "current_price": "155.00",
			"market_value": "1550.00", "unrealized_pl": "50.00",
			"unrealized_plpc": "0.0333", "asset_class": "us_equity",
		},
	})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("len = %d, want 1", len(positions))
	}
	if positions[0].Symbol != "AAPL" {
		t.Errorf("Symbol = %q, want AAPL", positions[0].Symbol)
	}
	if positions[0].Qty != 10 {
		t.Errorf("Qty = %v, want 10", positions[0].Qty)
	}
}

func TestGetPositionsEmpty(t *testing.T) {
	m := newMock(false)
	m.setResponse("/v2/positions", []map[string]interface{}{})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("len = %d, want 0", len(positions))
	}
}
