package strategy

import "time"

// Config holds all tunable parameters for the earnings momentum strategy.
// Loaded from earnings.json — no recompile needed to change thresholds.
type Config struct {
	DaysBefore      int      `json:"days_before"`      // enter when earnings are this many days away
	StopPct         float64  `json:"stop_pct"`         // hard stop % below entry
	TargetPct       float64  `json:"target_pct"`       // 0 = sell day-of close; >0 = limit sell
	Budget          float64  `json:"budget"`           // max USD per trade
	MinADV          float64  `json:"min_adv"`          // min avg daily volume
	MinPrice        float64  `json:"min_price"`        // min stock price (penny stock filter)
	MaxRunPct       float64  `json:"max_run_pct"`      // skip if already ran > this% in 5 days
	Watchlist       []string          `json:"watchlist"`        // symbols to scan
	ScanIntervalSec int               `json:"scan_interval_sec"` // seconds between run-mode scans
	EarningsDates   map[string]string `json:"earnings_dates"`   // symbol → "2026-05-20" manual override
}

// Setup is the raw market data for one symbol, fetched from Yahoo Finance.
type Setup struct {
	Symbol         string
	Price          float64
	AvgVolume      float64
	Run5d          float64   // % price change over last 5 trading days
	EarningsDate   time.Time // next earnings date; zero if unknown
	DaysToEarnings int       // calendar days until earnings; -1 if past
}

// Signal is the bot's decision for one symbol at a point in time.
type Signal struct {
	Symbol     string
	Action     string // "enter" | "exit" | "watch" | "skip"
	Reason     string
	Qty        int
	LimitPrice float64
	StopPrice  float64
	Setup      Setup
}
