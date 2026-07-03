package recipe

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"trade-kit-sidecar/broker"
)

// recipeDefinition is a registered recipe with its config path and strategy function.
type recipeDefinition struct {
	ID         string
	Name       string
	ConfigPath string // relative to trade-kit root, e.g. "daytrader/daytrader.json"
	Strategy   StrategyFunc
}

// runningRecipe tracks a currently executing recipe goroutine.
type runningRecipe struct {
	startedAt  time.Time
	cancel     context.CancelFunc
	bus        *MemBus
	scanCount  int
	lastScanAt *time.Time
	err        string
}

// Runner manages recipe lifecycle: start, stop, list.
type Runner struct {
	mu          sync.RWMutex
	definitions map[string]recipeDefinition
	running     map[string]*runningRecipe
	registry    *broker.Registry
	broadcaster EventBroadcaster
	baseDir     string // trade-kit root directory for config resolution
}

// NewRunner creates a runner with all four recipes registered.
func NewRunner(registry *broker.Registry, broadcaster EventBroadcaster) *Runner {
	// Find trade-kit root (sidecar binary is in sidecar/, go up one level).
	baseDir, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(baseDir, "daytrader", "daytrader.json")); err != nil {
		// Try parent directory.
		parent := filepath.Dir(baseDir)
		if _, err := os.Stat(filepath.Join(parent, "daytrader", "daytrader.json")); err == nil {
			baseDir = parent
		}
	}

	r := &Runner{
		definitions: make(map[string]recipeDefinition),
		running:     make(map[string]*runningRecipe),
		registry:    registry,
		broadcaster: broadcaster,
		baseDir:     baseDir,
	}

	r.definitions["daytrader"] = recipeDefinition{
		ID: "daytrader", Name: "Gap-Up Day Trader",
		ConfigPath: "daytrader/daytrader.json", Strategy: DaytraderStrategy,
	}
	r.definitions["bounce"] = recipeDefinition{
		ID: "bounce", Name: "RSI Bounce Scanner",
		ConfigPath: "bounce/bounce.json", Strategy: BounceStrategy,
	}
	r.definitions["earnings"] = recipeDefinition{
		ID: "earnings", Name: "Earnings Play Scanner",
		ConfigPath: "earnings/earnings.json", Strategy: EarningsStrategy,
	}
	r.definitions["index"] = recipeDefinition{
		ID: "index", Name: "QQQ/VIX Index Trader",
		ConfigPath: "index/index.json", Strategy: IndexStrategy,
	}

	return r
}

// List returns the state of all registered recipes.
func (r *Runner) List() []RecipeState {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]RecipeState, 0, len(r.definitions))
	for _, def := range r.definitions {
		state := RecipeState{
			ID:     def.ID,
			Name:   def.Name,
			Status: "stopped",
		}
		if run, ok := r.running[def.ID]; ok {
			state.Status = "running"
			state.StartedAt = &run.startedAt
			state.ScanCount = run.scanCount
			state.LastScanAt = run.lastScanAt
			state.SignalCount = len(run.bus.All())
			if run.err != "" {
				state.Status = "error"
				state.Error = run.err
			}
		}
		out = append(out, state)
	}
	return out
}

// RunningCount returns the number of currently running recipes.
func (r *Runner) RunningCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, run := range r.running {
		if run.err == "" {
			count++
		}
	}
	return count
}

// Signals returns the signals for a specific recipe.
func (r *Runner) Signals(id string) ([]RecipeSignal, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	run, ok := r.running[id]
	if !ok {
		return []RecipeSignal{}, nil
	}
	return run.bus.All(), nil
}

