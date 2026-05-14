// scheduler — order execution queue for trade-kit
//
// Accepts orders at any time (even when market is closed), queues them by
// market window, and executes automatically when the session opens.
// The daemon runs continuously, checks every 30s, and fires when it's time.
//
// Usage:
//
//	scheduler add sell QUBT 15                         → market sell at next open
//	scheduler add sell QUBT 15 --limit 9.50            → limit sell at next open
//	scheduler add buy SCHD 4 --limit 31.65             → limit buy at next open
//	scheduler add stop EXTR 9 --price 23.50            → new stop order at next open
//	scheduler add modify <ORDER_ID> --stop 23.50       → trail stop at next open
//	scheduler add cancel <ORDER_ID>                    → cancel order at next open
//	scheduler add --at pre_market sell QUBT 15         → sell at pre-market open
//	scheduler list                                     → show all queued orders
//	scheduler cancel <id>                              → remove from queue
//	scheduler clear                                    → cancel all pending
//	scheduler daemon [--log path]                      → start execution daemon
package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"scheduler-bot/internal/queue"
	"tiger-cli/client"
	"tiger-cli/ops"
)

const queuePath = "order-queue.json"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	q, err := queue.Open(queuePath)
	if err != nil {
		fatalf("queue: %v", err)
	}

	cmd := os.Args[1]
	if cmd != "daemon" {
		printTimeHeader()
	}

	switch cmd {
	case "add":
		cmdAdd(q, os.Args[2:])
	case "list", "ls":
		cmdList(q)
	case "cancel":
		if len(os.Args) < 3 {
			fatalf("usage: cancel <id>")
		}
		cmdCancel(q, os.Args[2])
	case "clear":
		cmdClear(q)
	case "daemon":
		logPath := ""
		for i := 2; i < len(os.Args)-1; i++ {
			if os.Args[i] == "--log" {
				logPath = os.Args[i+1]
			}
		}
		cmdDaemon(q, logPath)
	default:
		usage()
		os.Exit(1)
	}
}

// ─── ADD ─────────────────────────────────────────────────────────────────────

func cmdAdd(q *queue.Queue, args []string) {
	if len(args) == 0 {
		fatalf("usage: add [--at window] <type> <symbol|order_id> [qty] [flags]")
	}

	window := queue.WindowNextOpen
	var remaining []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--at" && i+1 < len(args) {
			window = args[i+1]
			i++
		} else {
			remaining = append(remaining, args[i])
		}
	}

	if len(remaining) == 0 {
		fatalf("missing order type (buy|sell|stop|target|modify|cancel)")
	}

	orderType := remaining[0]
	remaining = remaining[1:]

	var o queue.Order
	o.Window = window

	switch orderType {
	case "buy", "sell":
		o = parseBuySell(orderType, remaining)
		o.Window = window
	case "stop":
		o = parseStop(remaining)
		o.Window = window
	case "target":
		o = parseTarget(remaining)
		o.Window = window
	case "modify":
		o = parseModify(remaining)
		o.Window = window
	case "cancel":
		o = parseCancel(remaining)
		o.Window = window
	case "exec":
		o = parseExec(remaining)
		o.Window = window
	default:
		fatalf("unknown order type %q. Use: buy|sell|stop|target|modify|cancel", orderType)
	}

	if err := q.Add(o); err != nil {
		fatalf("add: %v", err)
	}

	// Print confirmation
	all := q.All()
	added := all[len(all)-1]
	sgt := added.ExecuteAt.Add(8 * time.Hour)
	fmt.Printf("✅ Queued [%s]  %s\n", added.ID, added.Summary())
	fmt.Printf("   Window: %s → execute at %s SGT\n", window, sgt.Format("2006-01-02 15:04"))
	if added.Note != "" {
		fmt.Printf("   Note: %s\n", added.Note)
	}
}

