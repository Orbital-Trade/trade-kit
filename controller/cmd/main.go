// controller — master controller for trade-kit strategy bots
//
// Commands:
//
//	status      Full dashboard (NAV trajectory, circuit breaker, gate, pattern, signals)
//	monitor     Signal bus detail with expiry countdowns
//	gate [cost] Pre-flight check — can you open a new position right now?
//	bootstrap N Show 40/60 split + 50/50 reinvestment for N dollars of profit
//	estop       EMERGENCY: cancel all orders + market-sell all positions
package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"orbital-ctrl/internal/signal"
	"tiger-cli/client"
	"tiger-cli/ops"
)

// ─── CONFIG ──────────────────────────────────────────────────────────────────

type Config struct {
	T1Symbols              []string `json:"t1_symbols"`
	PositionMaxPct         float64  `json:"position_max_pct"`
	CashReservePct         float64  `json:"cash_reserve_pct"`
	MaxPositions           int      `json:"max_positions"`
	CircuitBreakerAlertPct float64  `json:"circuit_breaker_alert_pct"`
	CircuitBreakerEstopPct float64  `json:"circuit_breaker_estop_pct"`
	RRMinSwing             float64  `json:"rr_min_swing"`
	RRMinScalp             float64  `json:"rr_min_scalp"`
}

var defaultConfig = Config{
	T1Symbols:              []string{"SCHD", "SGOV"},
	PositionMaxPct:         30.0,
	CashReservePct:         15.0,
	MaxPositions:           10,
	CircuitBreakerAlertPct: 10.0,
	CircuitBreakerEstopPct: 15.0,
	RRMinSwing:             3.0,
	RRMinScalp:             4.0,
}

func loadConfig() Config {
	data, err := os.ReadFile("controller.json")
	if err != nil {
		return defaultConfig
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return defaultConfig
	}
	return cfg
}

func (c Config) isT1(symbol string) bool {
	for _, s := range c.T1Symbols {
		if s == symbol {
			return true
		}
	}
	return false
}

// ─── NAV HISTORY (circuit breaker baseline) ───────────────────────────────

type NavEntry struct {
	Date      string  `json:"date"`
	NAV       float64 `json:"nav"`
	Timestamp string  `json:"timestamp"`
}

type NavHistory struct {
	SessionStart NavEntry `json:"session_start"`
	Yesterday    NavEntry `json:"yesterday"`
}

const navHistoryPath = "nav-history.json"

func loadNavHistory() NavHistory {
	data, err := os.ReadFile(navHistoryPath)
	if err != nil {
		return NavHistory{}
	}
	var h NavHistory
	_ = json.Unmarshal(data, &h)
	return h
}

