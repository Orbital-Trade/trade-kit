// alert — price threshold alert daemon
//
// Watches a list of symbols and fires a notification when price crosses
// a configured threshold. Integrates with the notifier binary for
// Telegram/Discord delivery.
//
// Usage:
//
//	alert daemon         → poll continuously, fire on threshold cross
//	alert check          → one-shot check, print triggered alerts and exit
//	alert list           → show configured alerts and current prices
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Config is loaded from alert.json.
type Config struct {
	PollIntervalSec int     `json:"poll_interval_sec"`
	Alerts          []Alert `json:"alerts"`
}

// Alert defines a single price threshold to watch.
type Alert struct {
	Symbol string  `json:"symbol"`
	Above  float64 `json:"above,omitempty"` // fire when price >= this
	Below  float64 `json:"below,omitempty"` // fire when price <= this
	Note   string  `json:"note,omitempty"`
	Repeat bool    `json:"repeat,omitempty"` // re-fire every poll while triggered
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	configPath := "alert.json"
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--config" {
			configPath = os.Args[i+1]
		}
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fatalf("config: %v", err)
	}
	if cfg.PollIntervalSec <= 0 {
		cfg.PollIntervalSec = 60
	}

	switch os.Args[1] {
	case "daemon":
		cmdDaemon(cfg)
	case "check":
		cmdCheck(cfg)
	case "list":
		cmdList(cfg)
	default:
		usage()
		os.Exit(1)
	}
}

// ─── DAEMON ──────────────────────────────────────────────────────────────────

func cmdDaemon(cfg Config) {
	// fired tracks which alerts have already triggered (symbol:direction:threshold).
	// Cleared when price moves back across the threshold so it can fire again.
	fired := make(map[string]bool)

	fmt.Printf("[%s] alert daemon started — %d alert(s), polling every %ds\n",
		nowSGT(), len(cfg.Alerts), cfg.PollIntervalSec)

	// Run immediately, then on every tick.
	runChecks(cfg, fired)
	tick := time.NewTicker(time.Duration(cfg.PollIntervalSec) * time.Second)
	for range tick.C {
		runChecks(cfg, fired)
	}
}

// ─── CHECK ───────────────────────────────────────────────────────────────────

func cmdCheck(cfg Config) {
	fired := make(map[string]bool)
	runChecks(cfg, fired)
}

// ─── LIST ────────────────────────────────────────────────────────────────────

func cmdList(cfg Config) {
	fmt.Printf("%-8s  %-10s  %-10s  %-6s  %s\n", "SYMBOL", "ABOVE", "BELOW", "REPEAT", "NOTE")
	fmt.Println(strings.Repeat("─", 55))
	for _, a := range cfg.Alerts {
		above := "─"
		below := "─"
		repeat := "no"
		if a.Above > 0 {
			above = fmt.Sprintf("$%.2f", a.Above)
		}
		if a.Below > 0 {
			below = fmt.Sprintf("$%.2f", a.Below)
		}
		if a.Repeat {
			repeat = "yes"
		}
		fmt.Printf("%-8s  %-10s  %-10s  %-6s  %s\n", a.Symbol, above, below, repeat, a.Note)
	}

	fmt.Println()
	fmt.Println("Live prices:")
	for _, a := range cfg.Alerts {
		price, err := fetchPrice(a.Symbol)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  %-8s  error: %v\n", a.Symbol, err)
			continue
		}
		fmt.Printf("  %-8s  $%.2f\n", a.Symbol, price)
	}
}

// ─── CORE ────────────────────────────────────────────────────────────────────

func runChecks(cfg Config, fired map[string]bool) {
	for _, a := range cfg.Alerts {
		price, err := fetchPrice(a.Symbol)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[%s] %s: fetch error: %v\n", nowSGT(), a.Symbol, err)
			continue
		}

		if a.Above > 0 {
			key := a.Symbol + ":above:" + strconv.FormatFloat(a.Above, 'f', 2, 64)
			if price >= a.Above {
				if !fired[key] || a.Repeat {
					fired[key] = true
					fire(a.Symbol, fmt.Sprintf("crossed ABOVE $%.2f", a.Above), price, a.Note)
				}
			} else {
				delete(fired, key) // price fell back — reset so it can fire again on next cross
			}
		}

		if a.Below > 0 {
			key := a.Symbol + ":below:" + strconv.FormatFloat(a.Below, 'f', 2, 64)
			if price <= a.Below {
				if !fired[key] || a.Repeat {
					fired[key] = true
					fire(a.Symbol, fmt.Sprintf("crossed BELOW $%.2f", a.Below), price, a.Note)
				}
			} else {
				delete(fired, key)
			}
		}
	}
}

func fire(symbol, direction string, price float64, note string) {
	msg := fmt.Sprintf("%s %s — now $%.2f", symbol, direction, price)
	if note != "" {
		msg += " (" + note + ")"
	}
	fmt.Printf("[%s] ALERT: %s\n", nowSGT(), msg)
	notify(msg)
}

// ─── PRICE FETCH ─────────────────────────────────────────────────────────────

func fetchPrice(symbol string) (float64, error) {
	url := "https://query1.finance.yahoo.com/v8/finance/chart/" + symbol + "?interval=1m&range=1d"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("read response: %w", err)
	}

	var result struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct {
				Description string `json:"description"`
			} `json:"error"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("parse: %w", err)
	}
	if result.Chart.Error != nil {
		return 0, fmt.Errorf("yahoo: %s", result.Chart.Error.Description)
	}
	if len(result.Chart.Result) == 0 {
		return 0, fmt.Errorf("no data returned")
	}
	price := result.Chart.Result[0].Meta.RegularMarketPrice
	if price == 0 {
		return 0, fmt.Errorf("zero price returned")
	}
	return price, nil
}

// ─── NOTIFIER ────────────────────────────────────────────────────────────────

func notify(msg string) {
	path, err := exec.LookPath("notifier")
	if err != nil {
		return // notifier not installed — stdout only
	}
	go func() {
		cmd := exec.Command(path, "send", msg)
		_ = cmd.Run()
	}()
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func loadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func nowSGT() string {
	return time.Now().UTC().Add(8 * time.Hour).Format("15:04:05 SGT")
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "alert: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`alert — price threshold alert daemon

Watches symbols and fires a notification when price crosses a threshold.
Integrates with the notifier binary for Telegram/Discord delivery.

COMMANDS
  daemon              Poll continuously, fire on threshold cross (runs forever)
  check               One-shot: check all alerts and exit
  list                Show configured alerts and current live prices

FLAGS
  --config <path>     Config file (default: alert.json)

EXAMPLES
  alert daemon &
  alert daemon --config ~/my-alerts.json &
  alert check
  alert list

CONFIG (alert.json)
  {
    "poll_interval_sec": 60,
    "alerts": [
      { "symbol": "LUNR",  "above": 40.00, "note": "breakout level" },
      { "symbol": "AAPL",  "below": 170.00, "note": "support breach" },
      { "symbol": "QQQ",   "above": 480.00 },
      { "symbol": "SQQQ",  "below": 8.00, "repeat": true }
    ]
  }

  above   Fire when price >= threshold (one-shot by default)
  below   Fire when price <= threshold (one-shot by default)
  repeat  Re-fire every poll cycle while price remains triggered

BUILD
  cd alert && go build -o alert ./cmd/

`)
}