func parseBuySell(t string, args []string) queue.Order {
	if len(args) < 2 {
		fatalf("usage: %s <SYMBOL> <QTY> [--limit price] [--note text]", t)
	}
	o := queue.Order{Type: t, Symbol: strings.ToUpper(args[0])}
	qty, err := strconv.Atoi(args[1])
	if err != nil {
		fatalf("invalid qty %q", args[1])
	}
	o.Qty = qty
	for i := 2; i < len(args)-1; i++ {
		switch args[i] {
		case "--limit":
			o.Limit, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		case "--note":
			o.Note = args[i+1]
			i++
		}
	}
	return o
}

func parseStop(args []string) queue.Order {
	if len(args) < 2 {
		fatalf("usage: stop <SYMBOL> <QTY> --price <stop_price>")
	}
	o := queue.Order{Type: queue.TypeStop, Symbol: strings.ToUpper(args[0])}
	o.Qty, _ = strconv.Atoi(args[1])
	for i := 2; i < len(args)-1; i++ {
		if args[i] == "--price" {
			o.Stop, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		}
	}
	if o.Stop == 0 {
		fatalf("--price is required for stop orders")
	}
	return o
}

func parseTarget(args []string) queue.Order {
	if len(args) < 2 {
		fatalf("usage: target <SYMBOL> <QTY> --price <target_price>")
	}
	o := queue.Order{Type: queue.TypeTarget, Symbol: strings.ToUpper(args[0])}
	o.Qty, _ = strconv.Atoi(args[1])
	for i := 2; i < len(args)-1; i++ {
		if args[i] == "--price" {
			o.Target, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		}
	}
	if o.Target == 0 {
		fatalf("--price is required for target orders")
	}
	return o
}

func parseModify(args []string) queue.Order {
	if len(args) < 1 {
		fatalf("usage: modify <ORDER_ID> [--stop price] [--limit price] [--qty n]")
	}
	o := queue.Order{Type: queue.TypeModify, OrderID: args[0]}
	for i := 1; i < len(args)-1; i++ {
		switch args[i] {
		case "--stop":
			o.Stop, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		case "--limit":
			o.Limit, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		case "--qty":
			o.Qty, _ = strconv.Atoi(args[i+1])
			i++
		case "--note":
			o.Note = args[i+1]
			i++
		}
	}
	if o.Stop == 0 && o.Limit == 0 && o.Qty == 0 {
		fatalf("modify requires at least one of: --stop, --limit, --qty")
	}
	return o
}

func parseCancel(args []string) queue.Order {
	if len(args) < 1 {
		fatalf("usage: cancel <ORDER_ID>")
	}
	return queue.Order{Type: queue.TypeCancel, OrderID: args[0]}
}

func parseExec(args []string) queue.Order {
	if len(args) < 1 {
		fatalf("usage: exec \"<shell command>\" [--daily] [--note text]")
	}
	o := queue.Order{Type: queue.TypeExec, Cmd: args[0]}
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--daily":
			o.Daily = true
		case "--note":
			if i+1 < len(args) {
				o.Note = args[i+1]
				i++
			}
		}
	}
	return o
}

// ─── LIST ────────────────────────────────────────────────────────────────────

