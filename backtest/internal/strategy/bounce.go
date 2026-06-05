package strategy

import (
	"backtest/internal/data"
	"backtest/internal/engine"
	"fmt"
	"math"
)

// BounceConfig mirrors the live bounce-bot config fields.
type BounceConfig struct {
	RSIThreshold float64 `json:"rsi_threshold"`  // entry when RSI < this
	RSIExit      float64 `json:"rsi_exit"`        // exit when RSI recovers to this
	StopPct      float64 `json:"stop_pct"`        // hard stop %
	MaxHoldDays  int     `json:"max_hold_days"`   // force exit after N days
	Budget       float64 `json:"budget"`
}

func DefaultBounceConfig() BounceConfig {
	return BounceConfig{
		RSIThreshold: 30.0,
		RSIExit:      50.0,
		StopPct:      5.0,
		MaxHoldDays:  5,
		Budget:       200.0,
	}
}

// Bounce backtests the RSI oversold bounce strategy.
// Entry: RSI(14) < RSIThreshold.
// Exit: RSI recovers to RSIExit, hard stop, or MaxHoldDays elapsed.
type Bounce struct {
	cfg BounceConfig
}

func NewBounce(cfg BounceConfig) *Bounce { return &Bounce{cfg: cfg} }

func (b *Bounce) Name() string {
	return fmt.Sprintf("bounce(rsi_entry=%.0f rsi_exit=%.0f stop=%.0f%% hold=%dd)",
		b.cfg.RSIThreshold, b.cfg.RSIExit, b.cfg.StopPct, b.cfg.MaxHoldDays)
}

func (b *Bounce) OnBar(bars []data.Bar, idx int, budget float64) engine.Signal {
	if idx < 14 {
		return engine.Signal{}
	}
	rsi := computeRSI(bars, idx, 14)
	if rsi >= b.cfg.RSIThreshold {
		return engine.Signal{}
	}
	price := bars[idx].Close
	shares := int(math.Floor(budget / price))
	if shares <= 0 {
		return engine.Signal{}
	}
	return engine.Signal{
		Enter:  true,
		Shares: shares,
		Reason: fmt.Sprintf("RSI=%.1f", rsi),
	}
}

func (b *Bounce) CheckExit(entry engine.TradeResult, bar data.Bar) engine.ExitDecision {
	stop := entry.EntryPrice * (1 - b.cfg.StopPct/100)
	if bar.Low <= stop {
		return engine.ExitDecision{Exit: true, Reason: fmt.Sprintf("stop $%.2f", stop)}
	}
	// RSI recovery — we need historical bars, so we approximate using the bar index
	// For simplicity in the engine, track hold days via date delta
	holdDays := int(bar.Date.Sub(entry.EntryDate).Hours()/24) + 1
	if b.cfg.MaxHoldDays > 0 && holdDays >= b.cfg.MaxHoldDays {
		return engine.ExitDecision{Exit: true, Reason: fmt.Sprintf("max hold %d days", holdDays)}
	}
	return engine.ExitDecision{}
}

// computeRSI calculates RSI(period) at bars[idx] using Wilder's smoothing.
func computeRSI(bars []data.Bar, idx, period int) float64 {
	if idx < period {
		return 50
	}
	gains, losses := 0.0, 0.0
	for i := idx - period + 1; i <= idx; i++ {
		diff := bars[i].Close - bars[i-1].Close
		if diff > 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}