func updateNavHistory(currentNAV float64) NavHistory {
	today := time.Now().Format("2006-01-02")
	h := loadNavHistory()

	if h.SessionStart.Date != today {
		// New day: promote session_start to yesterday, reset session_start
		if h.SessionStart.NAV > 0 {
			h.Yesterday = h.SessionStart
		}
		h.SessionStart = NavEntry{
			Date:      today,
			NAV:       currentNAV,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		data, _ := json.MarshalIndent(h, "", "  ")
		_ = os.WriteFile(navHistoryPath, data, 0644)
	}
	return h
}

// ─── MAIN ────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	signalsPath := "signals.json"
	var simulateDrawdown float64
	var remaining []string

	for i := 1; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "--signals":
			if i+1 < len(os.Args) {
				signalsPath = os.Args[i+1]
				i++
			}
		case "--simulate-drawdown":
			if i+1 < len(os.Args) {
				simulateDrawdown, _ = strconv.ParseFloat(os.Args[i+1], 64)
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

	cfg := loadConfig()

	bus, err := signal.Open(signalsPath)
	if err != nil {
		fatalf("signals %q: %v", signalsPath, err)
	}

	switch remaining[0] {
	case "status":
		cmdStatus(cfg, bus, signalsPath, simulateDrawdown)
	case "monitor":
		cmdMonitor(bus)
	case "gate":
		cost := 0.0
		if len(remaining) >= 2 {
			cost, _ = strconv.ParseFloat(remaining[1], 64)
		}
		cmdGate(cfg, cost)
	case "bootstrap":
		if len(remaining) < 2 {
			fatalf("usage: bootstrap <profit>")
		}
		profit, err := strconv.ParseFloat(remaining[1], 64)
		if err != nil {
			fatalf("invalid profit amount: %s", remaining[1])
		}
		cmdBootstrap(profit)
	case "estop":
		cmdEstop()
	default:
		usage()
		os.Exit(1)
	}
}

// ─── STATUS ──────────────────────────────────────────────────────────────────

func cmdStatus(cfg Config, bus *signal.Bus, signalsPath string, simulateDrawdown float64) {
	_ = bus.Reload()

	c, err := client.New(false)
	if err != nil {
		fatalf("tiger client: %v", err)
	}

	account, _ := ops.GetAccount(c)
	positions, _ := ops.GetPositions(c)
	orders, _ := ops.GetOrders(c)
	sigs := bus.All()

	// Update nav history (circuit breaker baseline) — use real NAV
	navHist := updateNavHistory(account.NetLiquidation)

	// Apply simulated drawdown AFTER recording real NAV history
	if simulateDrawdown > 0 {
		account.NetLiquidation *= (1 - simulateDrawdown/100)
		account.Cash *= (1 - simulateDrawdown/100)
		account.GrossPositionValue *= (1 - simulateDrawdown/100)
	}

	now := time.Now()
	fmt.Printf("═══ ORBITAL CTRL — %s ═══\n\n", now.Format("2006-01-02 15:04 MST"))

	if simulateDrawdown > 0 {
		fmt.Printf("  ⚠️  SIMULATED DRAWDOWN: -%.1f%% applied (display only — real NAV unchanged)\n\n", simulateDrawdown)
	}

	// 1. NAV Trajectory + Circuit Breaker
	printNavTrajectory(account.NetLiquidation, navHist, cfg)

	// 2. Playbook Gate
	printGate(cfg, positions, account)

	// 3. Per-bot signal expiry countdowns
	printExpiryCountdowns(sigs)

	// 4. Pattern classifier
	printPattern(sigs, positions, account, cfg)

	// 5. Positions (T1 / T2 separated)
	printPositions(cfg, positions)

	// 6. Capital deployment (T2 view)
	printCapitalDeployment(cfg, positions, account)

	// 7. Open orders
	printOpenOrders(orders)

	// 8. Signal bus with R/R warnings
	printSignalBus(cfg, sigs)

	// 9. Bootstrap for filled signals
	printBootstrapFilled(sigs)

	// 10. Daily log
	writeDailyLog(account, positions, sigs)
}

func printNavTrajectory(currentNAV float64, h NavHistory, cfg Config) {
	fmt.Printf("  NAV TRAJECTORY\n")
	fmt.Printf("  Current:         $%.2f\n", currentNAV)

	if h.SessionStart.NAV > 0 {
		sessionChange := currentNAV - h.SessionStart.NAV
		sessionPct := sessionChange / h.SessionStart.NAV * 100
		fmt.Printf("  Session start:   $%.2f  (%+.2f / %+.1f%%  since %s)\n",
			h.SessionStart.NAV, sessionChange, sessionPct, h.SessionStart.Timestamp[:16])

		// Circuit breaker
		alertLevel := h.SessionStart.NAV * (1 - cfg.CircuitBreakerAlertPct/100)
		estopLevel := h.SessionStart.NAV * (1 - cfg.CircuitBreakerEstopPct/100)
		distToAlert := currentNAV - alertLevel

		cbStatus := "✅"
		cbMsg := fmt.Sprintf("alert at $%.0f (-%d%%), $%.0f away", alertLevel, int(cfg.CircuitBreakerAlertPct), distToAlert)
		if currentNAV <= estopLevel {
			cbStatus = "🚨"
			cbMsg = fmt.Sprintf("BELOW ESTOP LEVEL — run estop immediately!")
		} else if currentNAV <= alertLevel {
			cbStatus = "🔴"
			cbMsg = fmt.Sprintf("CIRCUIT BREAKER TRIGGERED — close all T2 positions")
		}
		fmt.Printf("  Circuit breaker: %s %s\n", cbStatus, cbMsg)
	}

	if h.Yesterday.NAV > 0 {
		prevChange := currentNAV - h.Yesterday.NAV
		prevPct := prevChange / h.Yesterday.NAV * 100
		fmt.Printf("  Yesterday close: $%.2f  (%+.2f / %+.1f%%)\n",
			h.Yesterday.NAV, prevChange, prevPct)
	}
	fmt.Println()
}

func printGate(cfg Config, positions []ops.Position, account ops.Account) {
	nliq := account.NetLiquidation

	posStatus, posDetail := "✅", fmt.Sprintf("%d/%d open", len(positions), cfg.MaxPositions)
	if len(positions) >= cfg.MaxPositions {
		posStatus = "🔴"
		posDetail = fmt.Sprintf("MAX REACHED (%d/%d)", len(positions), cfg.MaxPositions)
	} else if len(positions) >= cfg.MaxPositions-2 {
		posStatus = "🟡"
	}

	cashStatus, cashDetail := "✅", "n/a"
	if nliq > 0 {
		cashPct := account.Cash / nliq * 100
		cashDetail = fmt.Sprintf("%.0f%% (min %.0f%%)", cashPct, cfg.CashReservePct)
		if cashPct < cfg.CashReservePct {
			cashStatus = "🔴"
		} else if cashPct < cfg.CashReservePct+5 {
			cashStatus = "🟡"
		}
	}

	maxPosRatio, maxPosSymbol := 0.0, ""
	for _, p := range positions {
		r := p.MarketValue / nliq * 100
		if r > maxPosRatio {
			maxPosRatio = r
			maxPosSymbol = p.Symbol
		}
	}
	posRatioStatus := "✅"
	posRatioDetail := fmt.Sprintf("largest %.0f%% (%s, max %.0f%%)", maxPosRatio, maxPosSymbol, cfg.PositionMaxPct)
	if maxPosRatio > cfg.PositionMaxPct {
		posRatioStatus = "🟡" // warn but don't block — existing position, can't force-sell
	}

	// Max available for next trade
	maxByPct := nliq * cfg.PositionMaxPct / 100
	maxByCash := account.Cash - nliq*cfg.CashReservePct/100
	maxNext := math.Min(maxByPct, maxByCash)

	fmt.Printf("  PLAYBOOK GATE\n")
	fmt.Printf("  %s Positions:    %s\n", posStatus, posDetail)
	fmt.Printf("  %s Cash reserve: %s\n", cashStatus, cashDetail)
	fmt.Printf("  %s Largest pos:  %s\n", posRatioStatus, posRatioDetail)
	if maxNext > 0 {
		fmt.Printf("  → Max next trade: $%.0f  (30%% NAV=$%.0f, cash-limited=$%.0f)\n",
			maxNext, maxByPct, math.Max(0, maxByCash))
	} else {
		fmt.Printf("  → 🔴 No capital available for new trades\n")
	}
	fmt.Println()
}

func printExpiryCountdowns(sigs []signal.Signal) {
	var pending []signal.Signal
	for _, s := range sigs {
		if s.Status == signal.StatusPending {
			pending = append(pending, s)
		}
	}
	if len(pending) == 0 {
		return
	}
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].ExpiresAt.Before(pending[j].ExpiresAt)
	})

	fmt.Printf("  SIGNAL EXPIRY\n")
	for _, s := range pending {
		expires := time.Until(s.ExpiresAt)
		emoji := expiryEmoji(expires)
		bar := expiryBar(expires, 48*time.Hour)
		fmt.Printf("  %s%-10s  %-6s  %s  %s\n",
			emoji, s.Strategy, s.Symbol, formatExpiry(expires), bar)
	}
	fmt.Println()
}

