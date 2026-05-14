// Package broker provides the execution layer for the earnings-bot.
// Three implementations: PaperBroker (log only), SemiBroker (confirm each order),
// LiveBroker (execute via Tiger API).
package broker

// Broker is the minimal execution interface shared by all modes.
type Broker interface {
	// Buy places a limit buy + protective stop. Returns entry and stop order IDs.
	Buy(symbol string, qty int, limitPrice, stopPrice float64) (entryID, stopID string, err error)
	// Sell places a market sell order. Returns the order ID.
	Sell(symbol string, qty int) (string, error)
	// Mode returns "paper", "semi", or "live".
	Mode() string
}
