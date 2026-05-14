package ops

// GetAccount — MCP tool: tiger_account
//
// Returns account-level cash, buying power, and net liquidation value.
// Calls Tiger REST method: assets.
//
// Tiger API quirks handled here:
//   - Response is double-encoded (client.Call unwraps the outer JSON string).
//   - After unwrapping, data is {"items":[{...}]} with flat camelCase fields.
//   - No "summary" sub-object — fields live directly on the account object.

import (
	"encoding/json"
	"fmt"
)

// Account holds the account-level financial summary.
type Account struct {
	NetLiquidation     float64 `json:"net_liquidation"`
	Cash               float64 `json:"cash"`
	BuyingPower        float64 `json:"buying_power"`
	GrossPositionValue float64 `json:"gross_position_value"`
}

// tigerAccount maps the raw Tiger API field names (camelCase) to Go fields.
type tigerAccount struct {
	NetLiquidation     float64 `json:"netLiquidation"`
	CashValue          float64 `json:"cashValue"`
	BuyingPower        float64 `json:"buyingPower"`
	GrossPositionValue float64 `json:"grossPositionValue"`
}

// GetAccount returns the account summary (cash, buying power, net liquidation).
func GetAccount(c Caller) (Account, error) {
	data, err := c.Call("assets", map[string]interface{}{
		"account": c.Account(),
		"lang":    "en_US",
	})
	if err != nil {
		return Account{}, fmt.Errorf("get_account: %w", err)
	}
	if data == nil || string(data) == "null" {
		return Account{}, nil
	}

	raw, err := parseAccount(data)
	if err != nil {
		return Account{}, fmt.Errorf("get_account: parse response: %w", err)
	}
	if len(raw) == 0 {
		return Account{}, nil
	}
	a := raw[0]
	return Account{
		NetLiquidation:     a.NetLiquidation,
		Cash:               a.CashValue,
		BuyingPower:        a.BuyingPower,
		GrossPositionValue: a.GrossPositionValue,
	}, nil
}

// parseAccount handles both response shapes Tiger may return:
//   - Direct array:            [{...}, ...]
//   - Wrapped in items object: {"items":[{...}], ...}
func parseAccount(data json.RawMessage) ([]tigerAccount, error) {
	// Try direct array first.
	var direct []tigerAccount
	if err := json.Unmarshal(data, &direct); err == nil {
		return direct, nil
	}

	// Try items wrapper — Tiger wraps responses in {"items":[...]}.
	var wrapper struct {
		Items []tigerAccount `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("neither array nor {items:[]} format: %w", err)
	}
	return wrapper.Items, nil
}
