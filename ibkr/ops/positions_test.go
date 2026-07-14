package ops

import "testing"

func TestGetPositions(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST123"
	m.setResponse("/v1/api/portfolio/TEST123/positions/0", []map[string]interface{}{
		{
			"conid": 265598, "contractDesc": "AAPL",
			"position": 10.0, "avgCost": 150.00,
			"mktPrice": 155.00, "mktValue": 1550.00,
			"unrealizedPnl": 50.00, "realizedPnl": 0.0,
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
	if positions[0].Side != "long" {
		t.Errorf("Side = %q, want long", positions[0].Side)
	}
	if positions[0].ConID != 265598 {
		t.Errorf("ConID = %d, want 265598", positions[0].ConID)
	}
}

func TestGetPositionsEmpty(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST123"
	m.setResponse("/v1/api/portfolio/TEST123/positions/0", []map[string]interface{}{})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("len = %d, want 0", len(positions))
	}
}

func TestGetPositionsShort(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST123"
	m.setResponse("/v1/api/portfolio/TEST123/positions/0", []map[string]interface{}{
		{
			"conid": 272093, "contractDesc": "MSFT",
			"position": -5.0, "avgCost": 400.00,
			"mktPrice": 395.00, "mktValue": -1975.00,
			"unrealizedPnl": 25.00, "realizedPnl": 0.0,
		},
	})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if positions[0].Side != "short" {
		t.Errorf("Side = %q, want short", positions[0].Side)
	}
}