func printPattern(sigs []signal.Signal, positions []ops.Position, account ops.Account, cfg Config) {
	// Count pending signals per strategy
	byStrat := map[string]int{}
	for _, s := range sigs {
		if s.Status == signal.StatusPending && !s.IsExpired() {
			byStrat[s.Strategy]++
		}
	}
	activeStrategies := len(byStrat)

	// Can gate allow a new trade?
	nliq := account.NetLiquidation
	gateOpen := len(positions) < cfg.MaxPositions &&
		account.Cash-nliq*cfg.CashReservePct/100 > 0

	var pattern, detail, priority string
	switch {
	case activeStrategies == 0:
		pattern = "B"
		detail = "No signals — cash is a position"
		priority = "Consider monthly SCHD contribution if deposit day"
	case activeStrategies == 1:
		pattern = "C"
		for strat := range byStrat {
			detail = fmt.Sprintf("One signal (%s) — cleanest case, full attention", strat)
		}
		priority = "Enter with conviction if gate is open and news is verified"
	default:
		pattern = "A"
		detail = fmt.Sprintf("%d strategies firing — capital constraint dominates", activeStrategies)
		priority = "Priority: earnings → daytrader → bounce → reversal (paper only)"
	}

	icon := map[string]string{"A": "⚡", "B": "💤", "C": "🎯"}[pattern]
	fmt.Printf("  PATTERN %s %s — %s\n", pattern, icon, detail)
	if !gateOpen {
		fmt.Printf("  ⛔ Gate closed — no new entries until cash or position limit clears\n")
	}
	fmt.Printf("  %s\n\n", priority)
}