func cmdList(q *queue.Queue) {
	orders := q.All()
	if len(orders) == 0 {
		fmt.Println("Queue is empty.")
		return
	}

	fmt.Printf("═══ SCHEDULER QUEUE (%d orders) ═══\n\n", len(orders))

	var pending, done []queue.Order
	for _, o := range orders {
		if o.Status == queue.StatusPending {
			pending = append(pending, o)
		} else {
			done = append(done, o)
		}
	}

	if len(pending) > 0 {
		fmt.Printf("  PENDING (%d)\n", len(pending))
		fmt.Printf("  %-8s  %-32s  %-10s  %-20s  %s\n",
			"ID", "ORDER", "WINDOW", "EXECUTE AT (SGT)", "COUNTDOWN")
		fmt.Printf("  %s\n", strings.Repeat("─", 90))
		for _, o := range pending {
			sgt := o.ExecuteAt.Add(8 * time.Hour)
			countdown := time.Until(o.ExecuteAt)
			cStr := formatDuration(countdown)
			fmt.Printf("  %-8s  %-32s  %-10s  %-20s  %s\n",
				o.ID, o.Summary(), o.Window,
				sgt.Format("Mon 15:04"), cStr)
			if o.Note != "" {
				fmt.Printf("  %8s  note: %s\n", "", o.Note)
			}
		}
		fmt.Println()
	}

	if len(done) > 0 {
		fmt.Printf("  HISTORY (%d)\n", len(done))
		for _, o := range done {
			icon := map[string]string{
				queue.StatusFilled:    "✅",
				queue.StatusFailed:    "❌",
				queue.StatusCancelled: "⬜",
			}[o.Status]
			detail := o.Result
			if o.Error != "" {
				detail = o.Error
			}
			fmt.Printf("  %s %-8s  %-32s  %s\n", icon, o.ID, o.Summary(), detail)
		}
		fmt.Println()
	}
}

// ─── CANCEL ──────────────────────────────────────────────────────────────────

func cmdCancel(q *queue.Queue, id string) {
	if err := q.Cancel(id); err != nil {
		fatalf("%v", err)
	}
	fmt.Printf("⬜ Cancelled order %s\n", id)
}

func cmdClear(q *queue.Queue) {
	pending := q.Pending()
	if len(pending) == 0 {
		fmt.Println("No pending orders.")
		return
	}
	for _, o := range pending {
		_ = q.Cancel(o.ID)
	}
	fmt.Printf("⬜ Cleared %d pending order(s)\n", len(pending))
}

// ─── DAEMON ──────────────────────────────────────────────────────────────────

func cmdDaemon(q *queue.Queue, logPath string) {
	var logFile *os.File
	if logPath != "" {
		var err error
		logFile, err = os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			fatalf("log file: %v", err)
		}
		defer logFile.Close()
	}

	logf := func(format string, args ...interface{}) {
		msg := fmt.Sprintf("[%s] "+format, append([]interface{}{nowSGT()}, args...)...)
		fmt.Println(msg)
		if logFile != nil {
			fmt.Fprintln(logFile, msg)
		}
	}

	c, err := client.New(false) // live
	if err != nil {
		fatalf("tiger client: %v", err)
	}

	logf("scheduler daemon started — checking every 30s")
	logf("queue: %s", queuePath)

	pending := q.Pending()
	if len(pending) > 0 {
		logf("%d pending order(s):", len(pending))
		for _, o := range pending {
			sgt := o.ExecuteAt.Add(8 * time.Hour)
			logf("  [%s] %s → %s SGT", o.ID, o.Summary(), sgt.Format("Mon 15:04"))
		}
	} else {
		logf("no pending orders")
	}

	tick := time.NewTicker(30 * time.Second)
	for {
		<-tick.C

		// Re-read queue from disk to pick up orders added by other processes
		q2, err := queue.Open(queuePath)
		if err != nil {
			logf("queue reload error: %v", err)
			continue
		}

		pending := q2.Pending()
		if len(pending) == 0 {
			continue
		}

		now := time.Now().UTC()
		due := filterDue(pending, now)
		if len(due) == 0 {
			next := pending[0].ExecuteAt
			sgt := next.Add(8 * time.Hour)
			logf("%d pending — next window: %s SGT (%s)",
				len(pending), sgt.Format("Mon 15:04"), formatDuration(time.Until(next)))
			continue
		}

		logf("%d order(s) due — executing", len(due))
		for _, o := range due {
			logf("  → %s [%s]", o.Summary(), o.ID)
			result, execErr := execute(c, o)
			if execErr != nil {
				logf("  ❌ FAILED: %v", execErr)
				_ = q2.SetResult(o.ID, queue.StatusFailed, "", execErr.Error())
				_ = q.Cancel(o.ID)
			} else {
				logf("  ✅ FILLED: %s", result)
				_ = q2.SetResult(o.ID, queue.StatusFilled, result, "")
				// Re-queue daily exec orders for the next occurrence.
				if o.Type == queue.TypeExec && o.Daily {
					next := queue.Order{
						Type:   queue.TypeExec,
						Cmd:    o.Cmd,
						Daily:  true,
						Window: o.Window,
						Note:   o.Note,
					}
					if err := q2.Add(next); err == nil {
						logf("  ↻ re-queued daily exec [%s]", o.Cmd[:min(len(o.Cmd), 40)])
					}
				}
			}
		}
	}
}

