package ops

import (
	"encoding/json"
	"fmt"
)

// Account holds the account-level financial summary.
type Account struct {
	ID              string  `json:"id"`
	Status          string  `json:"status"`
	Equity          float64 `json:"equity"`
	Cash            float64 `json:"cash"`
	BuyingPower     float64 `json:"buying_power"`
	GrossPosition   float64 `json:"gross_position_value"`
	NetLiquidation  float64 `json:"net_liquidation"`
}

// GetAccount returns the account summary.
func GetAccount(c Caller) (Account, error) {
	accountID := c.AccountID()
	path := fmt.Sprintf("/v1/api/portfolio/%s/summary", accountID)
	data, err := c.Get(path, nil)
	if err != nil {
		return Account{}, fmt.Errorf("get_account: %w", err)
	}

	// IBKR summary returns fields as objects with "amount" and "currency".
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return Account{}, fmt.Errorf("get_account: parse: %w", err)
	}

	acct := Account{ID: accountID, Status: "active"}

	acct.Cash = extractAmount(raw, "totalcashvalue")
	acct.NetLiquidation = extractAmount(raw, "netliquidation")
	acct.Equity = acct.NetLiquidation
	acct.GrossPosition = extractAmount(raw, "grosspositionvalue")
	acct.BuyingPower = extractAmount(raw, "buyingpower")

	return acct, nil
}

func extractAmount(raw map[string]json.RawMessage, key string) float64 {
	v, ok := raw[key]
	if !ok {
		return 0
	}
	// Try object with "amount" field first.
	var obj struct {
		Amount float64 `json:"amount"`
	}
	if json.Unmarshal(v, &obj) == nil && obj.Amount != 0 {
		return obj.Amount
	}
	// Try direct number.
	var f float64
	if json.Unmarshal(v, &f) == nil {
		return f
	}
	return 0
}
