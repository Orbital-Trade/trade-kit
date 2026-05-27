// bounce-bot — RSI oversold bounce strategy (Game 2)
//
// Buys stocks when RSI(14) drops below threshold (default 20) with no
// fundamental damage, and sells when RSI recovers to the mean (default 50).
// Max hold: 5 trading days. Hard stop 5% below entry.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"bounce-bot/internal/broker"
	"bounce-bot/internal/signal"
	"bounce-bot/internal/store"
	"bounce-bot/internal/strategy"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	mode := "paper"
	configPath := "bounce.json"
	signalsPath := "signals.json"
	var remaining []string

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--paper":
			mode = "paper"
		case "--semi":
			mode = "semi"
		case "--live":
			mode = "live"
		case "--config":
			if i+1 < len(os.Args) {
				configPath = os.Args[i+1]
				i++
			}
		case "--signals":
			if i+1 < len(os.Args) {
				signalsPath = os.Args[i+1]
				i++
			}
		default:
			remaining = append(remaining, os.Args[i])
		}
	}

	if len(remaining) == 0 {
		usage()
		os.Exit(1)
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fatalf("config: %v", err)
	}
	bus, err := signal.Open(signalsPath)
	if err != nil {
		fatalf("signals: %v", err)
	}
	db, err := store.New("")
	if err != nil {
		fatalf("store: %v", err)
	}
	b, err := newBroker(mode)
	if err != nil {
		fatalf("broker: %v", err)
	}

	switch remaining[0] {
	case "scan":
		cmdScan(cfg, bus, b)
	case "run":
		cmdRun(cfg, bus, db, b)
	case "monitor":
		cmdMonitor(bus)
	case "status":
		cmdStatus(db)
	case "close":
		if len(remaining) < 2 {
			fatalf("usage: close <SYMBOL>")
		}
		cmdClose(strings.ToUpper(remaining[1]), db, b)
	default:
		usage()
		os.Exit(1)
	}
}

