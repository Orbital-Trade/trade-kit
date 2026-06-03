// earnings-bot — earnings momentum strategy
//
// Buys stocks N days before their earnings report and sells day-of (before the
// report drops). Captures the pre-earnings run-up without binary event exposure.
//
// Signal flow:
//
//	scan  → writes pending signals to signals.json
//	run   → monitors signals.json, executes fresh pending signals
//
// Any external source (trade-kit scanner, manual edit, other bots) can write
// signals to signals.json and this bot will pick them up automatically.
//
// Usage:
//
//	earnings-bot [--paper|--semi|--live] [--config path] [--signals path] <command>
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"earnings-bot/internal/broker"
	"earnings-bot/internal/signal"
	"earnings-bot/internal/store"
	"earnings-bot/internal/strategy"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	mode := "paper"
	configPath := "earnings.json"
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
		fatalf("config %q: %v", configPath, err)
	}
	bus, err := signal.Open(signalsPath)
	if err != nil {
		fatalf("signals %q: %v", signalsPath, err)
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
		cmdClose(strings.ToUpper(remaining[1]), cfg, db, b)
	default:
		usage()
		os.Exit(1)
	}
}

// cmdScan evaluates the watchlist and writes pending signals to the signal bus.
// It does NOT execute — it only produces signals. Run picks them up.
func cmdScan(cfg *strategy.Config, bus *signal.Bus, b broker.Broker) {
	fmt.Printf("═══ EARNINGS SCAN — %s  [%s] ═══\n\n",
		time.Now().Format("2006-01-02 15:04 MST"), b.Mode())
	fmt.Printf("  Writing signals → %s\n\n", bus)

	added, skipped := 0, 0
	for _, sym := range cfg.Watchlist {
		setup, err := strategy.FetchSetup(sym, earningsOverride(cfg, sym))
		if err != nil {
			fmt.Printf("  %-6s  ERROR: %v\n", sym, err)
			time.Sleep(400 * time.Millisecond)
			continue
		}
		eval := strategy.Evaluate(setup, cfg)
		printEval(eval)

		if eval.Action == "enter" || eval.Action == "exit" {
			sig := buildSignal(eval, cfg)
			ok, err := bus.Add(sig)
			if err != nil {
				fmt.Printf("         ✗ write error: %v\n", err)
			} else if ok {
				fmt.Printf("         ✓ signal written (id: %s)\n", sig.ID)
				added++
			} else {
				fmt.Printf("         ─ already in queue\n")
				skipped++
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
	fmt.Printf("\n  %d new signal(s) added, %d already queued\n", added, skipped)
}

// cmdRun is the main loop: scans every interval, executes fresh pending signals.
func cmdRun(cfg *strategy.Config, bus *signal.Bus, db *store.Store, b broker.Broker) {
	fmt.Printf("earnings-bot [%s] running — signals from %s\n", b.Mode(), bus)
	fmt.Printf("Scan every %ds | Execute loop every 30s | Ctrl+C to stop\n\n", cfg.ScanIntervalSec)

	scanTick := time.NewTicker(time.Duration(cfg.ScanIntervalSec) * time.Second)
	execTick := time.NewTicker(30 * time.Second)

	// Run both immediately on start.
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
				_ = bus.Reload() // pick up signals written by other processes
				runExecutor(bus, db, b, cfg)
			} else {
				fmt.Printf("[%s] market closed\n", nowET())
			}
		}
	}
}

// runScan generates signals from the watchlist and adds them to the bus.
func runScan(cfg *strategy.Config, bus *signal.Bus) {
	fmt.Printf("[%s] scanning %d symbols\n", nowET(), len(cfg.Watchlist))
	for _, sym := range cfg.Watchlist {
		setup, err := strategy.FetchSetup(sym, earningsOverride(cfg, sym))
		if err != nil {
			time.Sleep(400 * time.Millisecond)
			continue
		}
		eval := strategy.Evaluate(setup, cfg)
		if eval.Action == "enter" || eval.Action == "exit" {
			sig := buildSignal(eval, cfg)
			if ok, _ := bus.Add(sig); ok {
				fmt.Printf("  %-6s  → new signal: %s\n", sym, eval.Reason)
				sigDir := "BUY"
				if eval.Action == "exit" { sigDir = "SELL" }
				notify("--symbol", sig.Symbol, "--signal", sigDir,
					"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
					"--stop", fmt.Sprintf("%.2f", sig.Stop),
					"--qty", fmt.Sprintf("%d", sig.Qty),
					"--strategy", "earnings",
					"--note", eval.Reason)
			}
		}
		time.Sleep(400 * time.Millisecond)
	}
}