// Start launches a recipe as a background goroutine.
func (r *Runner) Start(id string, configOverride json.RawMessage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	def, ok := r.definitions[id]
	if !ok {
		return fmt.Errorf("unknown recipe: %s", id)
	}
	if _, running := r.running[id]; running {
		return fmt.Errorf("recipe %s is already running", id)
	}

	// Load config from disk.
	configPath := filepath.Join(r.baseDir, def.ConfigPath)
	configJSON, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("load config %s: %w", configPath, err)
	}

	// Merge with override if provided.
	if len(configOverride) > 0 {
		var base map[string]interface{}
		json.Unmarshal(configJSON, &base)
		var override map[string]interface{}
		if json.Unmarshal(configOverride, &override) == nil {
			for k, v := range override {
				base[k] = v
			}
			configJSON, _ = json.Marshal(base)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	bus := NewMemBus(func(sig RecipeSignal) {
		r.broadcaster.Broadcast("recipe_signal", map[string]interface{}{
			"recipe_id": id,
			"signal":    sig,
		})
	})

	run := &runningRecipe{
		startedAt: time.Now(),
		cancel:    cancel,
		bus:       bus,
	}
	r.running[id] = run

	// Broadcast started event.
	r.broadcaster.Broadcast("recipe_state", map[string]interface{}{
		"recipe_id": id,
		"status":    "running",
	})

	// Find a connected broker for order execution.
	findBroker := func() BrokerExecutor {
		for _, b := range r.registry.List() {
			if b.Connected {
				adapter, _ := r.registry.Get(b.ID)
				if adapter != nil {
					return adapter
				}
			}
		}
		return nil
	}

	// Spawn strategy goroutine.
	go func() {
		defer func() {
			if p := recover(); p != nil {
				r.mu.Lock()
				if run, ok := r.running[id]; ok {
					run.err = fmt.Sprintf("panic: %v", p)
				}
				r.mu.Unlock()
				r.broadcaster.Broadcast("recipe_state", map[string]interface{}{
					"recipe_id": id,
					"status":    "error",
					"error":     fmt.Sprintf("panic: %v", p),
				})
			}
		}()

		onSignal := func(sig RecipeSignal) {
			bus.Add(sig)
			r.mu.Lock()
			if run, ok := r.running[id]; ok {
				run.scanCount++
				now := time.Now()
				run.lastScanAt = &now
			}
			r.mu.Unlock()
			r.broadcaster.Broadcast("recipe_scan", map[string]interface{}{
				"recipe_id": id,
			})
		}

		onExecute := func(sig RecipeSignal) error {
			broker := findBroker()
			if broker == nil || r.registry.IsPaper() {
				// Paper mode or no broker — record as paper fill.
				bus.Update(sig.ID, "filled", "PAPER-ENTRY", "PAPER-STOP")
				r.broadcaster.Broadcast("recipe_fill", map[string]interface{}{
					"recipe_id": id,
					"signal":    sig,
					"mode":      "paper",
				})
				return nil
			}

			if sig.Action == "exit" {
				orderID, err := broker.Sell(sig.Symbol, sig.Qty)
				if err != nil {
					bus.Update(sig.ID, "rejected", "", "")
					return err
				}
				bus.Update(sig.ID, "filled", orderID, "")
				r.broadcaster.Broadcast("recipe_fill", map[string]interface{}{
					"recipe_id": id,
					"signal":    sig,
					"order_id":  orderID,
				})
				return nil
			}

			entryID, stopID, err := broker.Buy(sig.Symbol, sig.Qty, sig.LimitPrice, sig.StopPrice)
			if err != nil {
				bus.Update(sig.ID, "rejected", "", "")
				return err
			}
			bus.Update(sig.ID, "filled", entryID, stopID)
			r.broadcaster.Broadcast("recipe_fill", map[string]interface{}{
				"recipe_id": id,
				"signal":    sig,
				"entry_id":  entryID,
				"stop_id":   stopID,
			})
			return nil
		}

		err := def.Strategy(ctx, configJSON, onSignal, onExecute)
		if err != nil {
			log.Printf("[recipe] %s exited with error: %v", id, err)
			r.mu.Lock()
			if run, ok := r.running[id]; ok {
				run.err = err.Error()
			}
			r.mu.Unlock()
			r.broadcaster.Broadcast("recipe_state", map[string]interface{}{
				"recipe_id": id,
				"status":    "error",
				"error":     err.Error(),
			})
		}
	}()

	return nil
}

// Stop cancels a running recipe.
func (r *Runner) Stop(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	run, ok := r.running[id]
	if !ok {
		return fmt.Errorf("recipe %s is not running", id)
	}

	run.cancel()
	delete(r.running, id)

	r.broadcaster.Broadcast("recipe_state", map[string]interface{}{
		"recipe_id": id,
		"status":    "stopped",
	})
	return nil
}

// StopAll stops all running recipes. Used during graceful shutdown.
func (r *Runner) StopAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for id, run := range r.running {
		run.cancel()
		r.broadcaster.Broadcast("recipe_state", map[string]interface{}{
			"recipe_id": id,
			"status":    "stopped",
		})
	}
	r.running = make(map[string]*runningRecipe)
}
