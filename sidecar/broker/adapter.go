// Package broker defines the unified BrokerAdapter interface and types
// that normalize Tiger, Moomoo, and eToro into a common API surface.
package broker

// BrokerAdapter is the unified interface for all broker integrations.
type BrokerAdapter interface {
	ID() string
	Name() string
	Connect(creds map[string]string) error
	Test() error
	Disconnect() error
	Connected() bool
	Positions() ([]Position, error)
	Account() (AccountInfo, error)
	Orders() ([]OrderInfo, error)
	SetPaper(paper bool)
}

// Position is the unified position type returned by all brokers.
type Position struct {
	Symbol     string  `json:"symbol"`
	Side       string  `json:"side"` // "BUY" or "SELL"
	Units      float64 `json:"units"`
	Amount     float64 `json:"amount"`
	OpenRate   float64 `json:"open_rate"`
	CurrentRate float64 `json:"current_rate"`
	PnL        float64 `json:"pnl"`
	PnLPct     float64 `json:"pnl_pct"`
	StopLoss   float64 `json:"stop_loss"`
	TakeProfit float64 `json:"take_profit"`
}

// AccountInfo is the unified account summary.
type AccountInfo struct {
	Equity        float64 `json:"equity"`
	Cash          float64 `json:"cash"`
	TotalInvested float64 `json:"total_invested"`
	TotalPnL      float64 `json:"total_pnl"`
	Available     float64 `json:"available"`
}

// OrderInfo is the unified pending order type.
type OrderInfo struct {
	OrderID string  `json:"order_id"`
	Symbol  string  `json:"symbol"`
	Side    string  `json:"side"`
	Amount  float64 `json:"amount"`
	Rate    float64 `json:"rate"`
	Status  string  `json:"status"`
}

// BrokerStatus is returned by the registry list endpoint.
type BrokerStatus struct {
	ID        string       `json:"id"`
	Name      string       `json:"name"`
	Connected bool         `json:"connected"`
	Account   *AccountInfo `json:"account_info,omitempty"`
}
