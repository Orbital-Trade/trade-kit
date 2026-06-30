// Package ops contains isolated eToro trade operations.
//
// Architecture:
//
//	Caller interface  ←  *client.EtoroClient (production)
//	                  ←  *mockCaller (tests)
//
// Each file in this package is one logical operation. Functions accept [Caller]
// so they can be unit-tested without a live eToro connection.
package ops

import (
	"encoding/json"
)

// Caller is the minimal interface that every ops function requires.
// *client.EtoroClient satisfies this interface; so does any mock.
type Caller interface {
	// Get makes an authenticated GET request to the eToro API.
	Get(path string, query map[string]string) (json.RawMessage, error)
	// Post makes an authenticated POST request to the eToro API.
	Post(path string, body interface{}) (json.RawMessage, error)
	// Put makes an authenticated PUT request to the eToro API.
	Put(path string, body interface{}) (json.RawMessage, error)
	// Delete makes an authenticated DELETE request to the eToro API.
	Delete(path string, query map[string]string) (json.RawMessage, error)
	// Patch makes an authenticated PATCH request to the eToro API.
	Patch(path string, body interface{}) (json.RawMessage, error)
	// IsPaper returns true when the client is in demo mode.
	IsPaper() bool
}

// OrderResult is returned by every order-placement operation.
type OrderResult struct {
	OrderID    string  `json:"order_id"`
	Mode       string  `json:"mode"`       // "LIVE" or "PAPER"
	Symbol     string  `json:"symbol"`     // ticker symbol
	Action     string  `json:"action"`     // "BUY" or "SELL"
	Type       string  `json:"order_type"` // "MKT", "LMT"
	Qty        int     `json:"qty"`
	Price      float64 `json:"price,omitempty"`
	InstrID    int     `json:"instrument_id,omitempty"`
}
