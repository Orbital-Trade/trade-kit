package strategy

import "fmt"

// Evaluate returns the bot's action for this setup given the config.
// Logic: enter 1-N days before earnings, exit day-of (before the report).
func Evaluate(s Setup, cfg *Config) Signal {
	if s.EarningsDate.IsZero() {
		return Signal{Symbol: s.Symbol, Action: "skip", Reason: "no earnings date available", Setup: s}
	}
	if s.DaysToEarnings < 0 {
		return Signal{Symbol: s.Symbol, Action: "skip", Reason: "earnings already passed", Setup: s}
	}
	// Day of earnings → exit before the report drops
	if s.DaysToEarnings == 0 {
		return Signal{Symbol: s.Symbol, Action: "exit", Reason: "earnings today — sell before close", Setup: s}
	}
	// Too far out — watch only
	if s.DaysToEarnings > cfg.DaysBefore {
		return Signal{
			Symbol: s.Symbol,
			Action: "watch",
			Reason: fmt.Sprintf("%d days to earnings on %s (enter at ≤%d)",
				s.DaysToEarnings, s.EarningsDate.Format("Jan 2"), cfg.DaysBefore),
			Setup: s,
		}
	}

	// In entry window — apply quality filters
	if s.Price < cfg.MinPrice {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("price $%.2f below minimum $%.2f", s.Price, cfg.MinPrice), Setup: s}
	}
	if s.AvgVolume > 0 && s.AvgVolume < cfg.MinADV {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("avg volume %.0f below minimum %.0f", s.AvgVolume, cfg.MinADV), Setup: s}
	}
	if s.Run5d > cfg.MaxRunPct {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("already up %.1f%% in 5 days — don't chase (max %.1f%%)", s.Run5d, cfg.MaxRunPct),
			Setup: s}
	}

	qty := int(cfg.Budget / s.Price)
	if qty < 1 {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("budget $%.0f too small for 1 share at $%.2f", cfg.Budget, s.Price), Setup: s}
	}
	stopPrice := s.Price * (1 - cfg.StopPct/100)

	return Signal{
		Symbol:     s.Symbol,
		Action:     "enter",
		Reason:     fmt.Sprintf("%d days to earnings on %s", s.DaysToEarnings, s.EarningsDate.Format("Jan 2 2006")),
		Qty:        qty,
		LimitPrice: s.Price,
		StopPrice:  stopPrice,
		Setup:      s,
	}
}
