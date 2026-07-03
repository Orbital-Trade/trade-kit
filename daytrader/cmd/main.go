// daytrader-bot — gap-up day trade strategy (Game 3)
//
// Scans for stocks gapping up 3-20% at open on real catalyst.
// Enters on pullback during 9:35-10:00 AM ET window.
// Hard exit by 11:00 AM ET regardless of P&L.
//
// --earnings mode: tighter params, both gap-up (long) and gap-down (short),
// RVOL filter, exit by 10:30 ET. Watchlist loaded from earnings config.
//
// NOTE: Verify Tiger Singapore account is PDT-exempt before using --live.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"daytrader-bot/internal/broker"
	"daytrader-bot/internal/signal"
	"daytrader-bot/internal/store"
	"daytrader-bot/strategy"
)

func main() {
	if len(os.Args) < 2 { usage(); os.Exit(1) }

	mode, configPath, signalsPath := "paper", "daytrader.json", "signals.json"
	earningsMode := false
	var remaining []string
	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--paper":    mode = "paper"
		case "--semi":     mode = "semi"
		case "--live":     mode = "live"
		case "--earnings": earningsMode = true
		case "--config":   if i+1 < len(os.Args) { configPath = os.Args[i+1]; i++ }
		case "--signals":  if i+1 < len(os.Args) { signalsPath = os.Args[i+1]; i++ }
		default: remaining = append(remaining, os.Args[i])
		}
	}
	if len(remaining) == 0 { usage(); os.Exit(1) }

	cfg, err := loadConfig(configPath)
	if err != nil { fatalf("config: %v", err) }

	if earningsMode {
		applyEarningsMode(cfg, configPath)
	}
	bus, err := signal.Open(signalsPath)
	if err != nil { fatalf("signals: %v", err) }
	db, err := store.New("")
	if err != nil { fatalf("store: %v", err) }
	b, err := newBroker(mode)
	if err != nil { fatalf("broker: %v", err) }

	switch remaining[0] {
	case "scan": cmdScan(cfg, bus, b)
	case "run": cmdRun(cfg, bus, db, b)
	case "monitor": cmdMonitor(bus)
	case "status": cmdStatus(db)
	case "close":
		if len(remaining) < 2 { fatalf("usage: close <SYMBOL>") }
		cmdClose(strings.ToUpper(remaining[1]), db, b)
	default: usage(); os.Exit(1)
	}
}

