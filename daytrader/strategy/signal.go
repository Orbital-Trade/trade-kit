package strategy

import (
	"fmt"
	"time"
)

// Evaluate returns the bot's action for a given setup and config.
// In earnings mode: checks RVOL, allows short on gap-down, tighter stops.
func Evaluate(s Setup, cfg *Config) Signal {
	absGap := s.GapPct
	if absGap < 0 {
		absGap = -absGap
	}

	// In earnings mode, we trade both gap-up (long) and gap-down (short).
	// In normal mode, only gap-up (positive gap).
	isShort := cfg.EarningsMode && cfg.AllowShort && s.GapPct < 0
	isLong := s.GapPct > 0

	if !isLong && !isShort {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("gap %.1f%% — flat, no direction", s.GapPct), Setup: s}
	}

	if absGap < cfg.GapMinPct {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("gap %.1f%% below minimum %.1f%%", s.GapPct, cfg.GapMinPct), Setup: s}
	}
	if absGap > cfg.GapMaxPct {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("gap %.1f%% too extended (max %.1f%%) — don't chase", s.GapPct, cfg.GapMaxPct), Setup: s}
	}
	if s.Price < cfg.MinPrice {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("price $%.2f below minimum", s.Price), Setup: s}
	}
	if s.AvgVolume > 0 && s.AvgVolume < cfg.MinADV {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("avg volume %.0f below minimum", s.AvgVolume), Setup: s}
	}

	// Earnings mode: require minimum relative volume (institutional participation).
	if cfg.EarningsMode && cfg.RVOLMin > 0 && s.RVOL > 0 && s.RVOL < cfg.RVOLMin {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("RVOL %.1fx below minimum %.1fx — weak volume", s.RVOL, cfg.RVOLMin), Setup: s}
	}

	// Check entry window
	et := time.Now().UTC().Add(-4 * time.Hour)
	h, m, _ := et.Clock()
	currentMin := h*60 + m
	if currentMin < cfg.EntryWindowStartMin || currentMin >= cfg.EntryWindowEndMin {
		rvolStr := ""
		if s.RVOL > 0 {
			rvolStr = fmt.Sprintf(" RVOL %.1fx", s.RVOL)
		}
		return Signal{
			Symbol: s.Symbol,
			Action: "watch",
			Reason: fmt.Sprintf("gap %+.1f%%%s — waiting for entry window (%02d:%02d–%02d:%02d ET)",
				s.GapPct, rvolStr,
				cfg.EntryWindowStartMin/60, cfg.EntryWindowStartMin%60,
				cfg.EntryWindowEndMin/60, cfg.EntryWindowEndMin%60),
			Setup: s,
		}
	}

	qty := int(cfg.Budget / s.Price)
	if qty < 1 {
		return Signal{Symbol: s.Symbol, Action: "skip",
			Reason: fmt.Sprintf("budget $%.0f too small", cfg.Budget), Setup: s}
	}

	var action, reason string
	var limitPrice, stopPrice, targetPrice float64

	if isLong {
		stopPrice = s.Price * (1 - cfg.StopPct/100)
		risk := s.Price - stopPrice
		targetPrice = s.Price + risk*cfg.RRMin
		gapHigh := s.PrevClose * (1 + absGap/100)
		if targetPrice > gapHigh {
			targetPrice = gapHigh
		}
		limitPrice = s.Price
		action = "enter"
		reason = fmt.Sprintf("gap +%.1f%% earnings reaction — long entry", s.GapPct)
		if s.RVOL > 0 {
			reason += fmt.Sprintf(" RVOL %.1fx", s.RVOL)
		}
	} else {
		// Short: sell into gap-down, stop above current price, target below
		stopPrice = s.Price * (1 + cfg.StopPct/100) // stop ABOVE for short
		risk := stopPrice - s.Price
		targetPrice = s.Price - risk*cfg.RRMin // profit target below
		gapLow := s.PrevClose * (1 - absGap/100)
		if targetPrice < gapLow {
			targetPrice = gapLow
		}
		limitPrice = s.Price
		action = "enter_short"
		reason = fmt.Sprintf("gap %.1f%% earnings miss — short entry", s.GapPct)
		if s.RVOL > 0 {
			reason += fmt.Sprintf(" RVOL %.1fx", s.RVOL)
		}
	}

	return Signal{
		Symbol:      s.Symbol,
		Action:      action,
		Reason:      reason,
		Qty:         qty,
		LimitPrice:  limitPrice,
		StopPrice:   stopPrice,
		TargetPrice: targetPrice,
		Setup:       s,
	}
}
