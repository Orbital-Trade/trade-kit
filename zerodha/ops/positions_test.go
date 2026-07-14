package ops

import "testing"

func TestGetPositions(t *testing.T) {
	m := newMock(false)
	m.setResponse("/portfolio/positions", map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"net": []map[string]interface{}{
				{
					"tradingsymbol": "RELIANCE",
					"exchange":      "NSE",
					"quantity":      10,
					"average_price": 2500.00,
					"last_price":    2550.00,
					"pnl":           500.00,
					"unrealised":    500.00,
					"realised":      0.00,
					"buy_quantity":  10,
					"sell_quantity": 0,
					"product":      "CNC",
				},
			},
		},
	})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("len = %d, want 1", len(positions))
	}
	if positions[0].Symbol != "RELIANCE" {
		t.Errorf("Symbol = %q, want RELIANCE", positions[0].Symbol)
	}
	if positions[0].Quantity != 10 {
		t.Errorf("Quantity = %v, want 10", positions[0].Quantity)
	}
	if positions[0].Exchange != "NSE" {
		t.Errorf("Exchange = %q, want NSE", positions[0].Exchange)
	}
}

func TestGetPositionsEmpty(t *testing.T) {
	m := newMock(false)
	m.setResponse("/portfolio/positions", map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"net": []map[string]interface{}{},
		},
	})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("len = %d, want 0", len(positions))
	}
}