func cmdScan(cfg *strategy.Config, bus *signal.Bus, b broker.Broker) {
	fmt.Printf("═══ BOUNCE SCAN — %s  [%s] ═══\n\n", time.Now().Format("2006-01-02 15:04 MST"), b.Mode())
	added := 0
	for _, sym := range cfg.Watchlist {
		setup, err := strategy.FetchSetup(sym)
		if err != nil {
			fmt.Printf("  %-6s  ERROR: %v\n", sym, err)
			time.Sleep(400 * time.Millisecond)
			continue
		}
		eval := strategy.Evaluate(setup, cfg)
		printEval(eval)
		if eval.Action == "enter" {
			sig := buildSignal(eval, cfg)
			if ok, _ := bus.Add(sig); ok {
				fmt.Printf("         ✓ signal written (id: %s)\n", sig.ID)
				added++
				notify("--symbol", sig.Symbol, "--signal", "BUY",
					"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
					"--stop", fmt.Sprintf("%.2f", sig.Stop),
					"--qty", fmt.Sprintf("%d", sig.Qty),
					"--strategy", "bounce",
					"--note", eval.Reason)
			} else {
				fmt.Printf("         ─ already queued\n")
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	fmt.Printf("\n  %d new signal(s) added\n", added)
}

func cmdRun(cfg *strategy.Config, bus *signal.Bus, db *store.Store, b broker.Broker) {
	fmt.Printf("bounce-bot [%s] running\n", b.Mode())
	scanTick := time.NewTicker(time.Duration(cfg.ScanIntervalSec) * time.Second)
	execTick := time.NewTicker(30 * time.Second)
	if isMarketHours() {
		runScan(cfg, bus)
		runExecutor(bus, db, b, cfg)
	}
	for {
		select {
		case <-scanTick.C:
			if isMarketHours() {
				runScan(cfg, bus)
			}
		case <-execTick.C:
			if isMarketHours() {
				_ = bus.Reload()
				runExecutor(bus, db, b, cfg)
				checkExpired(db, b)
			}
		}
	}
}

func runScan(cfg *strategy.Config, bus *signal.Bus) {
	fmt.Printf("[%s] bounce scan\n", nowET())
	for _, sym := range cfg.Watchlist {
		setup, err := strategy.FetchSetup(sym)
		if err != nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		eval := strategy.Evaluate(setup, cfg)
		if eval.Action == "enter" {
			sig := buildSignal(eval, cfg)
			if ok, _ := bus.Add(sig); ok {
				fmt.Printf("  %-6s  RSI %.1f → signal written\n", sym, setup.RSI)
				notify("--symbol", sig.Symbol, "--signal", "BUY",
					"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
					"--stop", fmt.Sprintf("%.2f", sig.Stop),
					"--qty", fmt.Sprintf("%d", sig.Qty),
					"--strategy", "bounce",
					"--note", fmt.Sprintf("RSI %.1f", setup.RSI))
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
}

func runExecutor(bus *signal.Bus, db *store.Store, b broker.Broker, cfg *strategy.Config) {
	pending := bus.Pending("bounce")
	if len(pending) == 0 {
		return
	}
	fmt.Printf("[%s] %d pending bounce signal(s)\n", nowET(), len(pending))
	for _, sig := range pending {
		if sig.IsExpired() {
			_ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusExpired })
			continue
		}
		setup, err := strategy.FetchSetup(sig.Symbol)
		if err != nil {
			continue
		}
		drift := abs((setup.Price - sig.EntryLimit) / sig.EntryLimit * 100)
		if drift > 3.0 {
			note := fmt.Sprintf("price drifted %.1f%%", drift)
			_ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusRejected; s.Notes = note })
			continue
		}
		if db.FindOpen(sig.Symbol) != nil {
			continue
		}
		entryID, stopID, err := b.Buy(sig.Symbol, sig.Qty, sig.EntryLimit, sig.Stop)
		if err != nil {
			if err.Error() == "declined" {
				_ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusDeclined })
			}
			continue
		}
		_ = bus.Update(sig.ID, func(s *signal.Signal) {
			s.Status = signal.StatusActive
			s.EntryOrderID = entryID
			s.StopOrderID = stopID
		})
		notify("--symbol", sig.Symbol, "--signal", "BUY",
			"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
			"--stop", fmt.Sprintf("%.2f", sig.Stop),
			"--qty", fmt.Sprintf("%d", sig.Qty),
			"--strategy", "bounce",
			"--note", fmt.Sprintf("order placed: %s", entryID))
		expires := time.Now().UTC().AddDate(0, 0, cfg.MaxHoldDays)
		_ = db.Save(store.Trade{
			Symbol: sig.Symbol, EntryPrice: sig.EntryLimit, EntryRSI: setup.RSI,
			StopPrice: sig.Stop, TargetRSI: cfg.RSIExit, Qty: sig.Qty,
			EntryOrderID: entryID, StopOrderID: stopID,
			Status: "open", EntryAt: time.Now().UTC(), ExpiresAt: expires,
		})
	}
}

// checkExpired exits any positions past their max hold date.
func checkExpired(db *store.Store, b broker.Broker) {
	for _, t := range db.All() {
		if t.Status != "open" {
			continue
		}
		if time.Now().UTC().After(t.ExpiresAt) {
			fmt.Printf("[%s] %s: max hold expired — closing\n", nowET(), t.Symbol)
			setup, err := strategy.FetchSetup(t.Symbol)
			if err != nil {
				continue
			}
			if _, err := b.Sell(t.Symbol, t.Qty); err == nil {
				_ = db.Close(t.Symbol, "expired", setup.Price)
			}
		}
	}
}

func cmdMonitor(bus *signal.Bus) {
	_ = bus.Reload()
	sigs := bus.All()
	fmt.Printf("═══ BOUNCE SIGNAL BUS (%d) ═══\n\n", len(sigs))
	for _, s := range sigs {
		fmt.Printf("  %-6s  %-8s  RSI entry %s  qty %d @ $%.2f  stop $%.2f\n",
			s.Symbol, s.Status, s.Reason, s.Qty, s.EntryLimit, s.Stop)
	}
}