func printPositions(cfg Config, positions []ops.Position) {
	if len(positions) == 0 {
		fmt.Println("  No open positions.\n")
		return
	}
	var t1, t2 []ops.Position
	for _, p := range positions {
		if cfg.isT1(p.Symbol) {
			t1 = append(t1, p)
		} else {
			t2 = append(t2, p)
		}
	}

	hdr := "  %-6s  %5s  %9s  %9s  %10s\n"
	sep := "  " + strings.Repeat("─", 48) + "\n"
	row := "  %-6s  %5d  %9.2f  %9.2f  %+10.2f\n"

	printPosGroup := func(label string, ps []ops.Position) {
		if len(ps) == 0 {
			return
		}
		var total float64
		fmt.Printf("  %s (%d)\n", label, len(ps))
		fmt.Printf(hdr, "SYMBOL", "QTY", "AVG COST", "MKT PRC", "UNREAL P&L")
		fmt.Print(sep)
		for _, p := range ps {
			fmt.Printf(row, p.Symbol, p.Shares, p.AvgCost, p.MarketPrice, p.UnrealizedPnL)
			total += p.UnrealizedPnL
		}
		fmt.Print(sep)
		fmt.Printf("  %-6s  %5s  %9s  %9s  %+10.2f\n\n", "TOTAL", "", "", "", total)
	}

	printPosGroup("TRACK 1", t1)
	printPosGroup("TRACK 2", t2)
}

func printCapitalDeployment(cfg Config, positions []ops.Position, account ops.Account) {
	nliq := account.NetLiquidation
	if nliq <= 0 {
		return
	}
	t2Estimated := nliq / 2

	var t2Deployed float64
	for _, p := range positions {
		if !cfg.isT1(p.Symbol) {
			t2Deployed += p.MarketValue
		}
	}
	t2CashEst := account.Cash / 2
	t2Util := t2Deployed / t2Estimated * 100

	utilBar := progressBar(t2Util, 100, 10)
	fmt.Printf("  TRACK 2 CAPITAL\n")
	fmt.Printf("  T2 estimated:  $%.0f (NAV÷2)\n", t2Estimated)
	fmt.Printf("  T2 deployed:   $%.0f  %s %.0f%%\n", t2Deployed, utilBar, t2Util)
	fmt.Printf("  T2 cash est:   $%.0f (of $%.0f total cash)\n", t2CashEst, account.Cash)
	if t2Util > 85 {
		fmt.Printf("  ⚠️  T2 near full deployment — exits needed before new entries\n")
	}
	fmt.Println()
}

