// Package ops contains isolated IBKR trade operations.
package ops

import "encoding/json"

// Caller is the minimal interface that every ops function requires.
type Caller interface {
	Get(path string, query map[string]string) (json.RawMessage, error)
	Post(path string, body interface{}) (json.RawMessage, error)
	Delete(path string, query map[string]string) (json.RawMessage, error)
	IsPaper() bool
	AccountID() string
	ResolveConID(symbol string) (int, error)
}

// OrderResult is returned by every order-placement operation.
type OrderResult struct {
	OrderID string  `json:"order_id"`
	Mode    string  `json:"mode"`
	Symbol  string  `json:"symbol"`
	Action  string  `json:"action"`
	Type    string  `json:"order_type"`
	Qty     int     `json:"qty"`
	Price   float64 `json:"price,omitempty"`
}
