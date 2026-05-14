// Package ops contains isolated Tiger Brokers trade operations.
//
// Architecture:
//
//	Caller interface  ←  *client.TigerClient (production)
//	                  ←  *mockCaller (tests)
//	                  ←  future MCP server wrapper
//
// Each file in this package is one logical operation and will map 1:1 to an
// MCP tool. Functions accept [Caller] so they can be unit-tested without a
// live Tiger connection, and so any other transport can be substituted.
package ops

import (
	"encoding/json"
)

// Caller is the minimal interface that every ops function requires.
// *client.TigerClient satisfies this interface; so does any mock or MCP adapter.
type Caller interface {
	// Call makes a signed REST request to the Tiger API gateway.
	Call(method string, bizContent interface{}) (json.RawMessage, error)
	// Account returns the Tiger account number used for all order params.
	Account() string
	// IsPaper returns true when the client is in simulation mode.
	// Ops functions return a PAPER result without calling the API.
	IsPaper() bool
	// ResolveFuturesContract maps a root symbol (MES, MNQ, M2K) to the
	// active contract code (e.g. MES2506).
	ResolveFuturesContract(symbol string) (string, error)
}

// OrderResult is returned by every order-placement operation.
type OrderResult struct {
	OrderID string  `json:"order_id"`
	Mode    string  `json:"mode"`       // "LIVE" or "PAPER"
	Symbol  string  `json:"symbol"`
	Action  string  `json:"action"`     // "BUY" or "SELL"
	Type    string  `json:"order_type"` // "MKT", "LMT", "STP"
	Qty     int     `json:"qty"`
	Price   float64 `json:"price,omitempty"` // omitted for market orders
}