// runExecutor processes all pending signals in the bus.
func runExecutor(bus *signal.Bus, db *store.Store, b broker.Broker, cfg *strategy.Config) {
	pending := bus.Pending("earnings")
	if len(pending) == 0 {
		return
	}
	fmt.Printf("[%s] checking %d pending signal(s)\n", nowET(), len(pending))

	for _, sig := range pending {
		// Expire stale signals.
		if sig.IsExpired() {
			_ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusExpired })
			fmt.Printf("  %-6s  expired (past %s)\n", sig.Symbol, sig.ExpiresAt.Format("Jan 2 15:04"))
			continue
		}

		// Fetch current price to check signal freshness.
		setup, err := strategy.FetchSetup(sig.Symbol, earningsOverride(cfg, sig.Symbol))
		if err != nil {
			fmt.Printf("  %-6s  fetch error: %v\n", sig.Symbol, err)
			continue
		}

		// Reject if price has moved >3% from the signal price.
		drift := abs((setup.Price - sig.EntryLimit) / sig.EntryLimit * 100)
		if drift > 3.0 {
			note := fmt.Sprintf("price $%.2f drifted %.1f%% from signal $%.2f", setup.Price, drift, sig.EntryLimit)
			_ = bus.Update(sig.ID, func(s *signal.Signal) {
				s.Status = signal.StatusRejected
				s.Notes = note
			})
			fmt.Printf("  %-6s  rejected — %s\n", sig.Symbol, note)
			continue
		}

		// Check if already in position (don't double-enter).
		if db.FindOpen(sig.Symbol) != nil {
			fmt.Printf("  %-6s  already in position — skip\n", sig.Symbol)
			continue
		}

		// Execute.
		fmt.Printf("  %-6s  executing — %s (price $%.2f, drift %.1f%%)\n",
			sig.Symbol, sig.Reason, setup.Price, drift)

		entryID, stopID, err := b.Buy(sig.Symbol, sig.Qty, sig.EntryLimit, sig.Stop)
		if err != nil {
			if err.Error() == "declined" {
				_ = bus.Update(sig.ID, func(s *signal.Signal) { s.Status = signal.StatusDeclined })
				fmt.Printf("  %-6s  declined\n", sig.Symbol)
			} else {
				fmt.Printf("  %-6s  buy error: %v\n", sig.Symbol, err)
			}
			continue
		}

		// Mark signal active and save trade.
		_ = bus.Update(sig.ID, func(s *signal.Signal) {
			s.Status = signal.StatusActive
			s.EntryOrderID = entryID
			s.StopOrderID = stopID
		})
		notify("--symbol", sig.Symbol, "--signal", "BUY",
			"--price", fmt.Sprintf("%.2f", sig.EntryLimit),
			"--stop", fmt.Sprintf("%.2f", sig.Stop),
			"--qty", fmt.Sprintf("%d", sig.Qty),
			"--strategy", "earnings",
			"--note", fmt.Sprintf("order placed: %s", entryID))
		t := store.Trade{
			Symbol:       sig.Symbol,
			EntryPrice:   sig.EntryLimit,
			StopPrice:    sig.Stop,
			Qty:          sig.Qty,
			EntryOrderID: entryID,
			StopOrderID:  stopID,
			Status:       "open",
			EntryAt:      time.Now().UTC(),
			EarningsDate: sig.ExpiresAt.Format("2006-01-02"),
		}
		if err := db.Save(t); err != nil {
			fmt.Printf("  %-6s  store error: %v\n", sig.Symbol, err)
		}
	}
}

// cmdMonitor prints the current state of the signal bus — readable by orbital-ctrl.
func cmdMonitor(bus *signal.Bus) {
	_ = bus.Reload()
	sigs := bus.All()
	fmt.Printf("═══ SIGNAL BUS (%d signals) ═══\n\n", len(sigs))
	if len(sigs) == 0 {
		fmt.Println("  Empty.")
		return
	}
	for _, s := range sigs {
		age := time.Since(s.CreatedAt).Round(time.Minute)
		switch s.Status {
		case signal.StatusPending:
			expires := time.Until(s.ExpiresAt).Round(time.Minute)
			fmt.Printf("  %-6s  %-8s  %-10s  %d sh @ $%.2f  stop $%.2f  expires in %v  — %s\n",
				s.Symbol, s.Status, s.Strategy, s.Qty, s.EntryLimit, s.Stop, expires, s.Reason)
		case signal.StatusActive:
			fmt.Printf("  %-6s  %-8s  %-10s  entry %s  order %s\n",
				s.Symbol, s.Status, s.Strategy, s.CreatedAt.Format("Jan 2 15:04"), s.EntryOrderID)
		default:
			fmt.Printf("  %-6s  %-8s  %-10s  %v ago  %s\n",
				s.Symbol, s.Status, s.Strategy, age, s.Notes)
		}
	}
}

