package strategy

import "fmt"

// Evaluate checks QQQ and VIX and returns a trade signal.
// Returns nil if no signal (flat market or grace period).
func Evaluate(cfg Config, qqq, vix Quote) *Signal {
	// Long: QQQ up enough, VIX calm
	if qqq.ChangePct >= cfg.QQQLongThreshold && vix.Price < cfg.VIXMax {
		return &Signal{
			Direction: Long,
			Symbol:    "TQQQ",
			Shares:    cfg.TQQQShares,
			Price:     qqq.Price,
			Reason: fmt.Sprintf("QQQ %+.2f%% ≥ +%.1f%% threshold | VIX %.2f < %.0f",
				qqq.ChangePct, cfg.QQQLongThreshold, vix.Price, cfg.VIXMax),
		}
	}
	// Short: QQQ down enough, VIX elevated
	if qqq.ChangePct <= cfg.QQQShortThreshold && vix.Price > cfg.VIXSpikeMin {
		return &Signal{
			Direction: Short,
			Symbol:    "SQQQ",
			Shares:    cfg.SQQQShares,
			Price:     qqq.Price,
			Reason: fmt.Sprintf("QQQ %+.2f%% ≤ -%.1f%% threshold | VIX %.2f > %.0f",
				qqq.ChangePct, -cfg.QQQShortThreshold, vix.Price, cfg.VIXSpikeMin),
		}
	}
	return nil
}

// CheckExit returns true + reason if the position should be closed.
func CheckExit(cfg Config, pos Position, currentPrice float64) (bool, string) {
	pnlPct := pos.PnLPct(currentPrice)
	if pnlPct <= -cfg.StopPct {
		return true, fmt.Sprintf("STOP HIT %.2f%% (limit -%.1f%%)", pnlPct, cfg.StopPct)
	}
	if pnlPct >= cfg.TargetPct {
		return true, fmt.Sprintf("TARGET HIT %.2f%% (target +%.1f%%)", pnlPct, cfg.TargetPct)
	}
	return false, ""
}