func cmdStatus(db *store.Store) {
	trades := db.All()
	fmt.Printf("═══ BOUNCE TRADE STORE ═══\n\n")
	var total float64
	for _, t := range trades {
		if t.Status == "open" {
			fmt.Printf("  %-6s  OPEN  %d sh @ $%.2f  RSI %.1f → %.0f  expires %s\n",
				t.Symbol, t.Qty, t.EntryPrice, t.EntryRSI, t.TargetRSI, t.ExpiresAt.Format("Jan 2"))
		} else {
			fmt.Printf("  %-6s  %-7s  P&L $%+.2f\n", t.Symbol, strings.ToUpper(t.Status), t.PnL)
			total += t.PnL
		}
	}
	fmt.Printf("\n  Closed P&L: $%+.2f\n", total)
}

func cmdClose(symbol string, db *store.Store, b broker.Broker) {
	open := db.FindOpen(symbol)
	if open == nil {
		fatalf("no open trade for %s", symbol)
	}
	setup, _ := strategy.FetchSetup(symbol)
	if _, err := b.Sell(symbol, open.Qty); err != nil {
		fatalf("sell: %v", err)
	}
	_ = db.Close(symbol, "exited", setup.Price)
	fmt.Printf("Closed %s %d sh @ ~$%.2f\n", symbol, open.Qty, setup.Price)
}

func buildSignal(eval strategy.Signal, cfg *strategy.Config) signal.Signal {
	return signal.Signal{
		ID:         signal.GenerateID("bounce", eval.Symbol, time.Now().UTC()),
		Symbol:     eval.Symbol,
		Strategy:   "bounce",
		Action:     "enter",
		EntryLimit: eval.LimitPrice,
		Stop:       eval.StopPrice,
		Qty:        eval.Qty,
		Reason:     eval.Reason,
		ExpiresAt:  time.Now().UTC().AddDate(0, 0, cfg.MaxHoldDays),
		Status:     signal.StatusPending,
	}
}

func printEval(sig strategy.Signal) {
	switch sig.Action {
	case "enter":
		fmt.Printf("  %-6s  ▶ ENTER  RSI %.1f  %d sh @ $%.2f  stop $%.2f  — %s\n",
			sig.Symbol, sig.Setup.RSI, sig.Qty, sig.LimitPrice, sig.StopPrice, sig.Reason)
	case "skip":
		fmt.Printf("  %-6s  — skip   RSI %.1f  — %s\n", sig.Symbol, sig.Setup.RSI, sig.Reason)
	}
}

func newBroker(mode string) (broker.Broker, error) {
	switch mode {
	case "paper":
		return broker.NewPaper(), nil
	case "semi":
		live, err := broker.NewLive()
		if err != nil {
			return nil, err
		}
		return broker.NewSemi(live), nil
	case "live":
		return broker.NewLive()
	}
	return nil, fmt.Errorf("unknown mode %q", mode)
}

func loadConfig(path string) (*strategy.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg strategy.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.ScanIntervalSec == 0 {
		cfg.ScanIntervalSec = 300
	}
	return &cfg, nil
}

func isMarketHours() bool {
	et := time.Now().UTC().Add(-4 * time.Hour)
	if w := et.Weekday(); w == time.Saturday || w == time.Sunday {
		return false
	}
	h, m, _ := et.Clock()
	mins := h*60 + m
	return mins >= 9*60+30 && mins < 16*60
}

func nowET() string { return time.Now().UTC().Add(-4 * time.Hour).Format("15:04:05 ET") }

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

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "bounce-bot: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`bounce-bot — RSI oversold bounce strategy

Usage:
  bounce-bot [--paper|--semi|--live] [--config path] [--signals path] <command>

Commands:
  scan              Scan watchlist for RSI oversold signals
  run               Continuous scan + execute loop
  monitor           Show signal bus state
  status            Show trade store
  close <SYMBOL>    Manual exit

`)
}
