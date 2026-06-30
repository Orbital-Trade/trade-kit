package ops

import (
	"testing"
)

func TestGetPositions(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/trading/real/portfolio", []map[string]interface{}{
		{
			"PositionID":     12345,
			"InstrumentID":   1001,
			"IsBuy":          true,
			"Amount":         500.00,
			"Units":          3.5,
			"OpenRate":       142.85,
			"CurrentRate":    148.50,
			"StopLossRate":   138.00,
			"TakeProfitRate": 160.00,
			"NetProfit":      19.73,
		},
	})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 1 {
		t.Fatalf("len(positions) = %d, want 1", len(positions))
	}
	p := positions[0]
	if p.PositionID != "12345" {
		t.Errorf("PositionID = %q, want 12345", p.PositionID)
	}
	if !p.IsBuy {
		t.Error("IsBuy = false, want true")
	}
	if p.Units != 3.5 {
		t.Errorf("Units = %v, want 3.5", p.Units)
	}
	if p.StopLoss != 138.00 {
		t.Errorf("StopLoss = %v, want 138.00", p.StopLoss)
	}
}

func TestGetPositionsPaperMode(t *testing.T) {
	m := newMock(true)
	m.setResponse("/api/v1/trading/demo/portfolio", []map[string]interface{}{})

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("len(positions) = %d, want 0", len(positions))
	}
}

func TestGetPositionsNullResponse(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/trading/real/portfolio", nil)

	positions, err := GetPositions(m)
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(positions) != 0 {
		t.Errorf("len(positions) = %d, want 0", len(positions))
	}
}
