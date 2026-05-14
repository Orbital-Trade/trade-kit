// Package ops contains market utilities for moomoo-cli.
// Trading operations are handled directly by the client package.
package ops

// Quote is a real-time price snapshot (Yahoo Finance).
type Quote struct {
	Symbol    string
	Yahoo     string
	Price     float64
	Open      float64
	High      float64
	Low       float64
	PrevClose float64
	Volume    int64
	ChangePct float64
	Currency  string
}
