package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jpramirez/trade-kit/index/strategy"
)

func main() {
	mode := flag.String("mode", "watch", "watch | semi | live")
	cfgPath := flag.String("config", "", "path to index.json (default: next to binary)")
	flag.Parse()

	// Resolve config path
	if *cfgPath == "" {
		exe, _ := os.Executable()
		// binary lives at tools/index/index-trader; config at tools/index/index.json
		dir := filepath.Dir(exe)
		candidate := filepath.Join(dir, "index.json")
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			// fallback: run from tools/index/ directly
			candidate = filepath.Join(dir, "..", "index.json")
		}
		*cfgPath = candidate
	}
	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		fatalf("config: %v", err)
	}

	fmt.Printf("  index-trader  mode=%s  interval=%ds\n\n", *mode, cfg.ScanIntervalSec)

	var pos *strategy.Position
	entryTime := time.Time{}

	for {
		now := time.Now()
		etLoc, _ := time.LoadLocation("America/New_York")
		if etLoc == nil {
			etLoc = time.FixedZone("EST", -5*60*60)
		}
		et := now.In(etLoc)
		h, m, _ := et.Clock()
		minsIntoSession := h*60 + m - (9*60 + 30)

		qqq, err := strategy.FetchQuote("QQQ")
		if err != nil {
			logf("fetch QQQ: %v", err)
			sleep(cfg.ScanIntervalSec)
			continue
		}
		vix, err := strategy.FetchQuote("%5EVIX")
		if err != nil {
			// VIX sometimes fails — use last known or default
			vix = strategy.Quote{Symbol: "VIX", Price: 18.0}
		}

		// Print pulse
		direction := "→"
		if qqq.ChangePct > 0 {
			direction = "↑"
		} else if qqq.ChangePct < 0 {
			direction = "↓"
		}
		fmt.Printf("[%s ET | %+dm session]  QQQ $%.2f %s%+.2f%%  |  VIX %.2f",
			et.Format("15:04:05"), minsIntoSession, qqq.Price, direction, qqq.ChangePct, vix.Price)

		// Monitor open position
		if pos != nil {
			tqqqQ, qErr := strategy.FetchQuote(pos.Symbol)
			if qErr != nil {
				logf("quote %s: %v — skipping position check", pos.Symbol, qErr)
				fmt.Println()
				sleep(cfg.ScanIntervalSec)
				continue
			}
			pnl := pos.PnL(tqqqQ.Price)
			pnlPct := pos.PnLPct(tqqqQ.Price)
			fmt.Printf("  |  %s %dsh @ $%.2f  P&L: $%.2f (%+.2f%%)",
				pos.Symbol, pos.Shares, tqqqQ.Price, pnl, pnlPct)

			exit, reason := strategy.CheckExit(cfg, *pos, tqqqQ.Price)
			if exit || minsIntoSession >= cfg.ExitByMin {
				if minsIntoSession >= cfg.ExitByMin {
					reason = fmt.Sprintf("EOD exit at %d min", minsIntoSession)
				}
				fmt.Printf("\n  ⚡ EXIT — %s\n", reason)
				if *mode == "live" || (*mode == "semi" && confirm(fmt.Sprintf("Sell %s %d shares?", pos.Symbol, pos.Shares))) {
					executeSell(pos.Symbol, pos.Shares)
				}
				pos = nil
			}
			fmt.Println()
			sleep(cfg.ScanIntervalSec)
			continue
		}

		// Grace period check
		if minsIntoSession < cfg.GracePeriodMin {
			fmt.Printf("  (grace: %dm/%dm)\n", minsIntoSession, cfg.GracePeriodMin)
			sleep(cfg.ScanIntervalSec)
			continue
		}

		// Market closed
		if minsIntoSession < 0 || minsIntoSession > 390 {
			fmt.Println("  (market closed)")
			sleep(cfg.ScanIntervalSec)
			continue
		}

		// Evaluate signal
		sig := strategy.Evaluate(cfg, qqq, vix)
		if sig == nil {
			fmt.Printf("  — flat (no signal)\n")
			sleep(cfg.ScanIntervalSec)
			continue
		}

		// Signal found
		var arrow string
		if sig.Direction == strategy.Long {
			arrow = "🟢 LONG"
		} else {
			arrow = "🔴 SHORT"
		}
		fmt.Printf("\n  ▶ SIGNAL %s %s %dsh  |  %s\n",
			arrow, sig.Symbol, sig.Shares, sig.Reason)
		sigDir := "BUY"
		if sig.Direction != strategy.Long { sigDir = "SELL" }
		notify("--symbol", sig.Symbol, "--signal", sigDir,
			"--qty", fmt.Sprintf("%d", sig.Shares),
			"--strategy", "index",
			"--note", sig.Reason)

		if *mode == "watch" {
			fmt.Println("  (watch mode — no action)")
			sleep(cfg.ScanIntervalSec)
			continue
		}

		execute := *mode == "live" || (*mode == "semi" && confirm(fmt.Sprintf("Buy %s %d shares?", sig.Symbol, sig.Shares)))
		if execute {
			if err := executeBuy(sig.Symbol, sig.Shares); err != nil {
				logf("buy failed: %v", err)
			} else {
				tqqqQ, qErr := strategy.FetchQuote(sig.Symbol)
				if qErr != nil {
					logf("quote after buy %s: %v", sig.Symbol, qErr)
				}
				pos = &strategy.Position{
					Symbol:     sig.Symbol,
					Shares:     sig.Shares,
					EntryPrice: tqqqQ.Price,
					Direction:  sig.Direction,
				}
				entryTime = time.Now()
				fmt.Printf("  ✅ BOUGHT %s %dsh @ $%.2f  (entry: %s)\n",
					pos.Symbol, pos.Shares, pos.EntryPrice, entryTime.Format("15:04:05"))
			}
		}

		sleep(cfg.ScanIntervalSec)
	}
}

func loadConfig(path string) (strategy.Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return strategy.Config{}, err
	}
	defer f.Close()
	var cfg strategy.Config
	return cfg, json.NewDecoder(f).Decode(&cfg)
}

func tigerCLIPath() string {
	_, file, _, _ := runtime.Caller(0)
	// tools/index/cmd/main.go → tools/tiger/tiger-cli
	base := filepath.Dir(filepath.Dir(filepath.Dir(file)))
	return filepath.Join(base, "tiger", "tiger-cli")
}

func executeBuy(symbol string, shares int) error {
	cmd := exec.Command(tigerCLIPath(), "buy", symbol, fmt.Sprintf("%d", shares), "--live")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func executeSell(symbol string, shares int) {
	cmd := exec.Command(tigerCLIPath(), "sell", symbol, fmt.Sprintf("%d", shares), "--live")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		logf("sell failed: %v", err)
	}
}

func confirm(prompt string) bool {
	fmt.Printf("  %s [y/N] (10s timeout): ", prompt)
	ch := make(chan string, 1)
	go func() {
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			ch <- sc.Text()
		}
	}()
	select {
	case input := <-ch:
		return strings.TrimSpace(strings.ToLower(input)) == "y"
	case <-time.After(10 * time.Second):
		fmt.Println("(timeout — skipped)")
		return false
	}
}

func sleep(sec int) {
	time.Sleep(time.Duration(sec) * time.Second)
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

func logf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "  [ERR] "+f+"\n", a...)
}

func fatalf(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "fatal: "+f+"\n", a...)
	os.Exit(1)
}
