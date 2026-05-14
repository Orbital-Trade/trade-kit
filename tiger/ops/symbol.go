package ops

import (
	"regexp"
	"strings"
)

// sgxPattern matches SGX ticker codes: 1–4 letters followed by 1–2 digits,
// optionally ending with a letter. Examples: ES3, G3B, D05, Z74, N2IU.
var sgxPattern = regexp.MustCompile(`^[A-Z]{1,4}[0-9]{1,2}[A-Z]?$`)

// SymbolInfo carries the resolved symbol, market, exchange, and currency for an order or quote.
type SymbolInfo struct {
	Symbol   string // clean symbol (no prefix/suffix)
	Market   string // "US" or "SG"
	Exchange string // "SGX" for Singapore, "" for US (Tiger default)
	Currency string // "USD" or "SGD"
}

// CurrencySign returns the display prefix for the currency (e.g. "S$" or "$").
func (s SymbolInfo) CurrencySign() string {
	if s.Currency == "SGD" {
		return "S$"
	}
	return "$"
}

// DetectMarket classifies a symbol as US or SGX and returns the appropriate
// market/currency pair. Detection rules, in order:
//  1. .SI suffix      → SGX
//  2. SG. prefix      → SGX
//  3. US. prefix      → US
//  4. Pure digits     → SGX (e.g. 558)
//  5. Alpha+digit mix → SGX (e.g. ES3, Z74, G3B, D05, N2IU)
//  6. Default         → US
func DetectMarket(raw string) SymbolInfo {
	sym := strings.ToUpper(strings.TrimSpace(raw))

	if strings.HasSuffix(sym, ".SI") {
		return SymbolInfo{Symbol: strings.TrimSuffix(sym, ".SI"), Market: "SG", Exchange: "SGX", Currency: "SGD"}
	}
	if strings.HasPrefix(sym, "SG.") {
		return SymbolInfo{Symbol: sym[3:], Market: "SG", Exchange: "SGX", Currency: "SGD"}
	}
	if strings.HasPrefix(sym, "US.") {
		return SymbolInfo{Symbol: sym[3:], Market: "US", Exchange: "SGX", Currency: "USD"}
	}
	if isAllDigits(sym) {
		return SymbolInfo{Symbol: sym, Market: "SG", Exchange: "SGX", Currency: "SGD"}
	}
	if sgxPattern.MatchString(sym) {
		return SymbolInfo{Symbol: sym, Market: "SG", Exchange: "SGX", Currency: "SGD"}
	}
	return SymbolInfo{Symbol: sym, Market: "US", Exchange: "SGX", Currency: "USD"}
}

// OrderParams returns the base biz_content fields for a stock place_order call.
// Exchange is omitted for US (Tiger default); required for SGX.
// Note: "market" is NOT a place_order field — only used in position queries.
func (s SymbolInfo) OrderParams(account string) map[string]interface{} {
	p := map[string]interface{}{
		"account":  account,
		"symbol":   s.Symbol,
		"sec_type": "STK",
		"currency": s.Currency,
		"lang":     "en_US",
	}
	if s.Exchange != "" {
		p["exchange"] = s.Exchange
	}
	return p
}

func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
