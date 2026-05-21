package ops

// Markov — MCP tool: tiger_markov
//
// Implements the Markov regime model for market state classification.
// Labels every trading day in history as BULL / SIDEWAYS / BEAR based on
// the 20-day rolling return, builds a 3×3 transition matrix from all
// historical state changes, then reads tomorrow's state probabilities
// from the row matching today's state.
//
// Signal = P(bull) − P(bear):
//   positive → long bias,  negative → short bias
//   magnitude → conviction / position sizing input
//
// N-day forecasts are computed by raising the matrix to the n-th power
// (matrix multiplication, no new data needed).
//
// Reference: Rowan's "Hedge Fund Method" — observable Markov regime model.

import (
	"fmt"
	"math"
	"time"
)

// State indices — keep in sync with stateNames.
const (
	stateBull = 0
	stateSide = 1
	stateBear = 2
)

var stateNames = [3]string{"BULL", "SIDE", "BEAR"}

// ForecastRow holds an n-day forward probability distribution.
type ForecastRow struct {
	Days int     `json:"days"`
	Bull float64 `json:"bull"`
	Side float64 `json:"side"`
	Bear float64 `json:"bear"`
}

// MarkovResult is returned by Markov.
type MarkovResult struct {
	Symbol    string `json:"symbol"`
	Market    string `json:"market"`
	Currency  string `json:"currency"`
	Timestamp string `json:"timestamp"`

	// Current regime
	Return20D    float64 `json:"return_20d"`    // 20-day return %
	CurrentState string  `json:"current_state"` // "BULL", "SIDE", "BEAR"

	// Historical label distribution
	TotalDays int `json:"total_days"`
	BullDays  int `json:"bull_days"`
	SideDays  int `json:"side_days"`
	BearDays  int `json:"bear_days"`

	// Transition matrix [from][to] — probabilities (0..1)
	Matrix [3][3]float64 `json:"matrix"`
	// Raw transition counts for transparency
	Counts [3][3]int `json:"counts"`

	// Tomorrow's distribution from current state
	TomorrowBull float64 `json:"tomorrow_bull"`
	TomorrowSide float64 `json:"tomorrow_side"`
	TomorrowBear float64 `json:"tomorrow_bear"`

	// Trading signal
	Signal     float64 `json:"signal"`     // P(bull) − P(bear), range −1..+1
	Direction  string  `json:"direction"`  // "LONG", "SHORT", "NEUTRAL"
	Confidence string  `json:"confidence"` // "HIGH", "MEDIUM", "LOW"

	// N-day forward projections (matrix^n)
	Forecasts []ForecastRow `json:"forecasts"`
}

// Markov fetches 2 years of daily bars and computes the regime model.
func Markov(c Caller, symbol string) (MarkovResult, error) {
	info := DetectMarket(symbol)
	res := MarkovResult{
		Symbol:    info.Symbol,
		Market:    info.Market,
		Currency:  info.Currency,
		Timestamp: time.Now().Format("2006-01-02 15:04 MST"),
	}

	bars, err := yahooKline(info.Symbol, "day")
	if err != nil {
		return res, fmt.Errorf("markov: fetch %s: %w", info.Symbol, err)
	}
	if len(bars) < 25 {
		return res, fmt.Errorf("markov: insufficient data (%d bars, need ≥25)", len(bars))
	}

	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}

	// Label each day with a state based on its 20-day lookback return.
	// First labelable day is index 20; states[i] corresponds to closes[i+20].
	window := 20
	states := make([]int, len(closes)-window)
	for i := window; i < len(closes); i++ {
		ret := (closes[i] - closes[i-window]) / closes[i-window] * 100
		states[i-window] = classifyReturn(ret)
	}

	// Count all consecutive-day transitions.
	var counts [3][3]int
	for i := 0; i < len(states)-1; i++ {
		counts[states[i]][states[i+1]]++
	}
	res.Counts = counts

	// Normalize each row to get probabilities.
	// Rows with zero transitions (unseen state) fall back to uniform 1/3.
	var matrix [3][3]float64
	for from := 0; from < 3; from++ {
		total := 0
		for to := 0; to < 3; to++ {
			total += counts[from][to]
		}
		if total > 0 {
			for to := 0; to < 3; to++ {
				matrix[from][to] = float64(counts[from][to]) / float64(total)
			}
		} else {
			matrix[from][0] = 1.0 / 3
			matrix[from][1] = 1.0 / 3
			matrix[from][2] = 1.0 / 3
		}
	}
	res.Matrix = matrix

	// Historical distribution.
	res.TotalDays = len(states)
	for _, s := range states {
		switch s {
		case stateBull:
			res.BullDays++
		case stateSide:
			res.SideDays++
		case stateBear:
			res.BearDays++
		}
	}

	// Current state from the last 20 days.
	last := len(closes) - 1
	ret20 := (closes[last] - closes[last-window]) / closes[last-window] * 100
	res.Return20D = ret20
	cur := classifyReturn(ret20)
	res.CurrentState = stateNames[cur]

	// Tomorrow's distribution = matrix row for current state.
	res.TomorrowBull = matrix[cur][stateBull]
	res.TomorrowSide = matrix[cur][stateSide]
	res.TomorrowBear = matrix[cur][stateBear]

	// Signal: P(bull) − P(bear).
	res.Signal = res.TomorrowBull - res.TomorrowBear
	switch {
	case res.Signal > 0.10:
		res.Direction = "LONG"
	case res.Signal < -0.10:
		res.Direction = "SHORT"
	default:
		res.Direction = "NEUTRAL"
	}
	abs := math.Abs(res.Signal)
	switch {
	case abs >= 0.40:
		res.Confidence = "HIGH"
	case abs >= 0.20:
		res.Confidence = "MEDIUM"
	default:
		res.Confidence = "LOW"
	}

	// N-day forecasts via matrix exponentiation.
	forecastDays := []int{2, 5, 10}
	powered := matIdentity()
	for day := 1; day <= forecastDays[len(forecastDays)-1]; day++ {
		powered = matMul(powered, matrix)
		for _, fd := range forecastDays {
			if day == fd {
				row := powered[cur]
				res.Forecasts = append(res.Forecasts, ForecastRow{
					Days: fd,
					Bull: row[stateBull],
					Side: row[stateSide],
					Bear: row[stateBear],
				})
			}
		}
	}

	return res, nil
}

// classifyReturn maps a 20-day return percentage to a market state.
// Thresholds: Bull ≥ +5%, Bear ≤ −5%, Sideways between.
func classifyReturn(ret float64) int {
	if ret >= 5.0 {
		return stateBull
	}
	if ret <= -5.0 {
		return stateBear
	}
	return stateSide
}

// ── Matrix math ───────────────────────────────────────────────────────────────

func matIdentity() [3][3]float64 {
	return [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
}

func matMul(a, b [3][3]float64) [3][3]float64 {
	var c [3][3]float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			for k := 0; k < 3; k++ {
				c[i][j] += a[i][k] * b[k][j]
			}
		}
	}
	return c
}
