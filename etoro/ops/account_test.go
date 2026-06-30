package ops

import (
	"testing"
)

func TestGetAccount(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/balances", map[string]interface{}{
		"equity":            10500.50,
		"cash":              3200.00,
		"total_invested":    7300.50,
		"total_pnl":         450.25,
		"available_balance": 3200.00,
		"currency":          "USD",
	})

	acct, err := GetAccount(m, "USD")
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acct.Equity != 10500.50 {
		t.Errorf("Equity = %v, want 10500.50", acct.Equity)
	}
	if acct.Cash != 3200.00 {
		t.Errorf("Cash = %v, want 3200.00", acct.Cash)
	}
	if acct.TotalInvested != 7300.50 {
		t.Errorf("TotalInvested = %v, want 7300.50", acct.TotalInvested)
	}
}

func TestGetAccountNullResponse(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/balances", nil)

	acct, err := GetAccount(m, "")
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acct.Equity != 0 {
		t.Errorf("Equity = %v, want 0", acct.Equity)
	}
}

func TestGetAccountByType(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/balances/Trading", map[string]interface{}{
		"equity":            8000.00,
		"cash":              2000.00,
		"total_invested":    6000.00,
		"total_pnl":         200.00,
		"available_balance": 2000.00,
		"currency":          "USD",
	})

	acct, err := GetAccountByType(m, "Trading", "USD")
	if err != nil {
		t.Fatalf("GetAccountByType: %v", err)
	}
	if acct.Equity != 8000.00 {
		t.Errorf("Equity = %v, want 8000.00", acct.Equity)
	}
}

func TestGetAccountHistory(t *testing.T) {
	m := newMock(false)
	m.setResponse("/api/v1/balances/history", []map[string]interface{}{
		{"date": "2026-06-01", "equity": 10000.00, "cash": 3000.00},
		{"date": "2026-06-02", "equity": 10200.00, "cash": 3100.00},
	})

	hist, err := GetAccountHistory(m, "2026-06-01", "2026-06-02", "USD")
	if err != nil {
		t.Fatalf("GetAccountHistory: %v", err)
	}
	if len(hist) != 2 {
		t.Fatalf("len(history) = %d, want 2", len(hist))
	}
	if hist[0].Equity != 10000.00 {
		t.Errorf("hist[0].Equity = %v, want 10000.00", hist[0].Equity)
	}
}