// filterDue returns orders whose ExecuteAt has passed.
func filterDue(orders []queue.Order, now time.Time) []queue.Order {
	var due []queue.Order
	for _, o := range orders {
		if !now.Before(o.ExecuteAt) {
			due = append(due, o)
		}
	}
	return due
}

// execute submits a queued order to Tiger via the ops package.
func execute(c ops.Caller, o queue.Order) (string, error) {
	switch o.Type {
	case queue.TypeBuy:
		var res ops.OrderResult
		var err error
		if o.Limit > 0 {
			res, err = ops.BuyLimit(c, o.Symbol, o.Qty, o.Limit)
		} else {
			res, err = ops.BuyMarket(c, o.Symbol, o.Qty)
		}
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("order %s", res.OrderID), nil

	case queue.TypeSell:
		// Guard: verify position exists before selling to prevent accidental shorts.
		positions, posErr := ops.GetPositions(c)
		if posErr == nil {
			held := 0
			for _, p := range positions {
				if p.Symbol == o.Symbol {
					held = p.Shares
					break
				}
			}
			if held <= 0 {
				return "", fmt.Errorf("SKIPPED: no %s position held (already sold?)", o.Symbol)
			}
			if held < o.Qty {
				o.Qty = held // sell only what we have
			}
		}
		var res ops.OrderResult
		var err error
		if o.Limit > 0 {
			res, err = ops.SellLimit(c, o.Symbol, o.Qty, o.Limit)
		} else {
			res, err = ops.SellMarket(c, o.Symbol, o.Qty)
		}
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("order %s", res.OrderID), nil

	case queue.TypeStop:
		res, err := ops.SetStopLoss(c, o.Symbol, o.Qty, o.Stop)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("order %s", res.OrderID), nil

	case queue.TypeTarget:
		res, err := ops.SetTakeProfit(c, o.Symbol, o.Qty, o.Target)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("order %s", res.OrderID), nil

	case queue.TypeModify:
		params := ops.ModifyParams{
			StopPrice:  o.Stop,
			LimitPrice: o.Limit,
			Quantity:   o.Qty,
		}
		res, err := ops.ModifyOrder(c, o.OrderID, params)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("modified order %s", res.OrderID), nil

	case queue.TypeCancel:
		res, err := ops.CancelOrder(c, o.OrderID)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("cancelled %s", res.Status), nil

	case queue.TypeExec:
		cmd := exec.Command("sh", "-c", o.Cmd)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%w\n%s", err, string(out))
		}
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		summary := lines[len(lines)-1]
		if len(summary) > 80 {
			summary = summary[:80]
		}
		return fmt.Sprintf("ok: %s", summary), nil
	}
	return "", fmt.Errorf("unknown order type %q", o.Type)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ─── TIME HEADER ─────────────────────────────────────────────────────────────

func printTimeHeader() {
	now := time.Now().UTC()
	sgt := now.Add(8 * time.Hour)
	et := now.Add(-4 * time.Hour) // EDT (UTC-4, covers Mar–Nov)

	state, nextEvent, nextIn := marketState(et)

	fmt.Printf("  %s SGT  |  %s ET  |  %s",
		sgt.Format("Mon 02 Jan 15:04:05"),
		et.Format("15:04:05"),
		state)
	if nextEvent != "" {
		fmt.Printf("  →  %s in %s", nextEvent, formatDuration(nextIn))
	}
	fmt.Printf("\n\n")
}

