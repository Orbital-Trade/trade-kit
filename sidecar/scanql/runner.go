package scanql

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"trade-kit-sidecar/recipe"
)

// Run executes a ScanPlan as a strategy function compatible with the recipe runner.
func Run(plan *ScanPlan) recipe.StrategyFunc {
	return func(ctx context.Context, _ []byte, onSignal func(recipe.RecipeSignal), onExecute func(recipe.RecipeSignal) error) error {
		ticker := time.NewTicker(plan.Interval)
		defer ticker.Stop()

		for {
			for _, symbol := range plan.Symbols {
				data, err := FetchData(symbol, plan.Fetch)
				if err != nil {
					log.Printf("[scanql:%s] %s: fetch error: %v", plan.Name, symbol, err)
					continue
				}

				pass, reason := Evaluate(plan.Where, data)

				if !pass {
					onSignal(recipe.RecipeSignal{
						RecipeID: plan.Name,
						Symbol:   symbol,
						Action:   "skip",
						Reason:   reason,
					})
					time.Sleep(400 * time.Millisecond) // Yahoo rate limit
					continue
				}

				// Conditions passed — build entry signal.
				price := data["price"]
				if price <= 0 {
					continue
				}

				targetSymbol := symbol
				if plan.Action.Symbol != "" {
					targetSymbol = plan.Action.Symbol
				}

				qty := plan.Action.Shares
				if qty == 0 && plan.Action.Budget > 0 && price > 0 {
					qty = int(math.Floor(plan.Action.Budget / price))
				}
				if qty <= 0 {
					qty = 1
				}

				stopPrice := 0.0
				if plan.Action.StopPct > 0 {
					if plan.Action.Side == "long" {
						stopPrice = price * (1 - plan.Action.StopPct/100)
					} else {
						stopPrice = price * (1 + plan.Action.StopPct/100)
					}
				}

				targetPrice := 0.0
				if plan.Action.TargetPct > 0 {
					if plan.Action.Side == "long" {
						targetPrice = price * (1 + plan.Action.TargetPct/100)
					} else {
						targetPrice = price * (1 - plan.Action.TargetPct/100)
					}
				} else if plan.Action.TargetR > 0 && stopPrice > 0 {
					risk := math.Abs(price - stopPrice)
					if plan.Action.Side == "long" {
						targetPrice = price + risk*plan.Action.TargetR
					} else {
						targetPrice = price - risk*plan.Action.TargetR
					}
				}

				action := "enter"
				if plan.Action.Side == "short" {
					action = "enter_short"
				}

				sig := recipe.RecipeSignal{
					RecipeID:    plan.Name,
					Symbol:      targetSymbol,
					Action:      action,
					Reason:      fmt.Sprintf("scanql conditions met"),
					Qty:         qty,
					LimitPrice:  price,
					StopPrice:   stopPrice,
					TargetPrice: targetPrice,
				}
				onSignal(sig)
				if err := onExecute(sig); err != nil {
					log.Printf("[scanql:%s] %s: execute error: %v", plan.Name, symbol, err)
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
}