func cmdScan(cfg *strategy.Config, bus *signal.Bus, b broker.Broker) {
	title := "GAP SCAN"
	if cfg.EarningsMode {
		title = "EARNINGS SCALP SCAN"
	}
	fmt.Printf("═══ %s — %s  [%s] ═══\n\n", title, time.Now().Format("2006-01-02 15:04 MST"), b.Mode())
	added := 0
	for _, sym := range cfg.Watchlist {
		setup, err := strategy.FetchSetup(sym)
		if err != nil { fmt.Printf("  %-6s  ERROR: %v\n", sym, err); time.Sleep(400*time.Millisecond); continue }
		eval := strategy.Evaluate(setup, cfg)
		printEval(eval)
		if eval.Action == "enter" || eval.Action == "enter_short" {
			sig := buildSignal(eval)
			if ok, _ := bus.Add(sig); ok {
				fmt.Printf("         ✓ signal written\n")
				added++
				sigDir := "BUY"
				if eval.Action == "enter_short" { sigDir = "SELL" }
				notify("--symbol", sig.Symbol, "--signal", sigDir,
					"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
					"--stop", fmt.Sprintf("%.2f", sig.Stop),
					"--target", fmt.Sprintf("%.2f", sig.Target),
					"--qty", fmt.Sprintf("%d", sig.Qty),
					"--strategy", "daytrader",
					"--note", eval.Reason)
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	fmt.Printf("\n  %d new signal(s)\n", added)
}

func cmdRun(cfg *strategy.Config, bus *signal.Bus, db *store.Store, b broker.Broker) {
	fmt.Printf("daytrader-bot [%s] running — hard exit by %02d:%02d ET\n", b.Mode(), cfg.ExitByMin/60, cfg.ExitByMin%60)
	tick := time.NewTicker(time.Duration(cfg.ScanIntervalSec) * time.Second)
	for {
		<-tick.C
		et := time.Now().UTC().Add(-4 * time.Hour)
		h, m, _ := et.Clock()
		currentMin := h*60 + m
		if !isMarketHours() { continue }

		// Hard time exit: close all open positions by ExitByMin
		if currentMin >= cfg.ExitByMin {
			for _, t := range db.OpenTrades() {
				fmt.Printf("[%s] TIME EXIT — %s\n", nowET(), t.Symbol)
				if setup, err := strategy.FetchSetup(t.Symbol); err == nil {
					if _, err := b.Sell(t.Symbol, t.Qty); err == nil {
						_ = db.Close(t.Symbol, "time-exit", setup.Price)
					}
				}
			}
			continue
		}

		_ = bus.Reload()
		runScan(cfg, bus)
		runExecutor(bus, db, b, cfg)
	}
}

func runScan(cfg *strategy.Config, bus *signal.Bus) {
	for _, sym := range cfg.Watchlist {
		setup, err := strategy.FetchSetup(sym)
		if err != nil { time.Sleep(400*time.Millisecond); continue }
		eval := strategy.Evaluate(setup, cfg)
		if eval.Action == "enter" {
			sig := buildSignal(eval)
			if ok, _ := bus.Add(sig); ok {
				fmt.Printf("  %-6s  gap %.1f%% → signal\n", sym, setup.GapPct)
				notify("--symbol", sig.Symbol, "--signal", "BUY",
					"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
					"--stop", fmt.Sprintf("%.2f", sig.Stop),
					"--target", fmt.Sprintf("%.2f", sig.Target),
					"--qty", fmt.Sprintf("%d", sig.Qty),
					"--strategy", "daytrader",
					"--note", fmt.Sprintf("gap %.1f%%", setup.GapPct))
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func runExecutor(bus *signal.Bus, db *store.Store, b broker.Broker, cfg *strategy.Config) {
	for _, sig := range bus.Pending("daytrader") {
		if sig.IsExpired() { _ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusExpired }); continue }
		setup, err := strategy.FetchSetup(sig.Symbol)
		if err != nil { continue }
		drift := abs((setup.Price - sig.EntryLimit) / sig.EntryLimit * 100)
		if drift > 2.0 {
			_ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusRejected; s.Notes = fmt.Sprintf("drifted %.1f%%", drift) })
			continue
		}
		if db.FindOpen(sig.Symbol) != nil { continue }

		// Compute exit time
		et := time.Now().UTC().Add(-4 * time.Hour)
		exitBy := time.Date(et.Year(), et.Month(), et.Day(), cfg.ExitByMin/60, cfg.ExitByMin%60, 0, 0, et.Location()).UTC().Add(4 * time.Hour)

		entryID, stopID, err := b.Buy(sig.Symbol, sig.Qty, sig.EntryLimit, sig.Stop)
		if err != nil {
			if err.Error() == "declined" { _ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusDeclined }) }
			continue
		}
		_ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusActive; s.EntryOrderID = entryID; s.StopOrderID = stopID })
		notify("--symbol", sig.Symbol, "--signal", "BUY",
			"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
			"--stop", fmt.Sprintf("%.2f", sig.Stop),
			"--target", fmt.Sprintf("%.2f", sig.Target),
			"--qty", fmt.Sprintf("%d", sig.Qty),
			"--strategy", "daytrader",
			"--note", fmt.Sprintf("order placed: %s", entryID))
		_ = db.Save(store.Trade{
			Symbol: sig.Symbol, EntryPrice: sig.EntryLimit, GapPct: setup.GapPct,
			StopPrice: sig.Stop, TargetPrice: sig.Target, Qty: sig.Qty,
			EntryOrderID: entryID, StopOrderID: stopID,
			Status: "open", EntryAt: time.Now().UTC(), ExitBy: exitBy,
		})
	}
}