func printOpenOrders(orders []ops.Order) {
	var open []ops.Order
	for _, o := range orders {
		if o.Status == "PendingSubmit" || o.Status == "PreSubmitted" || o.Status == "Submitted" {
			open = append(open, o)
		}
	}
	if len(open) == 0 {
		return
	}
	fmt.Printf("  OPEN ORDERS (%d)\n", len(open))
	for _, o := range open {
		if o.StopPrice > 0 {
			fmt.Printf("  %-20s  %-6s  STOP  qty %-4d  trigger $%.2f\n",
				o.ID, o.Symbol, o.Quantity, o.StopPrice)
		} else {
			fmt.Printf("  %-20s  %-6s  %-4s  qty %-4d  limit $%.2f\n",
				o.ID, o.Symbol, o.OrderType, o.Quantity, o.LimitPrice)
		}
	}
	fmt.Println()
}

func printSignalBus(cfg Config, sigs []signal.Signal) {
	if len(sigs) == 0 {
		fmt.Println("  Signal bus empty.")
		return
	}

	// Summary by strategy
	type stratCount struct {
		pending, active, other int
	}
	byStrat := map[string]*stratCount{}
	for _, s := range sigs {
		if byStrat[s.Strategy] == nil {
			byStrat[s.Strategy] = &stratCount{}
		}
		switch s.Status {
		case signal.StatusPending:
			byStrat[s.Strategy].pending++
		case signal.StatusActive:
			byStrat[s.Strategy].active++
		default:
			byStrat[s.Strategy].other++
		}
	}
	strategies := []string{}
	for k := range byStrat {
		strategies = append(strategies, k)
	}
	sort.Strings(strategies)

	fmt.Printf("  SIGNAL BUS (%d total)\n", len(sigs))
	for _, strat := range strategies {
		c := byStrat[strat]
		tag := ""
		if strat == "futures" || strat == "reversal" {
			tag = "  [paper/research — excluded from capital]"
		}
		fmt.Printf("  %-12s  %d pending  %d active  %d closed%s\n",
			strat, c.pending, c.active, c.other, tag)
	}

	// Pending signals with R/R check
	var pending []signal.Signal
	for _, s := range sigs {
		if s.Status == signal.StatusPending {
			pending = append(pending, s)
		}
	}
	if len(pending) > 0 {
		sort.Slice(pending, func(i, j int) bool {
			return pending[i].ExpiresAt.Before(pending[j].ExpiresAt)
		})
		fmt.Printf("\n  PENDING (%d)\n", len(pending))
		for _, s := range pending {
			expires := time.Until(s.ExpiresAt)
			emoji := expiryEmoji(expires)
			rrStr := rrWarning(cfg, s)
			fmt.Printf("  %s%-6s  %-10s  %d sh @ $%.2f  stop $%.2f  %s\n",
				emoji, s.Symbol, s.Strategy, s.Qty, s.EntryLimit, s.Stop, rrStr)
			fmt.Printf("      expires %s  |  %s\n", formatExpiry(expires), s.Reason)
		}
	}
	fmt.Println()
}

// rrWarning returns a R/R status string for a pending signal.
func rrWarning(cfg Config, s signal.Signal) string {
	if s.Target <= 0 {
		return "R/R n/a (exit-based)"
	}
	if s.EntryLimit <= s.Stop {
		return "R/R n/a (invalid)"
	}
	rr := (s.Target - s.EntryLimit) / (s.EntryLimit - s.Stop)
	minRR := cfg.RRMinSwing
	if s.Strategy == "daytrader" {
		minRR = cfg.RRMinScalp
	}
	if rr < minRR {
		return fmt.Sprintf("⚠️  R/R %.1f:1 BELOW %.0f:1 minimum", rr, minRR)
	}
	return fmt.Sprintf("R/R %.1f:1 ✅", rr)
}

func printBootstrapFilled(sigs []signal.Signal) {
	var filled []signal.Signal
	for _, s := range sigs {
		if s.Status == signal.StatusFilled && s.FilledPrice > 0 && s.EntryLimit > 0 {
			filled = append(filled, s)
		}
	}
	if len(filled) == 0 {
		return
	}
	fmt.Printf("  BOOTSTRAP (filled positions)\n")
	for _, s := range filled {
		profit := (s.FilledPrice - s.EntryLimit) * float64(s.Qty)
		if profit <= 0 {
			continue
		}
		out, reinvest := profit*0.40, profit*0.60
		t1, t2 := reinvest/2, reinvest/2
		fmt.Printf("  %-6s  profit $%.2f  → savings $%.2f  reinvest $%.2f (T1 $%.2f / T2 $%.2f)\n",
			s.Symbol, profit, out, reinvest, t1, t2)
	}
	fmt.Println()
}

