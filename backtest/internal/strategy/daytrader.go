// Package strategy implements backtest strategies.
package strategy

import (
	"backtest/internal/data"
	"backtest/internal/engine"
	"fmt"
	"math"
)

// DaytraderConfig mirrors the live daytrader-bot config fields used for
// signal generation.
type DaytraderConfig struct {
	GapMinPct float64 `json:"gap_min_pct"` // min gap% to trigger entry
	GapMaxPct float64 `json:"gap_max_pct"` // max gap% (avoid parabolic moves)
	StopPct   float64 `json:"stop_pct"`    // stop-loss % below entry
	RRMin     float64 `json:"rr_min"`      // minimum risk/reward ratio
	Budget    float64 `json:"budget"`      // capital per trade
}

func DefaultDaytraderConfig() DaytraderConfig {
	return DaytraderConfig{
		GapMinPct: 3.0,
		GapMaxPct: 20.0,
		StopPct:   2.0,
		RRMin:     3.0,
		Budget:    200.0,
	}
}

// Daytrader backtests the gap-up momentum strategy.
// Entry: open > prev_close by GapMinPct..GapMaxPct.
// Exit: close hits stop (StopPct below entry) or target (RRMin×stop).
type Daytrader struct {
	cfg DaytraderConfig
}

func NewDaytrader(cfg DaytraderConfig) *Daytrader { return &Daytrader{cfg: cfg} }

func (d *Daytrader) Name() string {
	return fmt.Sprintf("daytrader(gap=%.0f-%.0f%% stop=%.0f%% rr=%.1fx)",
		d.cfg.GapMinPct, d.cfg.GapMaxPct, d.cfg.StopPct, d.cfg.RRMin)
}

func (d *Daytrader) OnBar(bars []data.Bar, idx int, budget float64) engine.Signal {
	if idx == 0 {
		return engine.Signal{}
	}
	bar := bars[idx]
	prev := bars[idx-1]
	if prev.Close == 0 {
		return engine.Signal{}
	}
	gapPct := (bar.Open - prev.Close) / prev.Close * 100
	if gapPct < d.cfg.GapMinPct || gapPct > d.cfg.GapMaxPct {
		return engine.Signal{}
	}
	shares := int(math.Floor(budget / bar.Open))
	if shares <= 0 {
		return engine.Signal{}
	}
	return engine.Signal{
		Enter:  true,
		Shares: shares,
		Reason: fmt.Sprintf("gap+%.1f%%", gapPct),
	}
}

func (d *Daytrader) CheckExit(entry engine.TradeResult, bar data.Bar) engine.ExitDecision {
	stop := entry.EntryPrice * (1 - d.cfg.StopPct/100)
	riskPerShare := entry.EntryPrice - stop
	target := entry.EntryPrice + riskPerShare*d.cfg.RRMin

	if bar.Low <= stop {
		return engine.ExitDecision{Exit: true, Reason: fmt.Sprintf("stop $%.2f", stop)}
	}
	if bar.High >= target {
		return engine.ExitDecision{Exit: true, Reason: fmt.Sprintf("target $%.2f", target)}
	}
	return engine.ExitDecision{}
}
