package ops

import (
	"encoding/json"
	"fmt"
)

// Account holds the account-level financial summary.
type Account struct {
	ID                string  `json:"id"`
	Status            string  `json:"status"`
	Equity            float64 `json:"equity,string"`
	Cash              float64 `json:"cash,string"`
	BuyingPower       float64 `json:"buying_power,string"`
	LongMarketValue   float64 `json:"long_market_value,string"`
	ShortMarketValue  float64 `json:"short_market_value,string"`
	InitialMargin     float64 `json:"initial_margin,string"`
	MaintenanceMargin float64 `json:"maintenance_margin,string"`
	DaytradeCount     int     `json:"daytrade_count"`
	PatternDayTrader  bool    `json:"pattern_day_trader"`
}

// GetAccount returns the account summary.
func GetAccount(c Caller) (Account, error) {
	data, err := c.Get("/v2/account", nil)
	if err != nil {
		return Account{}, fmt.Errorf("get_account: %w", err)
	}

	var acct Account
	if err := json.Unmarshal(data, &acct); err != nil {
		return Account{}, fmt.Errorf("get_account: parse: %w", err)
	}
	return acct, nil
}
