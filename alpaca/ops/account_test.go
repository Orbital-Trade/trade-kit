package ops

import "testing"

func TestGetAccount(t *testing.T) {
	m := newMock(false)
	m.setResponse("/v2/account", map[string]interface{}{
		"id": "test-123", "status": "ACTIVE",
		"equity": "10500.50", "cash": "3200.00",
		"buying_power": "6400.00", "long_market_value": "7300.50",
		"short_market_value": "0", "initial_margin": "3650.25",
		"maintenance_margin": "2190.15", "daytrade_count": 2,
		"pattern_day_trader": false,
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
	if acct.Status != "ACTIVE" {
		t.Errorf("Status = %q, want ACTIVE", acct.Status)
	}
}