func cmdMonitor(bus *signal.Bus) {
	_ = bus.Reload()
	fmt.Printf("═══ DAYTRADER SIGNAL BUS ═══\n\n")
	for _, s := range bus.All() {
		if s.Strategy != "daytrader" { continue }
		fmt.Printf("  %-6s  %-8s  gap %s  %d sh @ $%.2f  stop $%.2f  target $%.2f\n",
			s.Symbol, s.Status, s.Reason, s.Qty, s.EntryLimit, s.Stop, s.Target)
	}
}

func cmdStatus(db *store.Store) {
	fmt.Printf("═══ DAYTRADER TRADE STORE ═══\n\n")
	var total float64
	for _, t := range db.All() {
		if t.Status == "open" {
			fmt.Printf("  %-6s  OPEN  %d sh @ $%.2f  stop $%.2f  target $%.2f  exit by %s ET\n",
				t.Symbol, t.Qty, t.EntryPrice, t.StopPrice, t.TargetPrice, t.ExitBy.UTC().Add(-4*time.Hour).Format("15:04"))
		} else {
			fmt.Printf("  %-6s  %-10s  P&L $%+.2f\n", t.Symbol, strings.ToUpper(t.Status), t.PnL)
			total += t.PnL
		}
	}
	fmt.Printf("\n  Closed P&L: $%+.2f\n", total)
}

func cmdClose(symbol string, db *store.Store, b broker.Broker) {
	open := db.FindOpen(symbol)
	if open == nil { fatalf("no open trade for %s", symbol) }
	setup, _ := strategy.FetchSetup(symbol)
	if _, err := b.Sell(symbol, open.Qty); err != nil { fatalf("sell: %v", err) }
	_ = db.Close(symbol, "exited", setup.Price)
	fmt.Printf("Closed %s %d sh @ ~$%.2f\n", symbol, open.Qty, setup.Price)
}

func buildSignal(eval strategy.Signal) signal.Signal {
	return signal.Signal{
		ID: signal.GenerateID("daytrader", eval.Symbol, time.Now().UTC()),
		Symbol: eval.Symbol, Strategy: "daytrader", Action: "enter",
		EntryLimit: eval.LimitPrice, Stop: eval.StopPrice, Target: eval.TargetPrice,
		Qty: eval.Qty, Reason: eval.Reason,
		ExpiresAt: time.Now().UTC().Add(2 * time.Hour), // expires within the trading session
		Status: signal.StatusPending,
	}
}

func printEval(sig strategy.Signal) {
	rvol := ""
	if sig.Setup.RVOL > 0 {
		rvol = fmt.Sprintf("  RVOL %.1fx", sig.Setup.RVOL)
	}
	switch sig.Action {
	case "enter":
		fmt.Printf("  %-6s  ▶ LONG   gap %+.1f%%%s  %d sh @ $%.2f  stop $%.2f  target $%.2f\n",
			sig.Symbol, sig.Setup.GapPct, rvol, sig.Qty, sig.LimitPrice, sig.StopPrice, sig.TargetPrice)
	case "enter_short":
		fmt.Printf("  %-6s  ▼ SHORT  gap %+.1f%%%s  %d sh @ $%.2f  stop $%.2f  target $%.2f\n",
			sig.Symbol, sig.Setup.GapPct, rvol, sig.Qty, sig.LimitPrice, sig.StopPrice, sig.TargetPrice)
	case "watch":
		fmt.Printf("  %-6s  ◎ WATCH  gap %+.1f%%%s  — %s\n", sig.Symbol, sig.Setup.GapPct, rvol, sig.Reason)
	case "skip":
		fmt.Printf("  %-6s  — skip   gap %+.1f%%%s  — %s\n", sig.Symbol, sig.Setup.GapPct, rvol, sig.Reason)
	}
}

func newBroker(mode string) (broker.Broker, error) {
	switch mode {
	case "paper": return broker.NewPaper(), nil
	case "semi": live, err := broker.NewLive(); if err != nil { return nil, err }; return broker.NewSemi(live), nil
	case "live": return broker.NewLive()
	}
	return nil, fmt.Errorf("unknown mode %q", mode)
}

func loadConfig(path string) (*strategy.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil { return nil, err }
	var cfg strategy.Config
	if err := json.Unmarshal(data, &cfg); err != nil { return nil, err }
	if cfg.ScanIntervalSec == 0 { cfg.ScanIntervalSec = 60 }
	if syms := centralWatchlist(); len(syms) > 0 {
		cfg.Watchlist = syms
	}
	return &cfg, nil
}

