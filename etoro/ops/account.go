package ops

// GetAccount — returns account balances via the eToro Balances API.
//
// Endpoints:
//   GET /api/v1/balances                         — aggregated across all account types
//   GET /api/v1/balances/{accountType}            — specific account type
//   GET /api/v1/balances/{accountType}/{accountId} — single account
//
// Account types: Trading, Cash, Options, Crypto, MoneyFarm, Spaceship

import (
	"encoding/json"
	"fmt"
)

// Account holds the account-level financial summary.
type Account struct {
	Equity          float64 `json:"equity"`
	Cash            float64 `json:"cash"`
	TotalInvested   float64 `json:"total_invested"`
	TotalPnL        float64 `json:"total_pnl"`
	AvailableBalance float64 `json:"available_balance"`
	Currency        string  `json:"currency"`
}

// AccountHistory holds a single EOD balance snapshot.
type AccountHistory struct {
	Date   string  `json:"date"`
	Equity float64 `json:"equity"`
	Cash   float64 `json:"cash"`
}

// GetAccount returns the aggregated account balance summary.
func GetAccount(c Caller, currency string) (Account, error) {
	query := map[string]string{}
	if currency != "" {
		query["displayCurrency"] = currency
	}

	data, err := c.Get("/api/v1/balances", query)
	if err != nil {
		return Account{}, fmt.Errorf("get_account: %w", err)
	}
	if data == nil || string(data) == "null" {
		return Account{}, nil
	}

	var acct Account
	if err := json.Unmarshal(data, &acct); err != nil {
		return Account{}, fmt.Errorf("get_account: parse: %w", err)
	}
	return acct, nil
}

// GetAccountByType returns the balance for a specific account type.
func GetAccountByType(c Caller, accountType, currency string) (Account, error) {
	query := map[string]string{}
	if currency != "" {
		query["displayCurrency"] = currency
	}

	path := fmt.Sprintf("/api/v1/balances/%s", accountType)
	data, err := c.Get(path, query)
	if err != nil {
		return Account{}, fmt.Errorf("get_account %s: %w", accountType, err)
	}
	if data == nil || string(data) == "null" {
		return Account{}, nil
	}

	var acct Account
	if err := json.Unmarshal(data, &acct); err != nil {
		return Account{}, fmt.Errorf("get_account %s: parse: %w", accountType, err)
	}
	return acct, nil
}

// GetAccountHistory returns EOD balance snapshots.
// fromDate and toDate must be YYYY-MM-DD format. Max range: 365 days.
func GetAccountHistory(c Caller, fromDate, toDate, currency string) ([]AccountHistory, error) {
	query := map[string]string{}
	if fromDate != "" {
		query["fromDate"] = fromDate
	}
	if toDate != "" {
		query["toDate"] = toDate
	}
	if currency != "" {
		query["displayCurrency"] = currency
	}

	data, err := c.Get("/api/v1/balances/history", query)
	if err != nil {
		return nil, fmt.Errorf("get_account_history: %w", err)
	}
	if data == nil || string(data) == "null" {
		return []AccountHistory{}, nil
	}

	var history []AccountHistory
	if err := json.Unmarshal(data, &history); err != nil {
		return nil, fmt.Errorf("get_account_history: parse: %w", err)
	}
	return history, nil
}