// ─── MONITOR ─────────────────────────────────────────────────────────────────

func cmdMonitor(bus *signal.Bus) {
	_ = bus.Reload()
	sigs := bus.All()
	fmt.Printf("═══ SIGNAL BUS — %s (%d signals) ═══\n\n",
		time.Now().Format("2006-01-02 15:04 MST"), len(sigs))

	if len(sigs) == 0 {
		fmt.Println("  Empty.")
		return
	}

	statusOrder := map[string]int{
		signal.StatusPending:  0,
		signal.StatusActive:   1,
		signal.StatusFilled:   2,
		signal.StatusExpired:  3,
		signal.StatusRejected: 4,
		signal.StatusDeclined: 5,
	}
	sort.Slice(sigs, func(i, j int) bool {
		oi, oj := statusOrder[sigs[i].Status], statusOrder[sigs[j].Status]
		if oi != oj {
			return oi < oj
		}
		return sigs[i].CreatedAt.After(sigs[j].CreatedAt)
	})

	for _, s := range sigs {
		switch s.Status {
		case signal.StatusPending:
			expires := time.Until(s.ExpiresAt)
			bar := expiryBar(expires, 48*time.Hour)
			emoji := expiryEmoji(expires)
			fmt.Printf("  %s%-6s  %-8s  %-10s  %d sh @ $%.2f  stop $%.2f  target $%.2f\n",
				emoji, s.Symbol, s.Status, s.Strategy, s.Qty, s.EntryLimit, s.Stop, s.Target)
			fmt.Printf("       expires %s  %s\n       %s\n\n", formatExpiry(expires), bar, s.Reason)
		case signal.StatusActive:
			age := time.Since(s.CreatedAt).Round(time.Minute)
			fmt.Printf("  🟢  %-6s  %-8s  %-10s  entry %s  (%v ago)\n\n",
				s.Symbol, s.Status, s.Strategy, s.EntryOrderID, age)
		case signal.StatusFilled:
			profit := (s.FilledPrice - s.EntryLimit) * float64(s.Qty)
			fmt.Printf("  ✅  %-6s  %-8s  %-10s  entry $%.2f → exit $%.2f  P&L $%+.2f\n\n",
				s.Symbol, s.Status, s.Strategy, s.EntryLimit, s.FilledPrice, profit)
		default:
			age := time.Since(s.CreatedAt).Round(time.Minute)
			fmt.Printf("  ⬜  %-6s  %-8s  %-10s  %v ago  %s\n\n",
				s.Symbol, s.Status, s.Strategy, age, s.Notes)
		}
	}
}

// ─── GATE ────────────────────────────────────────────────────────────────────

func cmdGate(cfg Config, newCost float64) {
	c, err := client.New(false)
	if err != nil {
		fatalf("tiger client: %v", err)
	}
	positions, _ := ops.GetPositions(c)
	account, _ := ops.GetAccount(c)

	fmt.Printf("═══ PRE-FLIGHT GATE CHECK ═══\n\n")
	if newCost > 0 {
		fmt.Printf("  Proposed trade cost: $%.2f\n\n", newCost)
	}

	printGate(cfg, positions, account)

	if newCost > 0 {
		nliq := account.NetLiquidation
		fmt.Printf("  VERDICT FOR $%.0f TRADE:\n", newCost)
		if len(positions) >= cfg.MaxPositions {
			fmt.Printf("  🔴  BLOCKED: max positions (%d/%d)\n", len(positions), cfg.MaxPositions)
			return
		}
		posRatio := newCost / nliq * 100
		if posRatio > cfg.PositionMaxPct {
			fmt.Printf("  🔴  BLOCKED: trade %.0f%% of NAV > %.0f%% limit\n", posRatio, cfg.PositionMaxPct)
			return
		}
		cashAfter := account.Cash - newCost
		cashRatio := cashAfter / nliq * 100
		if cashRatio < cfg.CashReservePct {
			fmt.Printf("  🔴  BLOCKED: cash reserve would drop to %.0f%% (min %.0f%%)\n", cashRatio, cfg.CashReservePct)
			return
		}
		fmt.Printf("  ✅  CLEAR — %.0f%% of NAV, cash reserve %.0f%% after trade\n", posRatio, cashRatio)
	}
}

