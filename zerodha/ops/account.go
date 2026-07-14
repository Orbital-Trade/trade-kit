package ops

import (
	"encoding/json"
	"fmt"
)

// Account holds the account-level financial summary from Kite margins.
type Account struct {
	Cash       float64 `json:"cash"`
	Collateral float64 `json:"collateral"`
	LiveBal    float64 `json:"live_balance"`
	Debits     float64 `json:"debits"`
	Exposure   float64 `json:"exposure"`
	Net        float64 `json:"net"`
}

// GetAccount returns equity segment margins.
func GetAccount(c Caller) (Account, error) {
	data, err := c.Get("/user/margins/equity", nil)
	if err != nil {
		return Account{}, fmt.Errorf("get_account: %w", err)
	}

	var resp struct {
		Status string `json:"status"`
		Data   struct {
			Available struct {
				Cash       float64 `json:"cash"`
				LiveBal    float64 `json:"live_balance"`
				Collateral float64 `json:"collateral"`
			} `json:"available"`
			Utilised struct {
				Debits   float64 `json:"debits"`
				Exposure float64 `json:"exposure"`
			} `json:"utilised"`
			Net float64 `json:"net"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return Account{}, fmt.Errorf("get_account: parse: %w", err)
	}

	return Account{
		Cash:       resp.Data.Available.Cash,
		Collateral: resp.Data.Available.Collateral,
		LiveBal:    resp.Data.Available.LiveBal,
		Debits:     resp.Data.Utilised.Debits,
		Exposure:   resp.Data.Utilised.Exposure,
		Net:        resp.Data.Net,
	}, nil
}
