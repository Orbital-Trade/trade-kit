package ops

import "testing"

func TestGetAccount(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST123"
	m.setResponse("/v1/api/portfolio/TEST123/summary", map[string]interface{}{
		"totalcashvalue":    map[string]interface{}{"amount": 3200.00, "currency": "USD"},
		"netliquidation":    map[string]interface{}{"amount": 10500.50, "currency": "USD"},
		"grosspositionvalue": map[string]interface{}{"amount": 7300.50, "currency": "USD"},
		"buyingpower":       map[string]interface{}{"amount": 6400.00, "currency": "USD"},
	})

	acct, err := GetAccount(m)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acct.Equity != 10500.50 {
		t.Errorf("Equity = %v, want 10500.50", acct.Equity)
	}
	if acct.Cash != 3200.00 {
		t.Errorf("Cash = %v, want 3200.00", acct.Cash)
	}
	if acct.BuyingPower != 6400.00 {
		t.Errorf("BuyingPower = %v, want 6400.00", acct.BuyingPower)
	}
	if acct.ID != "TEST123" {
		t.Errorf("ID = %q, want TEST123", acct.ID)
	}
}

func TestGetAccountDirectNumbers(t *testing.T) {
	m := newMock(false)
	m.accountID = "TEST456"
	m.setResponse("/v1/api/portfolio/TEST456/summary", map[string]interface{}{
		"totalcashvalue":    5000.00,
		"netliquidation":    12000.00,
		"grosspositionvalue": 7000.00,
		"buyingpower":       10000.00,
	})

	acct, err := GetAccount(m)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acct.Cash != 5000.00 {
		t.Errorf("Cash = %v, want 5000.00", acct.Cash)
	}
	if acct.NetLiquidation != 12000.00 {
		t.Errorf("NetLiquidation = %v, want 12000.00", acct.NetLiquidation)
	}
}