// ─── BOOTSTRAP ───────────────────────────────────────────────────────────────

func cmdBootstrap(profit float64) {
	out := profit * 0.40
	reinvest := profit * 0.60
	t1 := reinvest / 2
	t2 := reinvest / 2

	fmt.Printf("═══ BOOTSTRAP CALCULATOR ═══\n\n")
	fmt.Printf("  Net profit:        $%.2f\n\n", profit)
	fmt.Printf("  40%% → savings:     $%.2f  (transfer out of trading system)\n", out)
	fmt.Printf("  60%% → reinvest:    $%.2f\n", reinvest)
	fmt.Printf("     T1 (SCHD/div):  $%.2f  (buy at next opportunity)\n", t1)
	fmt.Printf("     T2 (cash):      $%.2f  (stays in Tiger for next play)\n\n", t2)
	fmt.Printf("  Total accounted:   $%.2f\n", out+reinvest)
}

// ─── EMERGENCY STOP ──────────────────────────────────────────────────────────

func cmdEstop() {
	fmt.Println("⚠️  EMERGENCY STOP — cancel ALL orders + market-sell ALL positions.")
	fmt.Print("Type 'CONFIRM' to proceed: ")
	var input string
	fmt.Scanln(&input)
	if input != "CONFIRM" {
		fmt.Println("Aborted.")
		return
	}

	c, err := client.New(false)
	if err != nil {
		fatalf("tiger client: %v", err)
	}

	orders, _ := ops.GetOrders(c)
	cancelled := 0
	for _, o := range orders {
		if o.Status == "PendingSubmit" || o.Status == "PreSubmitted" || o.Status == "Submitted" {
			if _, err := ops.CancelOrder(c, o.ID); err != nil {
				fmt.Printf("  cancel %s: %v\n", o.ID, err)
			} else {
				fmt.Printf("  cancelled %s (%s %s)\n", o.ID, o.Action, o.Symbol)
				cancelled++
			}
		}
	}
	fmt.Printf("  %d order(s) cancelled\n\n", cancelled)

	positions, _ := ops.GetPositions(c)
	for _, p := range positions {
		shares := int(math.Abs(float64(p.Shares)))
		if shares == 0 {
			continue
		}
		result, err := ops.SellMarket(c, p.Symbol, shares)
		if err != nil {
			fmt.Printf("  sell %s: %v\n", p.Symbol, err)
		} else {
			fmt.Printf("  SOLD %s %d shares (market)  order %s\n", p.Symbol, shares, result.OrderID)
		}
	}
	fmt.Println("\n  Emergency stop complete.")
}

// ─── DAILY LOG ───────────────────────────────────────────────────────────────

type dailyLog struct {
	Date        string  `json:"date"`
	GeneratedAt string  `json:"generated_at"`
	Account     struct {
		NetLiquidation float64 `json:"net_liquidation"`
		Cash           float64 `json:"cash"`
	} `json:"account"`
	Positions  []ops.Position          `json:"positions"`
	Signals    signalCounts            `json:"signals"`
	ByStrategy map[string]map[string]int `json:"by_strategy"`
	Detail     []signal.Signal         `json:"signals_detail"`
}

type signalCounts struct {
	Total, Pending, Active, Filled, Expired, Rejected, Declined int
}

