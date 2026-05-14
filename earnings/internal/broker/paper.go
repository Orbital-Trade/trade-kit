package broker

import "fmt"

// PaperBroker logs all orders without sending them to Tiger.
type PaperBroker struct{}

func NewPaper() *PaperBroker { return &PaperBroker{} }

func (p *PaperBroker) Buy(symbol string, qty int, limit, stop float64) (string, string, error) {
	fmt.Printf("[PAPER] BUY  %-6s  %d shares  limit $%.2f  stop $%.2f  cost $%.2f  max-loss $%.2f\n",
		symbol, qty, limit, stop, limit*float64(qty), (limit-stop)*float64(qty))
	return "PAPER-ENTRY", "PAPER-STOP", nil
}

func (p *PaperBroker) Sell(symbol string, qty int) (string, error) {
	fmt.Printf("[PAPER] SELL %-6s  %d shares  (market)\n", symbol, qty)
	return "PAPER-SELL", nil
}

func (p *PaperBroker) Mode() string { return "paper" }
