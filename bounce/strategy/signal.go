package strategy

import "fmt"

// Evaluate returns the bot's action for this setup.
func Evaluate(s Setup, cfg *Config) Signal {
	if s.RSI > cfg.RSIEntry {
		return Signal{
			Symbol: s.Symbol,
			Action: "skip",
			Reason: fmt.Sprintf("RSI %.1f not oversold (threshold ≤%.0f)", s.RSI, cfg.RSIEntry),
			Setup:  s,
		}
	}
	if s.Price < cfg.MinPrice {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("price $%.2f below minimum $%.2f", s.Price, cfg.MinPrice), Setup: s}
	}
	if s.AvgVolume > 0 && s.AvgVolume < cfg.MinADV {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("avg volume %.0f below minimum %.0f", s.AvgVolume, cfg.MinADV), Setup: s}
	}
	qty := int(cfg.Budget / s.Price)
	if qty < 1 {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("budget $%.0f too small for 1 share at $%.2f", cfg.Budget, s.Price), Setup: s}
	}
	stop := s.Price * (1 - cfg.StopPct/100)
	return Signal{
		Symbol:     s.Symbol,
		Action:     "enter",
		Reason:     fmt.Sprintf("RSI %.1f — extreme oversold (threshold ≤%.0f), target RSI %.0f", s.RSI, cfg.RSIEntry, cfg.RSIExit),
		Qty:        qty,
		LimitPrice: s.Price,
		StopPrice:  stop,
		Setup:      s,
	}
}