func writeDailyLog(account ops.Account, positions []ops.Position, sigs []signal.Signal) {
	log := dailyLog{
		Date:        time.Now().Format("2006-01-02"),
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Positions:   positions,
		Detail:      sigs,
		ByStrategy:  map[string]map[string]int{},
	}
	log.Account.NetLiquidation = account.NetLiquidation
	log.Account.Cash = account.Cash

	for _, s := range sigs {
		log.Signals.Total++
		switch s.Status {
		case signal.StatusPending:
			log.Signals.Pending++
		case signal.StatusActive:
			log.Signals.Active++
		case signal.StatusFilled:
			log.Signals.Filled++
		case signal.StatusExpired:
			log.Signals.Expired++
		case signal.StatusRejected:
			log.Signals.Rejected++
		case signal.StatusDeclined:
			log.Signals.Declined++
		}
		if log.ByStrategy[s.Strategy] == nil {
			log.ByStrategy[s.Strategy] = map[string]int{}
		}
		log.ByStrategy[s.Strategy][s.Status]++
	}

	_ = os.MkdirAll("logs", 0755)
	path := filepath.Join("logs", fmt.Sprintf("daily-%s.json", log.Date))
	data, _ := json.MarshalIndent(log, "", "  ")
	_ = os.WriteFile(path, data, 0644)
	fmt.Printf("  Daily log → %s\n\n", path)
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func formatExpiry(d time.Duration) string {
	if d < 0 {
		return "EXPIRED"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h >= 48 {
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	}
	return fmt.Sprintf("%dh %dm", h, m)
}

func expiryEmoji(d time.Duration) string {
	if d < 0 {
		return "💀  "
	}
	if d < 2*time.Hour {
		return "🔴  "
	}
	if d < 24*time.Hour {
		return "🟡  "
	}
	return "    "
}

func expiryBar(remaining, total time.Duration) string {
	if total <= 0 {
		return ""
	}
	pct := float64(remaining) / float64(total)
	if pct > 1 {
		pct = 1
	}
	if pct < 0 {
		pct = 0
	}
	filled := int(pct * 10)
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", 10-filled) + "]"
}

func progressBar(value, max float64, width int) string {
	pct := value / max
	if pct > 1 {
		pct = 1
	}
	filled := int(pct * float64(width))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return "[" + bar + "]"
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "orbital-ctrl: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`controller — master controller for trade-kit strategy bots

Usage:
  orbital-ctrl [--signals path] <command>

Commands:
  status              Full dashboard: NAV trajectory, circuit breaker, gate,
                      pattern, positions, capital deployment, signals with R/R
  monitor             Signal bus detail with expiry countdowns
  gate [cost]         Pre-flight check — pass a dollar amount to check a specific trade
  bootstrap <profit>  Show 40/60 split + 50/50 reinvestment for closed profit
  estop               EMERGENCY: cancel all orders + market-sell all positions

Flags:
  --signals path              Path to shared signals.json (default: ./signals.json)
  --simulate-drawdown PCT     Fake a NAV drop of PCT% to test the circuit breaker

Config:
  controller.json in working directory. Contains T1 symbols, gate thresholds,
  R/R minimums, circuit breaker levels.

State files (auto-created):
  nav-history.json  — session/yesterday NAV for circuit breaker
  logs/daily-*.json — end-of-day summaries written on every status run

Playbook gate (enforced in --semi and --live bot modes):
  Max 10 concurrent positions
  New position ≤ 30% of total NAV  [note: base = total NAV, not T2-only]
  Cash reserve ≥ 15% of NAV after trade

30% per position — base clarification:
  The 30% rule uses total account NAV (not T2-only allocation) because at
  current capital scale ($700-800), T2-only (50% of NAV = ~$385) would cap
  positions at ~$115 — below minimum viable size. Rule scales correctly as
  capital grows: at $2,000 NAV, 30% = $600; at $5,000, 30% = $1,500.

R/R minimums:
  Swings (earnings, bounce):  3:1
  Intraday scalps (daytrader): 3:1  [updated from 2:1 — matches playbook]
  FVG scalps:                 4:1
  MES reversal (paper):       2:1   [research mode, excluded from capital]

reversal_bot status:
  PAPER / RESEARCH ONLY — excluded from capital allocation until H2 validated.
  Do not size into reversal_bot trades. Run paper mode only.

`)
}
