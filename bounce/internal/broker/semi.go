package broker

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// SemiBroker runs the playbook gate check, prints the proposed order,
// and waits for y/N before forwarding to LiveBroker.
type SemiBroker struct {
	live *LiveBroker
}

func NewSemi(live *LiveBroker) *SemiBroker {
	return &SemiBroker{live: live}
}

func (s *SemiBroker) Buy(symbol string, qty int, limit, stop float64) (string, string, error) {
	cost := float64(qty) * limit

	// Gate check before showing the prompt — no point confirming a blocked trade.
	if err := s.live.CheckGate(cost); err != nil {
		fmt.Printf("\n  ⛔ GATE BLOCKED: %v\n", err)
		return "", "", fmt.Errorf("gate: %w", err)
	}

	fmt.Printf("\n┌─ EARNINGS ENTRY ────────────────────────────────────\n")
	fmt.Printf("│  Symbol:   %-6s\n", symbol)
	fmt.Printf("│  Action:   BUY %d shares\n", qty)
	fmt.Printf("│  Limit:    $%.2f\n", limit)
	fmt.Printf("│  Stop:     $%.2f  (%.1f%% risk / $%.2f per share)\n",
		stop, (limit-stop)/limit*100, limit-stop)
	fmt.Printf("│  Cost:     $%.2f\n", cost)
	fmt.Printf("│  Max loss: $%.2f\n", (limit-stop)*float64(qty))
	fmt.Printf("└─────────────────────────────────────────────────────\n")
	fmt.Print("  Execute? [y/N] ")

	r, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.TrimSpace(strings.ToLower(r)) != "y" {
		return "", "", fmt.Errorf("declined")
	}
	return s.live.Buy(symbol, qty, limit, stop)
}

func (s *SemiBroker) Sell(symbol string, qty int) (string, error) {
	fmt.Printf("\n┌─ EARNINGS EXIT ─────────────────────────────────────\n")
	fmt.Printf("│  Symbol:   %-6s\n", symbol)
	fmt.Printf("│  Action:   SELL %d shares (market)\n", qty)
	fmt.Printf("└─────────────────────────────────────────────────────\n")
	fmt.Print("  Execute? [y/N] ")

	r, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.TrimSpace(strings.ToLower(r)) != "y" {
		return "", fmt.Errorf("declined")
	}
	return s.live.Sell(symbol, qty)
}

func (s *SemiBroker) Mode() string { return "semi" }
