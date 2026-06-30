// etoro-cli — execute eToro trades from the command line.
//
// Each subcommand maps 1:1 to a function in ops/. This file is a thin
// dispatcher only.
//
// Usage:
//
//	etoro-cli [--paper|--live] [--json] <command> [args]
//
// Build:
//
//	cd etoro && go build -o etoro-cli ./cmd/
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"etoro-cli/client"
	"etoro-cli/internal/tlog"
	"etoro-cli/ops"
)

// Compile-time proof that *client.EtoroClient satisfies ops.Caller.
var _ ops.Caller = (*client.EtoroClient)(nil)

const usage = `etoro-cli — eToro trade execution CLI

Usage:
  etoro-cli [--paper|--live] [--json] <command> [args]

Global flags:
  --paper   Demo/paper mode (default — no real orders sent)
  --live    Live trading mode (requires Y confirmation for write ops)
  --json    Output as JSON (for scripting)

Read commands (safe — no confirmation needed):
  positions                         List open positions
  account                           Account balances (equity, cash, invested)
         --type <accountType>       Filter by account type (Trading, Crypto, etc.)
         --history                  Show balance history
         --currency <ISO>           Display currency (default: USD)
  quote  <SYMBOL>                   Real-time price snapshot
  orders                            List pending orders
  search <QUERY>                    Search instruments by name or symbol

Write commands (paper: log only  |  live: confirm then execute):
  buy    <SYMBOL> <AMOUNT>          Market buy (amount in USD)
         --limit  <price>           → Limit buy instead
         --stop   <price>           Set stop-loss on the position
         --target <price>           Set take-profit on the position

  sell   <SYMBOL>                   Close all positions for symbol
         <SYMBOL> <AMOUNT>          Close positions up to amount

  close  <POSITION_ID>             Close a specific position by ID

  stop   <POSITION_ID>             Set stop-loss on existing position
         --price  <price>           Stop trigger price (required)

  target <POSITION_ID>             Set take-profit on existing position
         --price  <price>           Target price (required)

  cancel <ORDER_ID>                Cancel a pending order
  modify <ORDER_ID|POSITION_ID>    Modify an order or position
         --stop   <price>           New stop-loss price
         --target <price>           New take-profit price
         --limit  <price>           New limit price (orders only)

Watchlist commands:
  watchlist                         List all watchlists
  watchlist create <NAME>           Create a new watchlist
  watchlist add <ID> <SYMBOLS...>   Add instruments to a watchlist
  watchlist delete <ID>             Delete a watchlist

Alert commands:
  alert list                        List active price alerts
  alert create <SYMBOL>             Create a price alert
         --above <price>            Alert when price goes above
         --below <price>            Alert when price goes below
  alert delete <ID>                 Delete an alert

Examples:
  etoro-cli positions
  etoro-cli quote AAPL
  etoro-cli buy AAPL 200 --stop 145 --target 165
  etoro-cli buy AAPL 200 --limit 150.00
  etoro-cli sell AAPL
  etoro-cli close 12345678
  etoro-cli cancel 87654321
  etoro-cli --live buy AAPL 200
  etoro-cli search "tesla"
  etoro-cli watchlist create "Tech Stocks"
  etoro-cli alert create AAPL --above 180 --below 140
`