// centralWatchlist loads symbols from ~/.trade-kit/watchlist.json.
// Returns nil if the file does not exist, leaving the tool-local watchlist intact.
func centralWatchlist() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(home, ".trade-kit", "watchlist.json"))
	if err != nil {
		return nil
	}
	var wl struct {
		Symbols []string `json:"symbols"`
	}
	if err := json.Unmarshal(data, &wl); err != nil {
		return nil
	}
	return wl.Symbols
}

// applyEarningsMode overrides config with tighter earnings scalp parameters
// and replaces the watchlist with today's earnings stocks from earnings.json.
func applyEarningsMode(cfg *strategy.Config, configPath string) {
	cfg.EarningsMode = true
	cfg.AllowShort  = true
	cfg.GapMinPct   = 5.0   // earnings moves are real — require bigger gap
	cfg.StopPct     = 3.0   // tighter stop: confirmed catalyst, no reason to give much room
	cfg.ExitByMin   = 10*60 + 30 // hard exit 10:30 ET (first 60 min of session)
	cfg.EntryWindowStartMin = 9*60 + 30 // open at bell
	cfg.EntryWindowEndMin   = 10*60     // entry window closes 10:00 ET
	cfg.RVOLMin     = 2.0   // require 2x normal volume — institutional participation
	cfg.RRMin       = 2.0   // 2:1 R/R minimum

	// Load today's earnings stocks from earnings.json (sibling directory)
	today := time.Now().Format("2006-01-02")
	earningsPath := filepath.Join(filepath.Dir(configPath), "..", "earnings", "earnings.json")
	if data, err := os.ReadFile(earningsPath); err == nil {
		var ecfg struct {
			Watchlist    []string          `json:"watchlist"`
			EarningsDates map[string]string `json:"earnings_dates"`
		}
		if json.Unmarshal(data, &ecfg) == nil {
			var todayStocks []string
			for sym, date := range ecfg.EarningsDates {
				if date == today {
					todayStocks = append(todayStocks, sym)
				}
			}
			if len(todayStocks) > 0 {
				cfg.Watchlist = todayStocks
				fmt.Printf("  [earnings mode] %d stocks reporting today: %s\n\n",
					len(todayStocks), strings.Join(todayStocks, ", "))
				return
			}
		}
	}
	// Fallback: use full earnings watchlist if no dates match today
	fmt.Printf("  [earnings mode] no earnings dates for today — using full watchlist\n\n")
}

func isMarketHours() bool {
	et := time.Now().UTC().Add(-4 * time.Hour)
	if w := et.Weekday(); w == time.Saturday || w == time.Sunday { return false }
	h, m, _ := et.Clock(); mins := h*60 + m
	return mins >= 9*60+30 && mins < 16*60
}

func nowET() string { return time.Now().UTC().Add(-4*time.Hour).Format("15:04:05 ET") }
func abs(x float64) float64 { if x < 0 { return -x }; return x }
func fatalf(format string, args ...interface{}) { fmt.Fprintf(os.Stderr, "daytrader-bot: "+format+"\n", args...); os.Exit(1) }

// notify shells out to the notifier binary (if present in PATH).
// Runs in a goroutine so it never blocks the trading loop.
// Silent if notifier is not installed — stdout-only fallback.
func notify(args ...string) {
	path, err := exec.LookPath("notifier")
	if err != nil {
		return
	}
	go func() {
		cmd := exec.Command(path, append([]string{"send"}, args...)...)
		_ = cmd.Run()
	}()
}
func usage() {
	fmt.Print(`daytrader-bot — gap-up day trade strategy (Game 3)

IMPORTANT: Verify Tiger Singapore account is PDT-exempt before using --live.

Usage:
  daytrader-bot [--paper|--semi|--live] [--config path] [--signals path] <command>

Commands:
  scan      Scan watchlist for gap-up setups
  run       Continuous scan + execute + time-exit loop
  monitor   Show signal bus
  status    Show trade store
  close <SYMBOL>  Manual exit

`)
}