// marketState returns the current US market state, next event name, and time until it.
func marketState(et time.Time) (state, nextEvent string, nextIn time.Duration) {
	if w := et.Weekday(); w == time.Saturday || w == time.Sunday {
		// Next Monday pre-market
		daysUntilMon := (int(time.Monday) - int(w) + 7) % 7
		if daysUntilMon == 0 {
			daysUntilMon = 7
		}
		next := time.Date(et.Year(), et.Month(), et.Day()+daysUntilMon, 4, 0, 0, 0, time.UTC)
		return "🔴 weekend", "pre-market Mon", time.Until(next.Add(4 * time.Hour))
	}

	h, m, _ := et.Clock()
	mins := h*60 + m

	switch {
	case mins < 4*60:
		// Overnight — next pre-market today
		next := time.Date(et.Year(), et.Month(), et.Day(), 4, 0, 0, 0, time.UTC)
		return "🔴 closed (overnight)", "pre-market", time.Until(next.Add(4 * time.Hour))
	case mins < 9*60+30:
		// Pre-market
		next := time.Date(et.Year(), et.Month(), et.Day(), 9, 30, 0, 0, time.UTC)
		return "🟡 pre-market", "market open", time.Until(next.Add(4 * time.Hour))
	case mins < 16*60:
		// Regular hours
		next := time.Date(et.Year(), et.Month(), et.Day(), 16, 0, 0, 0, time.UTC)
		return "🟢 market open", "market close", time.Until(next.Add(4 * time.Hour))
	case mins < 20*60:
		// After-hours
		next := time.Date(et.Year(), et.Month(), et.Day(), 20, 0, 0, 0, time.UTC)
		return "🟡 after-hours", "closed", time.Until(next.Add(4 * time.Hour))
	default:
		// Overnight — next pre-market tomorrow
		next := time.Date(et.Year(), et.Month(), et.Day()+1, 4, 0, 0, 0, time.UTC)
		return "🔴 closed (overnight)", "pre-market", time.Until(next.Add(4 * time.Hour))
	}
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func formatDuration(d time.Duration) string {
	if d < 0 {
		return "NOW"
	}
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h >= 24 {
		return fmt.Sprintf("%dd %dh", h/24, h%24)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func nowSGT() string {
	return time.Now().UTC().Add(8 * time.Hour).Format("15:04:05 SGT")
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "scheduler: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`scheduler — order execution queue

Accepts orders anytime (market open or closed). Executes them automatically
when the target market window opens. Daemon runs continuously in background.

COMMANDS
  add [--at window] <type> <args>   Queue an order
  list                              Show all queued and executed orders
  cancel <id>                       Remove a pending order from queue
  clear                             Cancel all pending orders
  daemon [--log path]               Start execution daemon (blocks)

ORDER TYPES
  buy  <SYMBOL> <QTY> [--limit price]           Market or limit buy
  sell <SYMBOL> <QTY> [--limit price]           Market or limit sell
  stop <SYMBOL> <QTY> --price <price>           New stop-loss order
  target <SYMBOL> <QTY> --price <price>         New take-profit order
  modify <ORDER_ID> [--stop p] [--limit p] [--qty n]  Modify existing order
  cancel <ORDER_ID>                             Cancel existing Tiger order

WINDOWS (--at flag)
  next_open    US regular session open: 9:30 AM ET / 21:30 SGT  (default)
  pre_market   US pre-market open:      4:00 AM ET / 16:00 SGT
  now          Execute on next daemon tick (market must be open)

EXAMPLES
  # Queue tonight's orders now (market closed)
  scheduler add sell QUBT 15
  scheduler add buy SCHD 4 --limit 31.65
  scheduler add modify 43116645746347008 --stop 23.50
  scheduler add --at pre_market sell QUBT 15

  # Review and manage queue
  scheduler list
  scheduler cancel a3f2b1

  # Start daemon (keep running in background)
  scheduler daemon &
  scheduler daemon --log ~/scheduler.log &

`)
}
