package recipe

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	daystrat "daytrader-bot/strategy"
	bouncestrat "bounce-bot/strategy"
	earnstrat "earnings-bot/strategy"
	idxstrat "github.com/jpramirez/trade-kit/index/strategy"
)

// StrategyFunc is the common signature for all recipe scan loops.
type StrategyFunc func(ctx context.Context, configJSON []byte, onSignal func(RecipeSignal), onExecute func(RecipeSignal) error) error

// DaytraderStrategy wraps the daytrader gap-up scanner.
func DaytraderStrategy(ctx context.Context, configJSON []byte, onSignal func(RecipeSignal), onExecute func(RecipeSignal) error) error {
	var cfg daystrat.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return fmt.Errorf("daytrader config: %w", err)
	}
	if cfg.ScanIntervalSec <= 0 {
		cfg.ScanIntervalSec = 60
	}

	ticker := time.NewTicker(time.Duration(cfg.ScanIntervalSec) * time.Second)
	defer ticker.Stop()

	// Run one scan immediately, then on ticker.
	for {
		for _, symbol := range cfg.Watchlist {
			setup, err := daystrat.FetchSetup(symbol)
			if err != nil {
				log.Printf("[daytrader] %s: fetch error: %v", symbol, err)
				continue
			}
			sig := daystrat.Evaluate(setup, &cfg)
			rs := RecipeSignal{
				RecipeID:    "daytrader",
				Symbol:      sig.Symbol,
				Action:      sig.Action,
				Reason:      sig.Reason,
				Qty:         sig.Qty,
				LimitPrice:  sig.LimitPrice,
				StopPrice:   sig.StopPrice,
				TargetPrice: sig.TargetPrice,
			}
			onSignal(rs)
			if sig.Action == "enter" || sig.Action == "enter_short" {
				if err := onExecute(rs); err != nil {
					log.Printf("[daytrader] %s: execute error: %v", symbol, err)
				}
			}
			time.Sleep(400 * time.Millisecond) // Yahoo rate limit
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// BounceStrategy wraps the RSI oversold bounce scanner.
func BounceStrategy(ctx context.Context, configJSON []byte, onSignal func(RecipeSignal), onExecute func(RecipeSignal) error) error {
	var cfg bouncestrat.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return fmt.Errorf("bounce config: %w", err)
	}
	if cfg.ScanIntervalSec <= 0 {
		cfg.ScanIntervalSec = 300
	}

	ticker := time.NewTicker(time.Duration(cfg.ScanIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		for _, symbol := range cfg.Watchlist {
			setup, err := bouncestrat.FetchSetup(symbol)
			if err != nil {
				log.Printf("[bounce] %s: fetch error: %v", symbol, err)
				continue
			}
			sig := bouncestrat.Evaluate(setup, &cfg)
			rs := RecipeSignal{
				RecipeID:   "bounce",
				Symbol:     sig.Symbol,
				Action:     sig.Action,
				Reason:     sig.Reason,
				Qty:        sig.Qty,
				LimitPrice: sig.LimitPrice,
				StopPrice:  sig.StopPrice,
			}
			onSignal(rs)
			if sig.Action == "enter" {
				if err := onExecute(rs); err != nil {
					log.Printf("[bounce] %s: execute error: %v", symbol, err)
				}
			}
			time.Sleep(400 * time.Millisecond)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// EarningsStrategy wraps the earnings play scanner.
func EarningsStrategy(ctx context.Context, configJSON []byte, onSignal func(RecipeSignal), onExecute func(RecipeSignal) error) error {
	var cfg earnstrat.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return fmt.Errorf("earnings config: %w", err)
	}
	if cfg.ScanIntervalSec <= 0 {
		cfg.ScanIntervalSec = 300
	}

	ticker := time.NewTicker(time.Duration(cfg.ScanIntervalSec) * time.Second)
	defer ticker.Stop()

	for {
		for _, symbol := range cfg.Watchlist {
			var earningsDate time.Time
			if dateStr, ok := cfg.EarningsDates[symbol]; ok {
				earningsDate, _ = time.Parse("2006-01-02", dateStr)
			}
			setup, err := earnstrat.FetchSetup(symbol, earningsDate)
			if err != nil {
				log.Printf("[earnings] %s: fetch error: %v", symbol, err)
				continue
			}
			sig := earnstrat.Evaluate(setup, &cfg)
			rs := RecipeSignal{
				RecipeID:   "earnings",
				Symbol:     sig.Symbol,
				Action:     sig.Action,
				Reason:     sig.Reason,
				Qty:        sig.Qty,
				LimitPrice: sig.LimitPrice,
				StopPrice:  sig.StopPrice,
			}
			onSignal(rs)
			if sig.Action == "enter" || sig.Action == "enter_short" {
				if err := onExecute(rs); err != nil {
					log.Printf("[earnings] %s: execute error: %v", symbol, err)
				}
			}
			time.Sleep(400 * time.Millisecond)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

// IndexStrategy wraps the QQQ/VIX momentum scanner.
func IndexStrategy(ctx context.Context, configJSON []byte, onSignal func(RecipeSignal), onExecute func(RecipeSignal) error) error {
	var cfg idxstrat.Config
	if err := json.Unmarshal(configJSON, &cfg); err != nil {
		return fmt.Errorf("index config: %w", err)
	}
	if cfg.ScanIntervalSec <= 0 {
		cfg.ScanIntervalSec = 30
	}

	ticker := time.NewTicker(time.Duration(cfg.ScanIntervalSec) * time.Second)
	defer ticker.Stop()

	var hasPosition bool
	var posSymbol string

	for {
		qqq, err := idxstrat.FetchQuote("QQQ")
		if err != nil {
			log.Printf("[index] QQQ fetch error: %v", err)
			goto wait
		}
		{
			vix, err := idxstrat.FetchQuote("^VIX")
			if err != nil {
				log.Printf("[index] VIX fetch error: %v", err)
				goto wait
			}

			if hasPosition {
				// Check exit on current position.
				pos := idxstrat.Position{Symbol: posSymbol, EntryPrice: qqq.Price}
				shouldExit, reason := idxstrat.CheckExit(cfg, pos, qqq.Price)
				if shouldExit {
					rs := RecipeSignal{
						RecipeID: "index",
						Symbol:   posSymbol,
						Action:   "exit",
						Reason:   reason,
					}
					onSignal(rs)
					if err := onExecute(rs); err != nil {
						log.Printf("[index] exit error: %v", err)
					}
					hasPosition = false
					posSymbol = ""
				}
			} else {
				// Check entry.
				sig := idxstrat.Evaluate(cfg, qqq, vix)
				if sig != nil {
					action := "enter"
					rs := RecipeSignal{
						RecipeID: "index",
						Symbol:   sig.Symbol,
						Action:   action,
						Reason:   sig.Reason,
						Qty:      sig.Shares,
					}
					onSignal(rs)
					if err := onExecute(rs); err != nil {
						log.Printf("[index] entry error: %v", err)
					} else {
						hasPosition = true
						posSymbol = sig.Symbol
					}
				}
			}
		}
	wait:
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
