package ops

import "testing"

func TestGetAccount(t *testing.T) {
	m := newMock(false)
	m.setResponse("/user/margins/equity", map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"available": map[string]interface{}{
				"cash":        50000.00,
				"live_balance": 48000.00,
				"collateral":  12000.00,
			},
			"utilised": map[string]interface{}{
				"debits":   2000.00,
				"exposure": 5000.00,
			},
			"net": 60000.00,
		},
	})

	acct, err := GetAccount(m)
	if err != nil {
		t.Fatalf("GetAccount: %v", err)
	}
	if acct.Cash != 50000.00 {
		t.Errorf("Cash = %v, want 50000.00", acct.Cash)
	}
	if acct.Net != 60000.00 {
		t.Errorf("Net = %v, want 60000.00", acct.Net)
	}
	if acct.Collateral != 12000.00 {
		t.Errorf("Collateral = %v, want 12000.00", acct.Collateral)
	}
}