// cmdStatus prints the trade store (filled/open positions).
func cmdStatus(db *store.Store) {
	trades := db.All()
	fmt.Printf("═══ TRADE STORE (%d trades) ═══\n\n", len(trades))
	if len(trades) == 0 {
		fmt.Println("  No trades recorded.")
		return
	}
	var closedPnL float64
	for _, t := range trades {
		if t.Status == "open" {
			fmt.Printf("  %-6s  OPEN    %d sh @ $%.2f  stop $%.2f  earnings %s\n",
				t.Symbol, t.Qty, t.EntryPrice, t.StopPrice, t.EarningsDate)
		} else {
			fmt.Printf("  %-6s  %-8s  P&L $%+.2f  exit $%.2f  earnings %s\n",
				t.Symbol, strings.ToUpper(t.Status), t.PnL, t.ExitPrice, t.EarningsDate)
			closedPnL += t.PnL
		}
	}
	fmt.Printf("\n  Closed P&L: $%+.2f\n", closedPnL)
}

// cmdClose manually exits an open position at market.
func cmdClose(symbol string, cfg *strategy.Config, db *store.Store, b broker.Broker) {
	open := db.FindOpen(symbol)
	if open == nil {
		fatalf("no open trade for %s", symbol)
	}
	setup, err := strategy.FetchSetup(symbol, earningsOverride(cfg, symbol))
	if err != nil {
		fatalf("fetch price for %s: %v", symbol, err)
	}
	if _, err := b.Sell(symbol, open.Qty); err != nil {
		fatalf("sell: %v", err)
	}
	_ = db.Close(symbol, "exited", setup.Price)
	fmt.Printf("Closed %s  %d sh @ ~$%.2f\n", symbol, open.Qty, setup.Price)
}

// buildSignal converts an evaluated setup into a Signal for the bus.
func buildSignal(eval strategy.Signal, cfg *strategy.Config) signal.Signal {
	// Signal expires at close on earnings day.
	expiresAt := eval.Setup.EarningsDate.Add(20 * time.Hour) // 20:00 UTC = 16:00 ET
	if expiresAt.IsZero() {
		expiresAt = time.Now().UTC().Add(72 * time.Hour)
	}
	return signal.Signal{
		ID:         signal.GenerateID("earnings", eval.Symbol, time.Now().UTC()),
		Symbol:     eval.Symbol,
		Strategy:   "earnings",
		Action:     eval.Action,
		EntryLimit: eval.LimitPrice,
		Stop:       eval.StopPrice,
		Qty:        eval.Qty,
		Reason:     eval.Reason,
		ExpiresAt:  expiresAt,
		Status:     signal.StatusPending,
	}
}

func printEval(sig strategy.Signal) {
	switch sig.Action {
	case "enter":
		fmt.Printf("  %-6s  ▶ ENTER  %d sh @ $%.2f  stop $%.2f  — %s\n",
			sig.Symbol, sig.Qty, sig.LimitPrice, sig.StopPrice, sig.Reason)
	case "exit":
		fmt.Printf("  %-6s  ◀ EXIT   earnings today — sell before close\n", sig.Symbol)
	case "watch":
		fmt.Printf("  %-6s  ◎ WATCH  — %s\n", sig.Symbol, sig.Reason)
	case "skip":
		fmt.Printf("  %-6s  — skip   — %s\n", sig.Symbol, sig.Reason)
	}
}

func earningsOverride(cfg *strategy.Config, symbol string) time.Time {
	if cfg.EarningsDates == nil {
		return time.Time{}
	}
	s, ok := cfg.EarningsDates[symbol]
	if !ok {
		return time.Time{}
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}
	}
	return t.UTC()
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
	default:
		return nil, fmt.Errorf("unknown mode %q", mode)
	}
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
	data, err := os.ReadFile(home + "/.trade-kit/watchlist.json")
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

func isMarketHours() bool {
	et := timeET()
	if w := et.Weekday(); w == time.Saturday || w == time.Sunday {
		return false
	}
	h, m, _ := et.Clock()
	mins := h*60 + m
	return mins >= 9*60+30 && mins < 16*60
}

func timeET() time.Time {
	return time.Now().UTC().Add(-4 * time.Hour)
}

func nowET() string {
	return timeET().Format("15:04:05 ET")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

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

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "earnings-bot: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`earnings-bot — earnings momentum strategy

Usage:
  earnings-bot [--paper|--semi|--live] [--config path] [--signals path] <command>

Commands:
  scan                  Evaluate watchlist → write pending signals to signals.json
  run                   Scan loop + execute fresh pending signals automatically
  monitor               Print current state of the signal bus
  status                Show trade store (filled positions, closed P&L)
  close <SYMBOL>        Manually exit an open position at market

Modes:
  --paper    Log orders only, no real fills (default)
  --semi     Confirm each order before sending
  --live     Execute automatically

Flags:
  --config path    Path to earnings.json  (default: ./earnings.json)
  --signals path   Path to signals.json   (default: ./signals.json)

Signal bus:
  Any process can write to signals.json — the OrbitalTrade scanner, other bots,
  or manual JSON edits. earnings-bot run picks up all pending "earnings" signals.

`)
}
