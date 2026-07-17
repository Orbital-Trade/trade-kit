// Package scanql implements a SQL-like strategy scripting language for trade-kit.
//
// Users define strategies in .scan files:
//
//	SCAN rsi_bounce
//	  EVERY 300s
//	  SYMBOLS AAPL, NVDA
//	  FETCH quote, rsi(14)
//	  WHERE rsi <= 25 AND volume >= 500000
//	  ENTER LONG
//	    STOP 5%
//	    TARGET 3R
//	    BUDGET 150
//
// The sidecar parses and executes these at runtime — no Go recompile needed.
package scanql

import "time"

// ScanPlan is the parsed representation of a .scan file.
type ScanPlan struct {
	Name     string
	Interval time.Duration
	Symbols  []string
	Fetch    []FetchSpec
	Where    []Condition
	Action   ActionSpec
	Exit     *ExitSpec
}

// FetchSpec describes an indicator to compute.
type FetchSpec struct {
	Name   string    // "quote", "rsi", "macd", "rvol", "ema", "bb", "atr"
	Params []float64 // e.g. rsi(14) → [14], macd(12,26,9) → [12,26,9]
}

// Condition is a single WHERE clause.
type Condition struct {
	Field    string  // "rsi", "gap_pct", "volume", "price", etc.
	Op       string  // ">=", ">", "<=", "<", "==", "!=", "between"
	Value    float64
	ValueEnd float64 // only for BETWEEN
}

// ActionSpec describes what to do when all conditions pass.
type ActionSpec struct {
	Side      string  // "long" or "short"
	Symbol    string  // override symbol (e.g. "TQQQ" instead of scanned symbol)
	Shares    int     // fixed share count (0 = use budget)
	StopPct   float64 // stop-loss as percentage
	TargetPct float64 // take-profit as percentage (0 = use TargetR)
	TargetR   float64 // take-profit as risk multiple
	Budget    float64 // max USD per trade
}

// ExitSpec describes custom exit conditions.
type ExitSpec struct {
	WhenField string        // EXIT WHEN field (e.g. "rsi")
	WhenOp    string        // EXIT WHEN operator
	WhenValue float64       // EXIT WHEN threshold
	MaxHold   int           // HOLD MAX N DAYS
	ExitBy    string        // EXIT BY time (e.g. "12:30")
}
