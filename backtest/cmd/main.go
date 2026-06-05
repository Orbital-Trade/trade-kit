// backtest — historical strategy validation
//
// Replays a strategy against OHLCV data and produces a performance report.
// Data sources: Yahoo Finance (default, no key), Alpha Vantage, Polygon.io.
//
// Usage:
//
//	backtest run --strategy daytrader --symbol LUNR --from 2024-01-01 --to 2024-12-31
//	backtest run --strategy bounce    --symbol AAPL --from 2024-01-01
//	backtest run --strategy daytrader --symbol NVDA --from 2024-01-01 --json
package main

import (
	"backtest/internal/data"
	"backtest/internal/engine"
	"backtest/internal/strategy"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config is loaded from backtest.json.
type Config struct {
	DataSource    string                     `json:"data_source"`
	APIKey        string                     `json:"api_key"`
	DefaultBudget float64                    `json:"default_budget"`
	Strategies    map[string]json.RawMessage `json:"strategies"`
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	configPath := "backtest.json"
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--config" {
			configPath = os.Args[i+1]
		}
	}

	cfg, err := loadConfig(configPath)
	if err != nil {
		fatalf("config: %v", err)
	}

	switch os.Args[1] {
	case "run":
		cmdRun(cfg, os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

// ─── RUN ─────────────────────────────────────────────────────────────────────

func cmdRun(cfg Config, args []string) {
	stratName := ""
	symbol := ""
	fromStr := ""
	toStr := time.Now().Format("2006-01-02")
	asJSON := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--strategy":
			if i+1 < len(args) {
				stratName = args[i+1]
				i++
			}
		case "--symbol":
			if i+1 < len(args) {
				symbol = args[i+1]
				i++
			}
		case "--from":
			if i+1 < len(args) {
				fromStr = args[i+1]
				i++
			}
		case "--to":
			if i+1 < len(args) {
				toStr = args[i+1]
				i++
			}
		case "--json":
			asJSON = true
		}
	}

	if stratName == "" {
		fatalf("--strategy required (daytrader|bounce)")
	}
	if symbol == "" {
		fatalf("--symbol required")
	}
	if fromStr == "" {
		fatalf("--from required (YYYY-MM-DD)")
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		fatalf("--from must be YYYY-MM-DD, got %q", fromStr)
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		fatalf("--to must be YYYY-MM-DD, got %q", toStr)
	}

	// Build strategy
	strat, err := buildStrategy(stratName, cfg)
	if err != nil {
		fatalf("%v", err)
	}

	// Fetch data
	fmt.Fprintf(os.Stderr, "fetching %s (%s) %s → %s via %s...\n",
		symbol, stratName, from.Format("2006-01-02"), to.Format("2006-01-02"),
		sourceLabel(cfg.DataSource))
	bars, err := data.Fetch(symbol, cfg.DataSource, cfg.APIKey, from, to)
	if err != nil {
		fatalf("data: %v", err)
	}
	if len(bars) == 0 {
		fatalf("no bars returned for %s in that date range", symbol)
	}
	fmt.Fprintf(os.Stderr, "loaded %d bars\n", len(bars))

	budget := cfg.DefaultBudget
	if budget == 0 {
		budget = 200
	}

	report := engine.Run(symbol, cfg.DataSource, bars, strat, budget)

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(report)
		return
	}
	engine.PrintReport(report)
}

// ─── STRATEGY FACTORY ────────────────────────────────────────────────────────

func buildStrategy(name string, cfg Config) (engine.Strategy, error) {
	switch name {
	case "daytrader":
		dcfg := strategy.DefaultDaytraderConfig()
		if raw, ok := cfg.Strategies["daytrader"]; ok {
			if err := json.Unmarshal(raw, &dcfg); err != nil {
				return nil, fmt.Errorf("daytrader config: %w", err)
			}
		}
		return strategy.NewDaytrader(dcfg), nil

	case "bounce":
		bcfg := strategy.DefaultBounceConfig()
		if raw, ok := cfg.Strategies["bounce"]; ok {
			if err := json.Unmarshal(raw, &bcfg); err != nil {
				return nil, fmt.Errorf("bounce config: %w", err)
			}
		}
		return strategy.NewBounce(bcfg), nil

	default:
		return nil, fmt.Errorf("unknown strategy %q — available: daytrader, bounce", name)
	}
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func loadConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		// Return defaults if no config file found
		return Config{DataSource: "yahoo", DefaultBudget: 200}, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func sourceLabel(s string) string {
	switch s {
	case "", "yahoo":
		return "Yahoo Finance"
	case "alphavantage":
		return "Alpha Vantage"
	case "polygon":
		return "Polygon.io"
	default:
		return s
	}
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "backtest: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`backtest — historical strategy validation

Replays a strategy against OHLCV data and produces a performance report.

COMMANDS
  run --strategy <name> --symbol <SYM> --from <YYYY-MM-DD> [--to <YYYY-MM-DD>]

STRATEGIES
  daytrader   Gap-up momentum (gap%, stop%, R:R)
  bounce      RSI oversold bounce (RSI entry/exit, stop%, max hold days)

FLAGS
  --from <YYYY-MM-DD>   Start date (required)
  --to   <YYYY-MM-DD>   End date (default: today)
  --json                Output report as JSON
  --config <path>       Config file (default: backtest.json)

DATA SOURCES (set in backtest.json)
  yahoo         Yahoo Finance — no API key required (default)
  alphavantage  Alpha Vantage — free tier, requires api_key
  polygon       Polygon.io — free tier, requires api_key

EXAMPLES
  backtest run --strategy daytrader --symbol LUNR --from 2024-01-01 --to 2024-12-31
  backtest run --strategy bounce    --symbol AAPL --from 2024-01-01
  backtest run --strategy daytrader --symbol NVDA --from 2025-01-01 --json

CONFIG (backtest.json)
  {
    "data_source": "yahoo",
    "api_key": "",
    "default_budget": 200.0,
    "strategies": {
      "daytrader": { "gap_min_pct": 3.0, "gap_max_pct": 20.0, "stop_pct": 2.0, "rr_min": 3.0 },
      "bounce":    { "rsi_threshold": 30.0, "rsi_exit": 50.0, "stop_pct": 5.0, "max_hold_days": 5 }
    }
  }

BUILD
  cd backtest && go build -o backtest ./cmd/

`)
}