func main() {
	paperFlag := flag.Bool("paper", false, "Demo/paper mode (default when neither flag set)")
	liveFlag := flag.Bool("live", false, "Live trading — real orders")
	jsonFlag := flag.Bool("json", false, "Output as JSON")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	// Default to paper unless --live is explicitly set.
	paper := !*liveFlag || *paperFlag
	if paper {
		tlog.Info("demo mode — no real orders will be sent (use --live for live trading)")
	}

	c, err := client.New(paper)
	if err != nil {
		tlog.Error("connect: %v", err)
		os.Exit(1)
	}

	mode := "DEMO"
	if !paper {
		mode = "LIVE"
	}
	tlog.Info("connected (mode: %s)", mode)

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "positions":
		cmdPositions(c, *jsonFlag)
	case "account":
		cmdAccount(c, rest, *jsonFlag)
	case "quote":
		requireArgs(rest, 1, "quote <SYMBOL>")
		cmdQuote(c, rest[0], *jsonFlag)
	case "orders":
		cmdOrders(c, *jsonFlag)
	case "search":
		requireArgs(rest, 1, "search <QUERY>")
		cmdSearch(c, strings.Join(rest, " "), *jsonFlag)
	case "cancel":
		requireArgs(rest, 1, "cancel <ORDER_ID>")
		cmdCancel(c, rest[0], *jsonFlag, *liveFlag)
	case "modify":
		requireArgs(rest, 1, "modify <ORDER_ID|POSITION_ID> [--stop P] [--target P] [--limit P]")
		cmdModify(c, rest, *jsonFlag, *liveFlag)
	case "buy":
		requireArgs(rest, 2, "buy <SYMBOL> <AMOUNT>")
		cmdBuy(c, rest, *jsonFlag, *liveFlag)
	case "sell":
		requireArgs(rest, 1, "sell <SYMBOL> [AMOUNT]")
		cmdSell(c, rest, *jsonFlag, *liveFlag)
	case "close":
		requireArgs(rest, 1, "close <POSITION_ID>")
		cmdClose(c, rest[0], *jsonFlag, *liveFlag)
	case "stop":
		requireArgs(rest, 1, "stop <POSITION_ID>")
		cmdStop(c, rest, *jsonFlag, *liveFlag)
	case "target":
		requireArgs(rest, 1, "target <POSITION_ID>")
		cmdTarget(c, rest, *jsonFlag, *liveFlag)
	case "watchlist":
		cmdWatchlist(c, rest, *jsonFlag)
	case "alert":
		cmdAlert(c, rest, *jsonFlag)
	default:
		fatalf("unknown command %q\n\n%s", cmd, usage)
	}
}

// ── Read commands ─────────────────────────────────────────────────────────────

