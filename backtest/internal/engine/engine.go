// Package engine runs a strategy against historical bars and collects results.
package engine

import (
	"backtest/internal/data"
	"fmt"
	"math"
	"time"
)

// TradeResult records a single completed trade.
type TradeResult struct {
	Symbol    string
	EntryDate time.Time
	ExitDate  time.Time
	EntryPrice float64
	ExitPrice  float64
	Shares     int
	PnL        float64
	PnLPct     float64
	ExitReason string
}

// Report summarises a full backtest run.
type Report struct {
	Symbol       string
	Strategy     string
	From         time.Time
	To           time.Time
	TotalBars    int
	Trades       []TradeResult
	WinCount     int
	LoseCount    int
	TotalPnL     float64
	MaxDrawdown  float64
	MaxConsecLoss int
}

func (r *Report) WinRate() float64 {
	total := r.WinCount + r.LoseCount
	if total == 0 {
		return 0
	}
	return float64(r.WinCount) / float64(total) * 100
}

func (r *Report) AvgWin() float64 {
	if r.WinCount == 0 {
		return 0
	}
	sum := 0.0
	for _, t := range r.Trades {
		if t.PnL > 0 {
			sum += t.PnL
		}
	}
	return sum / float64(r.WinCount)
}

func (r *Report) AvgLoss() float64 {
	if r.LoseCount == 0 {
		return 0
	}
	sum := 0.0
	for _, t := range r.Trades {
		if t.PnL < 0 {
			sum += t.PnL
		}
	}
	return sum / float64(r.LoseCount)
}

// Signal is returned by a strategy to indicate an entry decision.
type Signal struct {
	Enter  bool
	Shares int    // number of shares to buy
	Reason string
}

// ExitDecision is returned by a strategy's exit check.
type ExitDecision struct {
	Exit   bool
	Reason string
}

// Strategy is the interface a backtest strategy must implement.
type Strategy interface {
	Name() string
	// OnBar is called for each bar. prev is the preceding bar (zero if first).
	// Returns a Signal if an entry should be made (only called when flat).
	OnBar(bars []data.Bar, idx int, budget float64) Signal
	// CheckExit is called each bar while a position is open.
	CheckExit(entry TradeResult, current data.Bar) ExitDecision
}

// Run executes the strategy over bars and returns a Report.
func Run(symbol, source string, bars []data.Bar, strat Strategy, budget float64) Report {
	report := Report{
		Symbol:    symbol,
		Strategy:  strat.Name(),
		TotalBars: len(bars),
	}
	if len(bars) > 0 {
		report.From = bars[0].Date
		report.To = bars[len(bars)-1].Date
	}

	var open *TradeResult
	equity := budget
	peak := budget
	consecLoss := 0

	for i, bar := range bars {
		if open != nil {
			// Check exit on each bar while in a position
			dec := strat.CheckExit(*open, bar)
			if dec.Exit {
				open.ExitDate = bar.Date
				open.ExitPrice = bar.Close
				open.ExitReason = dec.Reason
				open.PnL = float64(open.Shares) * (open.ExitPrice - open.EntryPrice)
				open.PnLPct = (open.ExitPrice - open.EntryPrice) / open.EntryPrice * 100

				report.Trades = append(report.Trades, *open)
				if open.PnL > 0 {
					report.WinCount++
					consecLoss = 0
				} else {
					report.LoseCount++
					consecLoss++
					if consecLoss > report.MaxConsecLoss {
						report.MaxConsecLoss = consecLoss
					}
				}
				report.TotalPnL += open.PnL
				equity += open.PnL
				if equity > peak {
					peak = equity
				}
				dd := (peak - equity) / peak * 100
				if dd > report.MaxDrawdown {
					report.MaxDrawdown = dd
				}
				open = nil
			}
			continue
		}

		// Flat — check for entry
		sig := strat.OnBar(bars, i, equity)
		if sig.Enter && sig.Shares > 0 {
			open = &TradeResult{
				Symbol:     symbol,
				EntryDate:  bar.Date,
				EntryPrice: bar.Close,
				Shares:     sig.Shares,
			}
		}
	}
	// Force-close any open position at end of data
	if open != nil {
		last := bars[len(bars)-1]
		open.ExitDate = last.Date
		open.ExitPrice = last.Close
		open.ExitReason = "end of data"
		open.PnL = float64(open.Shares) * (open.ExitPrice - open.EntryPrice)
		open.PnLPct = (open.ExitPrice - open.EntryPrice) / open.EntryPrice * 100
		report.Trades = append(report.Trades, *open)
		if open.PnL > 0 {
			report.WinCount++
		} else {
			report.LoseCount++
		}
		report.TotalPnL += open.PnL
	}

	_ = math.Pi // keep import
	return report
}

// PrintReport writes a human-readable report to stdout.
func PrintReport(r Report) {
	fmt.Printf("═══ BACKTEST REPORT ═══\n\n")
	fmt.Printf("  Symbol:    %s\n", r.Symbol)
	fmt.Printf("  Strategy:  %s\n", r.Strategy)
	fmt.Printf("  Period:    %s → %s\n", r.From.Format("2006-01-02"), r.To.Format("2006-01-02"))
	fmt.Printf("  Bars:      %d trading days\n\n", r.TotalBars)

	if len(r.Trades) == 0 {
		fmt.Printf("  No trades triggered.\n\n")
		return
	}

	fmt.Printf("  Trades:        %d\n", len(r.Trades))
	fmt.Printf("  Win rate:      %.1f%%  (%d W / %d L)\n", r.WinRate(), r.WinCount, r.LoseCount)
	fmt.Printf("  Total P&L:     $%.2f\n", r.TotalPnL)
	if r.WinCount > 0 {
		fmt.Printf("  Avg win:       +$%.2f\n", r.AvgWin())
	}
	if r.LoseCount > 0 {
		fmt.Printf("  Avg loss:      -$%.2f\n", math.Abs(r.AvgLoss()))
	}
	fmt.Printf("  Max drawdown:  %.1f%%\n", r.MaxDrawdown)
	fmt.Printf("  Max consec L:  %d\n\n", r.MaxConsecLoss)

	fmt.Printf("  %-12s  %-12s  %-8s  %8s  %8s  %7s  %s\n",
		"ENTRY DATE", "EXIT DATE", "SHARES", "ENTRY", "EXIT", "P&L", "REASON")
	fmt.Printf("  %s\n", fmt.Sprintf("%s", make([]byte, 80)))
	for _, t := range r.Trades {
		sign := "+"
		if t.PnL < 0 {
			sign = ""
		}
		fmt.Printf("  %-12s  %-12s  %8d  %8.2f  %8.2f  %s%6.2f  %s\n",
			t.EntryDate.Format("2006-01-02"),
			t.ExitDate.Format("2006-01-02"),
			t.Shares,
			t.EntryPrice,
			t.ExitPrice,
			sign, t.PnL,
			t.ExitReason,
		)
	}
	fmt.Println()
}
