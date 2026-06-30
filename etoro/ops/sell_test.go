package ops

import (
	"testing"
)

func TestClosePositionPaper(t *testing.T) {
	m := newMock(true)

	res, err := ClosePosition(m, "12345")
	if err != nil {
		t.Fatalf("ClosePosition: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", res.Mode)
	}
	if res.Status != "CLOSED" {
		t.Errorf("Status = %q, want CLOSED", res.Status)
	}
}

func TestCancelOrderPaper(t *testing.T) {
	m := newMock(true)

	res, err := CancelOrder(m, "99887766")
	if err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", res.Mode)
	}
	if res.Status != "CANCELLED" {
		t.Errorf("Status = %q, want CANCELLED", res.Status)
	}
}

func TestSellBySymbolPaper(t *testing.T) {
	m := newMock(true)

	results, err := SellBySymbol(m, "AAPL", 200)
	if err != nil {
		t.Fatalf("SellBySymbol: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d, want 1", len(results))
	}
	if results[0].Mode != "PAPER" {
		t.Errorf("Mode = %q, want PAPER", results[0].Mode)
	}
}

func TestModifyPositionPaper(t *testing.T) {
	m := newMock(true)

	err := ModifyPosition(m, "12345", 140.00, 165.00)
	if err != nil {
		t.Fatalf("ModifyPosition: %v", err)
	}
}

func TestModifyOrderPaper(t *testing.T) {
	m := newMock(true)

	err := ModifyOrder(m, "99887766", 140.00, 165.00, 152.00)
	if err != nil {
		t.Fatalf("ModifyOrder: %v", err)
	}
}