func cmdPositions(c *client.EtoroClient, asJSON bool) {
	positions, err := ops.GetPositions(c)
	check(err)
	if asJSON {
		printJSON(positions)
		return
	}
	if len(positions) == 0 {
		fmt.Println("No open positions.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "POS ID\tSYMBOL\tSIDE\tUNITS\tAMOUNT\tOPEN\tCURRENT\tSL\tTP\tP&L\tP&L%")
	fmt.Fprintln(w, "------\t------\t----\t-----\t------\t----\t-------\t--\t--\t---\t----")
	for _, p := range positions {
		side := "BUY"
		if !p.IsBuy {
			side = "SELL"
		}
		sl, tp := "-", "-"
		if p.StopLoss > 0 {
			sl = fmt.Sprintf("$%.2f", p.StopLoss)
		}
		if p.TakeProfit > 0 {
			tp = fmt.Sprintf("$%.2f", p.TakeProfit)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%.2f\t$%.2f\t$%.4f\t$%.4f\t%s\t%s\t$%.2f\t%.1f%%\n",
			p.PositionID, p.Symbol, side, p.Units, p.Amount,
			p.OpenRate, p.CurrentRate, sl, tp, p.PnL, p.PnLPct)
	}
	w.Flush()
}

func cmdAccount(c *client.EtoroClient, args []string, asJSON bool) {
	fs := flag.NewFlagSet("account", flag.ExitOnError)
	accType := fs.String("type", "", "Account type (Trading, Crypto, etc.)")
	history := fs.Bool("history", false, "Show balance history")
	currency := fs.String("currency", "USD", "Display currency")
	fs.Parse(args)

	if *history {
		hist, err := ops.GetAccountHistory(c, "", "", *currency)
		check(err)
		if asJSON {
			printJSON(hist)
			return
		}
		if len(hist) == 0 {
			fmt.Println("No balance history.")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DATE\tEQUITY\tCASH")
		fmt.Fprintln(w, "----\t------\t----")
		for _, h := range hist {
			fmt.Fprintf(w, "%s\t$%.2f\t$%.2f\n", h.Date, h.Equity, h.Cash)
		}
		w.Flush()
		return
	}

	var acct ops.Account
	var err error
	if *accType != "" {
		acct, err = ops.GetAccountByType(c, *accType, *currency)
	} else {
		acct, err = ops.GetAccount(c, *currency)
	}
	check(err)

	if asJSON {
		printJSON(acct)
		return
	}
	fmt.Printf("Equity:           $%.2f\n", acct.Equity)
	fmt.Printf("Cash:             $%.2f\n", acct.Cash)
	fmt.Printf("Total Invested:   $%.2f\n", acct.TotalInvested)
	fmt.Printf("Total P&L:        $%.2f\n", acct.TotalPnL)
	fmt.Printf("Available:        $%.2f\n", acct.AvailableBalance)
}

func cmdQuote(c *client.EtoroClient, symbol string, asJSON bool) {
	q, err := ops.GetQuote(c, strings.ToUpper(symbol))
	check(err)
	if asJSON {
		printJSON(q)
		return
	}
	sign := "+"
	if q.ChangePct < 0 {
		sign = ""
	}
	fmt.Printf("%s  $%.4f  (%s%.2f%%)\n", q.Symbol, q.Price, sign, q.ChangePct)
	fmt.Printf("  O: $%.4f  H: $%.4f  L: $%.4f  Vol: %.0f\n", q.Open, q.High, q.Low, q.Volume)
	fmt.Printf("  Prev close: $%.4f\n", q.PrevClose)
}

func cmdOrders(c *client.EtoroClient, asJSON bool) {
	orders, err := ops.GetOrders(c)
	check(err)
	if asJSON {
		printJSON(orders)
		return
	}
	if len(orders) == 0 {
		fmt.Println("No pending orders.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ORDER ID\tSYMBOL\tSIDE\tAMOUNT\tUNITS\tRATE\tSL\tTP\tSTATUS")
	fmt.Fprintln(w, "--------\t------\t----\t------\t-----\t----\t--\t--\t------")
	for _, o := range orders {
		side := "BUY"
		if !o.IsBuy {
			side = "SELL"
		}
		sl, tp := "-", "-"
		if o.StopLoss > 0 {
			sl = fmt.Sprintf("$%.2f", o.StopLoss)
		}
		if o.TakeProfit > 0 {
			tp = fmt.Sprintf("$%.2f", o.TakeProfit)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t$%.2f\t%.2f\t$%.4f\t%s\t%s\t%s\n",
			o.OrderID, o.Symbol, side, o.Amount, o.Units, o.Rate, sl, tp, o.Status)
	}
	w.Flush()
}

func cmdSearch(c *client.EtoroClient, query string, asJSON bool) {
	results, err := ops.SearchInstruments(c, query)
	check(err)
	if asJSON {
		printJSON(results)
		return
	}
	if len(results) == 0 {
		fmt.Println("No instruments found.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tSYMBOL\tNAME\tEXCHANGE\tACTIVE")
	fmt.Fprintln(w, "--\t------\t----\t--------\t------")
	for _, inst := range results {
		active := "yes"
		if !inst.IsActive {
			active = "no"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\n",
			inst.ID, inst.Symbol, inst.Name, inst.Exchange, active)
	}
	w.Flush()
}

// ── Write commands ─────────────────────────────────────────────────────────────

func cmdBuy(c *client.EtoroClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("buy", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "Limit price")
	stopPrice := fs.Float64("stop", 0, "Stop-loss price")
	targetPrice := fs.Float64("target", 0, "Take-profit price")
	fs.Parse(args[2:])

	symbol := strings.ToUpper(args[0])
	amount := parseFloat(args[1], "amount")

	if *limitPrice > 0 || *stopPrice > 0 || *targetPrice > 0 {
		confirmLive(live, "BUY %s $%.2f limit=$%.2f stop=$%.2f target=$%.2f",
			symbol, amount, *limitPrice, *stopPrice, *targetPrice)
		res, err := ops.BuyWithStops(c, symbol, amount, *limitPrice, *stopPrice, *targetPrice)
		check(err)
		printResult(res, asJSON)
	} else {
		confirmLive(live, "BUY %s $%.2f @ MARKET", symbol, amount)
		res, err := ops.BuyMarket(c, symbol, amount)
		check(err)
		printResult(res, asJSON)
	}
}

func cmdSell(c *client.EtoroClient, args []string, asJSON, live bool) {
	symbol := strings.ToUpper(args[0])
	var amount float64
	if len(args) > 1 {
		amount = parseFloat(args[1], "amount")
	}

	confirmLive(live, "SELL %s (amount: $%.2f)", symbol, amount)
	results, err := ops.SellBySymbol(c, symbol, amount)
	check(err)
	if asJSON {
		printJSON(results)
		return
	}
	for _, r := range results {
		fmt.Printf("[%s] Closed position %s (%s): %s\n", r.Mode, r.PositionID, r.Symbol, r.Status)
	}
}

func cmdClose(c *client.EtoroClient, positionID string, asJSON, live bool) {
	confirmLive(live, "CLOSE position %s", positionID)
	res, err := ops.ClosePosition(c, positionID)
	check(err)
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("[%s] Position %s: %s\n", res.Mode, res.PositionID, res.Status)
}

func cmdCancel(c *client.EtoroClient, orderID string, asJSON, live bool) {
	confirmLive(live, "Cancel order %s", orderID)
	res, err := ops.CancelOrder(c, orderID)
	check(err)
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("[%s] Order %s: %s\n", res.Mode, res.PositionID, res.Status)
}

func cmdModify(c *client.EtoroClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("modify", flag.ExitOnError)
	stopPrice := fs.Float64("stop", 0, "New stop-loss price")
	targetPrice := fs.Float64("target", 0, "New take-profit price")
	limitPrice := fs.Float64("limit", 0, "New limit price (orders only)")
	fs.Parse(args[1:])

	id := args[0]
	if *stopPrice == 0 && *targetPrice == 0 && *limitPrice == 0 {
		fatalf("modify requires at least one of: --stop, --target, --limit")
	}

	confirmLive(live, "MODIFY %s stop=$%.2f target=$%.2f limit=$%.2f", id, *stopPrice, *targetPrice, *limitPrice)

	// Try as order first (if --limit is set), then as position.
	if *limitPrice > 0 {
		err := ops.ModifyOrder(c, id, *stopPrice, *targetPrice, *limitPrice)
		check(err)
	} else {
		err := ops.ModifyPosition(c, id, *stopPrice, *targetPrice)
		check(err)
	}

	if asJSON {
		printJSON(map[string]string{"id": id, "status": "modified"})
	} else {
		fmt.Printf("Modified %s\n", id)
	}
}

func cmdStop(c *client.EtoroClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	price := fs.Float64("price", 0, "Stop trigger price (required)")
	fs.Parse(args[1:])
	if *price == 0 {
		fatalf("--price is required for stop command")
	}

	posID := args[0]
	confirmLive(live, "SET STOP-LOSS on position %s @ $%.4f", posID, *price)
	err := ops.ModifyPosition(c, posID, *price, 0)
	check(err)
	if asJSON {
		printJSON(map[string]string{"position_id": posID, "stop_loss": fmt.Sprintf("%.4f", *price)})
	} else {
		fmt.Printf("Stop-loss set on position %s @ $%.4f\n", posID, *price)
	}
}

func cmdTarget(c *client.EtoroClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("target", flag.ExitOnError)
	price := fs.Float64("price", 0, "Take-profit price (required)")
	fs.Parse(args[1:])
	if *price == 0 {
		fatalf("--price is required for target command")
	}

	posID := args[0]
	confirmLive(live, "SET TAKE-PROFIT on position %s @ $%.4f", posID, *price)
	err := ops.ModifyPosition(c, posID, 0, *price)
	check(err)
	if asJSON {
		printJSON(map[string]string{"position_id": posID, "take_profit": fmt.Sprintf("%.4f", *price)})
	} else {
		fmt.Printf("Take-profit set on position %s @ $%.4f\n", posID, *price)
	}
}

// ── Watchlist commands ───────────────────────────────────────────────────────

func cmdWatchlist(c *client.EtoroClient, args []string, asJSON bool) {
	if len(args) == 0 {
		// List watchlists.
		lists, err := ops.GetWatchlists(c)
		check(err)
		if asJSON {
			printJSON(lists)
			return
		}
		if len(lists) == 0 {
			fmt.Println("No watchlists.")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tNAME\tINSTRUMENTS")
		fmt.Fprintln(w, "--\t----\t-----------")
		for _, wl := range lists {
			fmt.Fprintf(w, "%s\t%s\t%d\n", wl.ID, wl.Name, len(wl.Instruments))
		}
		w.Flush()
		return
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "create":
		requireArgs(rest, 1, "watchlist create <NAME>")
		wl, err := ops.CreateWatchlist(c, strings.Join(rest, " "))
		check(err)
		if asJSON {
			printJSON(wl)
		} else {
			fmt.Printf("Created watchlist %q (ID: %s)\n", wl.Name, wl.ID)
		}

	case "add":
		requireArgs(rest, 2, "watchlist add <ID> <SYMBOLS...>")
		wlID := rest[0]
		symbols := rest[1:]
		ids := make([]int, 0, len(symbols))
		for _, sym := range symbols {
			inst, err := ops.ResolveInstrument(c, sym)
			check(err)
			ids = append(ids, inst.ID)
		}
		err := ops.AddToWatchlist(c, wlID, ids)
		check(err)
		if !asJSON {
			fmt.Printf("Added %d instruments to watchlist %s\n", len(ids), wlID)
		}

	case "delete":
		requireArgs(rest, 1, "watchlist delete <ID>")
		err := ops.DeleteWatchlist(c, rest[0])
		check(err)
		if !asJSON {
			fmt.Printf("Deleted watchlist %s\n", rest[0])
		}

	default:
		fatalf("unknown watchlist command %q — use: create, add, delete", sub)
	}
}

// ── Alert commands ──────────────────────────────────────────────────────────

func cmdAlert(c *client.EtoroClient, args []string, asJSON bool) {
	if len(args) == 0 {
		fatalf("alert requires a subcommand: list, create, delete")
	}

	sub := args[0]
	rest := args[1:]
	switch sub {
	case "list":
		alerts, err := ops.GetAlerts(c)
		check(err)
		if asJSON {
			printJSON(alerts)
			return
		}
		if len(alerts) == 0 {
			fmt.Println("No active alerts.")
			return
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tSYMBOL\tPRICE\tDIRECTION\tACTIVE")
		fmt.Fprintln(w, "--\t------\t-----\t---------\t------")
		for _, a := range alerts {
			fmt.Fprintf(w, "%s\t%s\t$%.2f\t%s\t%v\n",
				a.ID, a.Symbol, a.Price, a.Direction, a.Active)
		}
		w.Flush()

	case "create":
		requireArgs(rest, 1, "alert create <SYMBOL> [--above P] [--below P]")
		fs := flag.NewFlagSet("alert-create", flag.ExitOnError)
		above := fs.Float64("above", 0, "Alert when price goes above")
		below := fs.Float64("below", 0, "Alert when price goes below")
		fs.Parse(rest[1:])

		symbol := strings.ToUpper(rest[0])
		if *above > 0 {
			a, err := ops.CreateAlert(c, symbol, *above, "above")
			check(err)
			if asJSON {
				printJSON(a)
			} else {
				fmt.Printf("Alert created: %s above $%.2f (ID: %s)\n", symbol, *above, a.ID)
			}
		}
		if *below > 0 {
			a, err := ops.CreateAlert(c, symbol, *below, "below")
			check(err)
			if asJSON {
				printJSON(a)
			} else {
				fmt.Printf("Alert created: %s below $%.2f (ID: %s)\n", symbol, *below, a.ID)
			}
		}
		if *above == 0 && *below == 0 {
			fatalf("alert create requires --above and/or --below")
		}

	case "delete":
		requireArgs(rest, 1, "alert delete <ID>")
		err := ops.DeleteAlert(c, rest[0])
		check(err)
		if !asJSON {
			fmt.Printf("Deleted alert %s\n", rest[0])
		}

	default:
		fatalf("unknown alert command %q — use: list, create, delete", sub)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printResult(r ops.OrderResult, asJSON bool) {
	if asJSON {
		printJSON(r)
		return
	}
	if r.Price > 0 {
		fmt.Printf("[%s] %s %s %s $%d @ $%.4f → order %s\n",
			r.Mode, r.Action, r.Symbol, r.Type, r.Qty, r.Price, r.OrderID)
	} else {
		fmt.Printf("[%s] %s %s %s $%d → order %s\n",
			r.Mode, r.Action, r.Symbol, r.Type, r.Qty, r.OrderID)
	}
}

func printJSON(v interface{}) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(v)
}

func confirmLive(live bool, format string, args ...interface{}) {
	if !live {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Printf("\nLIVE ORDER: %s\nExecute? [y/N] ", msg)
	reader := bufio.NewReader(os.Stdin)
	resp, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(resp)) != "y" {
		fmt.Println("Aborted.")
		os.Exit(0)
	}
}

func requireArgs(args []string, n int, usage string) {
	if len(args) < n {
		fatalf("usage: etoro-cli %s", usage)
	}
}

func parseFloat(s, name string) float64 {
	n, err := strconv.ParseFloat(s, 64)
	if err != nil || n <= 0 {
		fatalf("%s must be a positive number, got %q", name, s)
	}
	return n
}

func check(err error) {
	if err != nil {
		tlog.Error("%v", err)
		os.Exit(1)
	}
}

func fatalf(format string, args ...interface{}) {
	tlog.Error(format, args...)
	os.Exit(1)
}
