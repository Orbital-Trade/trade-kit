// tiger-cli — execute Tiger Brokers trades from the command line.
//
// Each subcommand maps 1:1 to a function in ops/ — the same functions
// that will become MCP tools. This file is a thin dispatcher only.
//
// Usage:
//
//	tiger-cli [--paper|--live] [--json] <command> [args]
//
// Build:
//
//	cd tools/tiger && go build -o tiger-cli ./cmd/
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

	"tiger-cli/client"
	"tiger-cli/internal/tlog"
	"tiger-cli/ops"
)

// Compile-time proof that *client.TigerClient satisfies ops.Caller.
// If TigerClient ever drops a required method this line will fail to build.
var _ ops.Caller = (*client.TigerClient)(nil)

const usage = `tiger-cli — Tiger Brokers trade execution CLI

Usage:
  tiger-cli [--paper|--live] [--json] <command> [args]

Global flags:
  --paper   Paper/simulation mode (default — no real orders sent)
  --live    Live trading mode (requires Y confirmation for write ops)
  --json    Output as JSON (for scripting or MCP piping)

Read commands (safe — no confirmation needed):
  positions                         List open stock positions
  account                           Account cash, buying power, net liquidation
  quote  <SYMBOL>                   Real-time price snapshot
  orders                            List open/pending orders
  analyze <SYMBOL>                  Multi-timeframe technical analysis (RSI/MACD/BB/EMA)
           --futures                Treat SYMBOL as a futures root (MNQ, MES, ES …)

Write commands (paper: log only  |  live: confirm then execute):
  buy    <SYMBOL> <SHARES>          Market buy
         --limit  <price>           → Limit buy instead
         --stop   <price>           Also place stop-loss after buy
         --target <price>           Also place take-profit limit after buy

  sell   <SYMBOL> <SHARES>          Market sell
         --limit  <price>           → Limit sell instead (GTC)

  stop   <SYMBOL> <SHARES>          Set stop-loss on existing position
         --price  <price>           Stop trigger price (required)

  target <SYMBOL> <SHARES>          Set take-profit limit on existing position
         --price  <price>           Limit price (required)

  cancel <ORDER_ID>                 Cancel an open order
  modify <ORDER_ID>                 Modify an open order (in-place, no cancel+replace)
         --limit  <price>           New limit price
         --stop   <price>           New stop/aux price
         --qty    <n>               New quantity
         --tif    DAY|GTC           New time-in-force

Futures commands:
  futures entry <SYM> <long|short> <N>
          --entry <price>           Entry limit price (required)
          --stop  <price>           Protective stop price (required)

  futures close <SYM> <long|short> <N>
                                    Market close (opposite side)

  futures update-stop <SYM> <long|short> <N>
          --stop   <price>          New stop price (required)
          --old-id <id>             Previous stop order ID to cancel (required)

Examples:
  tiger-cli positions
  tiger-cli quote NOK
  tiger-cli buy NOK 100 --limit 4.50 --stop 4.20 --target 5.00
  tiger-cli sell NOK 50
  tiger-cli sell NOK 50 --limit 4.80
  tiger-cli stop NOK 100 --price 4.20
  tiger-cli target NOK 100 --price 5.00
  tiger-cli cancel 123456789
  tiger-cli modify 123456789 --limit 4.60
  tiger-cli modify 123456789 --stop 4.10
  tiger-cli modify 123456789 --qty 50 --tif GTC
  tiger-cli --live buy NOK 100 --limit 4.50
  tiger-cli futures entry MES long 1 --entry 5100 --stop 5090
  tiger-cli futures close MES long 1
  tiger-cli futures update-stop MES long 1 --stop 5095 --old-id 789012345
  tiger-cli analyze AAPL
  tiger-cli analyze ES3.SI
  tiger-cli analyze MNQ --futures
  tiger-cli --json analyze AAPL
`

func main() {
	paperFlag := flag.Bool("paper", false, "Paper/simulation mode (default when neither flag set)")
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
		tlog.Info("paper mode — no real orders will be sent (use --live for live trading)")
	}

	c, err := client.New(paper)
	if err != nil {
		tlog.Error("connect: %v", err)
		os.Exit(1)
	}
	tlog.Info("connected (account: %s, mode: %s)", c.Account(), map[bool]string{true: "PAPER", false: "LIVE"}[paper])

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "positions":
		cmdPositions(c, *jsonFlag)
	case "account":
		cmdAccount(c, *jsonFlag)
	case "quote":
		requireArgs(rest, 1, "quote <SYMBOL>")
		cmdQuote(c, rest[0], *jsonFlag)
	case "orders":
		cmdOrders(c, *jsonFlag)
	case "cancel":
		requireArgs(rest, 1, "cancel <ORDER_ID>")
		cmdCancel(c, rest[0], *jsonFlag, *liveFlag)
	case "modify":
		requireArgs(rest, 1, "modify <ORDER_ID> [--limit P] [--stop P] [--qty N] [--tif DAY|GTC]")
		cmdModify(c, rest, *jsonFlag, *liveFlag)
	case "buy":
		requireArgs(rest, 2, "buy <SYMBOL> <SHARES>")
		cmdBuy(c, rest, *jsonFlag, *liveFlag)
	case "sell":
		requireArgs(rest, 2, "sell <SYMBOL> <SHARES>")
		cmdSell(c, rest, *jsonFlag, *liveFlag)
	case "stop":
		requireArgs(rest, 2, "stop <SYMBOL> <SHARES>")
		cmdStop(c, rest, *jsonFlag, *liveFlag)
	case "target":
		requireArgs(rest, 2, "target <SYMBOL> <SHARES>")
		cmdTarget(c, rest, *jsonFlag, *liveFlag)
	case "futures":
		cmdFutures(c, rest, *jsonFlag, *liveFlag)
	case "analyze":
		requireArgs(rest, 1, "analyze <SYMBOL> [--futures]")
		cmdAnalyze(c, rest, *jsonFlag)
	default:
		fatalf("unknown command %q\n\n%s", cmd, usage)
	}
}

// ── Read commands ─────────────────────────────────────────────────────────────

func cmdPositions(c *client.TigerClient, asJSON bool) {
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
	fmt.Fprintln(w, "SYMBOL\tSHARES\tAVG COST\tMKT PRICE\tMKT VALUE\tUNREAL P&L\tREAL P&L")
	fmt.Fprintln(w, "------\t------\t--------\t---------\t---------\t----------\t--------")
	for _, p := range positions {
		fmt.Fprintf(w, "%s\t%d\t$%.4f\t$%.4f\t$%.2f\t$%.2f\t$%.2f\n",
			p.Symbol, p.Shares, p.AvgCost, p.MarketPrice, p.MarketValue, p.UnrealizedPnL, p.RealizedPnL)
	}
	w.Flush()
}

func cmdAccount(c *client.TigerClient, asJSON bool) {
	acct, err := ops.GetAccount(c)
	check(err)
	if asJSON {
		printJSON(acct)
		return
	}
	fmt.Printf("Net Liquidation:  $%.2f\n", acct.NetLiquidation)
	fmt.Printf("Cash:             $%.2f\n", acct.Cash)
	fmt.Printf("Buying Power:     $%.2f\n", acct.BuyingPower)
	fmt.Printf("Gross Positions:  $%.2f\n", acct.GrossPositionValue)
}

func cmdQuote(c *client.TigerClient, symbol string, asJSON bool) {
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
	cur := ops.SymbolInfo{Currency: q.Currency}
	cs := cur.CurrencySign()
	fmt.Printf("%s  %s%.4f  (%s%.2f%%)\n", q.Symbol, cs, q.Price, sign, q.ChangePct)
	fmt.Printf("  O: %s%.4f  H: %s%.4f  L: %s%.4f  Vol: %.0f\n", cs, q.Open, cs, q.High, cs, q.Low, q.Volume)
	fmt.Printf("  Prev close: %s%.4f\n", cs, q.PrevClose)
}

func cmdOrders(c *client.TigerClient, asJSON bool) {
	orders, err := ops.GetOrders(c)
	check(err)
	if asJSON {
		printJSON(orders)
		return
	}
	if len(orders) == 0 {
		fmt.Println("No open orders.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ORDER ID\tSYMBOL\tACTION\tTYPE\tQTY\tFILLED\tLIMIT\tSTOP\tTIF\tSTATUS")
	fmt.Fprintln(w, "--------\t------\t------\t----\t---\t------\t-----\t----\t---\t------")
	for _, o := range orders {
		lim, stp := "-", "-"
		if o.LimitPrice > 0 {
			lim = fmt.Sprintf("$%.4f", o.LimitPrice)
		}
		if o.StopPrice > 0 {
			stp = fmt.Sprintf("$%.4f", o.StopPrice)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%s\n",
			o.ID, o.Symbol, o.Action, o.OrderType, o.Quantity, o.FilledQty, lim, stp, o.TimeInForce, o.Status)
	}
	w.Flush()
}

func cmdAnalyze(c *client.TigerClient, args []string, asJSON bool) {
	fs := flag.NewFlagSet("analyze", flag.ExitOnError)
	isFutures := fs.Bool("futures", false, "Treat symbol as futures root")
	fs.Parse(args[1:])

	symbol := strings.ToUpper(args[0])
	result, err := ops.Analyze(c, symbol, *isFutures)
	check(err)

	if asJSON {
		printJSON(result)
		return
	}
	printAnalysis(result)
}

func printAnalysis(r ops.AnalyzeResult) {
	cs := "$"
	if r.Currency == "SGD" {
		cs = "S$"
	}

	header := fmt.Sprintf("%s — Multi-Timeframe Analysis  %s", r.Symbol, r.Timestamp)
	if r.IsFutures {
		header = fmt.Sprintf("%s (%s) — Multi-Timeframe Analysis  %s", r.Symbol, r.Contract, r.Timestamp)
	}
	fmt.Println(header)
	fmt.Println(strings.Repeat("─", len(header)))

	biasIcon := map[string]string{"BULLISH": "▲", "BEARISH": "▼", "NEUTRAL": "◆", "MIXED": "◈"}

	for _, tf := range r.Timeframes {
		fmt.Printf("\n%s  (%d bars)   Price: %s%.4f\n", tf.Timeframe, tf.Bars, cs, tf.Price)

		switch tf.Bias {
		case "INSUFFICIENT_DATA":
			fmt.Println("  ⚠ Not enough bars to compute indicators")
			continue
		case "ERROR":
			fmt.Printf("  ✗ Failed to fetch data: %s\n", tf.Err)
			continue
		}

		rsiTag := "Neutral"
		if tf.RSI > 70 {
			rsiTag = "Overbought"
		} else if tf.RSI > 55 {
			rsiTag = "Bullish"
		} else if tf.RSI < 30 {
			rsiTag = "Oversold"
		} else if tf.RSI < 45 {
			rsiTag = "Bearish"
		}
		fmt.Printf("  RSI(14):  %.1f  [%s]\n", tf.RSI, rsiTag)

		histSign := "+"
		if tf.MACDHist < 0 {
			histSign = ""
		}
		fmt.Printf("  MACD:     Line: %.4f  Sig: %.4f  Hist: %s%.4f\n",
			tf.MACDLine, tf.MACDSignal, histSign, tf.MACDHist)

		fmt.Printf("  BB(20):   %%B: %.2f  Upper: %s%.2f  Mid: %s%.2f  Lower: %s%.2f\n",
			tf.BBPct, cs, tf.BBUpper, cs, tf.BBMiddle, cs, tf.BBLower)

		emaStr := func(v float64) string {
			if v == 0 {
				return "N/A"
			}
			rel := "above"
			if tf.Price < v {
				rel = "below"
			}
			return fmt.Sprintf("%s%.2f (%s)", cs, v, rel)
		}
		fmt.Printf("  EMA:      20: %s  50: %s  200: %s\n",
			emaStr(tf.EMA20), emaStr(tf.EMA50), emaStr(tf.EMA200))

		icon := biasIcon[tf.Bias]
		if icon == "" {
			icon = "?"
		}
		fmt.Printf("  Bias:     %s %s  (score %+d/5)\n", tf.Bias, icon, tf.Score)
	}

	fmt.Println()
	sep := strings.Repeat("═", 50)
	fmt.Println(sep)
	icon := biasIcon[r.Alignment]
	if icon == "" {
		icon = "?"
	}
	fmt.Printf("ALIGNMENT:  %s %s  (%dB / %dBr / %dN)\n",
		r.Alignment, icon, r.BullCount, r.BearCount,
		len(r.Timeframes)-r.BullCount-r.BearCount)
	fmt.Println(sep)
}

// ── Write commands ─────────────────────────────────────────────────────────────

func cmdCancel(c *client.TigerClient, orderID string, asJSON, live bool) {
	confirmLive(live, "Cancel order %s", orderID)
	res, err := ops.CancelOrder(c, orderID)
	check(err)
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("Order %s: %s\n", res.OrderID, res.Status)
}

func cmdModify(c *client.TigerClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("modify", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "New limit price")
	stopPrice := fs.Float64("stop", 0, "New stop/aux price")
	qty := fs.Int("qty", 0, "New quantity")
	tif := fs.String("tif", "", "New time-in-force: DAY or GTC")
	fs.Parse(args[1:])

	orderID := args[0]
	if *limitPrice == 0 && *stopPrice == 0 && *qty == 0 && *tif == "" {
		fatalf("modify requires at least one of: --limit, --stop, --qty, --tif")
	}

	confirmLive(live, "MODIFY order %s  limit=%v stop=%v qty=%v tif=%v",
		orderID, *limitPrice, *stopPrice, *qty, *tif)

	res, err := ops.ModifyOrder(c, orderID, ops.ModifyParams{
		LimitPrice:  *limitPrice,
		StopPrice:   *stopPrice,
		Quantity:    *qty,
		TimeInForce: *tif,
	})
	check(err)
	if asJSON {
		printJSON(res)
		return
	}
	fmt.Printf("[%s] Order %s: %s\n", res.Mode, res.OrderID, res.Status)
}

func cmdBuy(c *client.TigerClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("buy", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "Limit price")
	stopPrice := fs.Float64("stop", 0, "Stop-loss price to set after buy")
	targetPrice := fs.Float64("target", 0, "Take-profit price to set after buy")
	fs.Parse(args[2:])

	symbol := strings.ToUpper(args[0])
	shares := parseInt(args[1], "shares")

	if *limitPrice > 0 {
		confirmLive(live, "BUY %d %s @ LIMIT $%.4f", shares, symbol, *limitPrice)
		res, err := ops.BuyLimit(c, symbol, shares, *limitPrice)
		check(err)
		printResult(res, asJSON)
	} else {
		confirmLive(live, "BUY %d %s @ MARKET", shares, symbol)
		res, err := ops.BuyMarket(c, symbol, shares)
		check(err)
		printResult(res, asJSON)
	}

	if *stopPrice > 0 {
		res, err := ops.SetStopLoss(c, symbol, shares, *stopPrice)
		check(err)
		if !asJSON {
			fmt.Printf("  + stop-loss → order %s\n", res.OrderID)
		}
	}
	if *targetPrice > 0 {
		res, err := ops.SetTakeProfit(c, symbol, shares, *targetPrice)
		check(err)
		if !asJSON {
			fmt.Printf("  + take-profit → order %s\n", res.OrderID)
		}
	}
}

func cmdSell(c *client.TigerClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("sell", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "Limit price (GTC)")
	fs.Parse(args[2:])

	symbol := strings.ToUpper(args[0])
	shares := parseInt(args[1], "shares")

	if *limitPrice > 0 {
		confirmLive(live, "SELL %d %s @ LIMIT $%.4f (GTC)", shares, symbol, *limitPrice)
		res, err := ops.SellLimit(c, symbol, shares, *limitPrice)
		check(err)
		printResult(res, asJSON)
	} else {
		confirmLive(live, "SELL %d %s @ MARKET", shares, symbol)
		res, err := ops.SellMarket(c, symbol, shares)
		check(err)
		printResult(res, asJSON)
	}
}

func cmdStop(c *client.TigerClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("stop", flag.ExitOnError)
	price := fs.Float64("price", 0, "Stop trigger price (required)")
	fs.Parse(args[2:])
	if *price == 0 {
		fatalf("--price is required for stop command")
	}

	symbol := strings.ToUpper(args[0])
	shares := parseInt(args[1], "shares")

	confirmLive(live, "SET STOP-LOSS %d %s @ $%.4f (GTC, stop-market)", shares, symbol, *price)
	res, err := ops.SetStopLoss(c, symbol, shares, *price)
	check(err)
	printResult(res, asJSON)
}

func cmdTarget(c *client.TigerClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("target", flag.ExitOnError)
	price := fs.Float64("price", 0, "Take-profit limit price (required)")
	fs.Parse(args[2:])
	if *price == 0 {
		fatalf("--price is required for target command")
	}

	symbol := strings.ToUpper(args[0])
	shares := parseInt(args[1], "shares")

	confirmLive(live, "SET TAKE-PROFIT %d %s @ $%.4f (GTC, limit)", shares, symbol, *price)
	res, err := ops.SetTakeProfit(c, symbol, shares, *price)
	check(err)
	printResult(res, asJSON)
}

func cmdFutures(c *client.TigerClient, args []string, asJSON, live bool) {
	if len(args) == 0 {
		fatalf("futures requires a subcommand: entry, close, update-stop")
	}
	sub := args[0]
	rest := args[1:]

	switch sub {
	case "entry":
		requireArgs(rest, 3, "futures entry <SYM> <long|short> <N>")
		fs := flag.NewFlagSet("futures-entry", flag.ExitOnError)
		entryPrice := fs.Float64("entry", 0, "Entry limit price (required)")
		stopPrice := fs.Float64("stop", 0, "Stop-loss price (required)")
		fs.Parse(rest[3:])
		if *entryPrice == 0 || *stopPrice == 0 {
			fatalf("--entry and --stop are required")
		}
		sym := strings.ToUpper(rest[0])
		dir := strings.ToUpper(rest[1])
		n := parseInt(rest[2], "contracts")
		confirmLive(live, "FUTURES ENTRY %s %s %d @ $%.2f  STOP @ $%.2f", dir, sym, n, *entryPrice, *stopPrice)
		res, err := ops.FuturesEntry(c, sym, dir, n, *entryPrice, *stopPrice)
		check(err)
		if asJSON {
			printJSON(res)
		} else {
			fmt.Printf("Entry order: %s\nStop  order: %s\nMode: %s\n", res.EntryOrderID, res.StopOrderID, res.Mode)
		}

	case "close":
		requireArgs(rest, 3, "futures close <SYM> <long|short> <N>")
		sym := strings.ToUpper(rest[0])
		dir := strings.ToUpper(rest[1])
		n := parseInt(rest[2], "contracts")
		confirmLive(live, "FUTURES CLOSE %s %s %d @ MARKET", dir, sym, n)
		res, err := ops.FuturesClose(c, sym, dir, n)
		check(err)
		printResult(res, asJSON)

	case "update-stop":
		requireArgs(rest, 3, "futures update-stop <SYM> <long|short> <N>")
		fs := flag.NewFlagSet("futures-update-stop", flag.ExitOnError)
		newStop := fs.Float64("stop", 0, "New stop price (required)")
		oldID := fs.String("old-id", "", "Previous stop order ID to cancel (required)")
		fs.Parse(rest[3:])
		if *newStop == 0 || *oldID == "" {
			fatalf("--stop and --old-id are required")
		}
		sym := strings.ToUpper(rest[0])
		dir := strings.ToUpper(rest[1])
		n := parseInt(rest[2], "contracts")
		confirmLive(live, "FUTURES UPDATE STOP %s %s → $%.2f (cancel %s)", sym, dir, *newStop, *oldID)
		res, err := ops.FuturesUpdateStop(c, sym, dir, n, *newStop, *oldID)
		check(err)
		printResult(res, asJSON)

	default:
		fatalf("unknown futures subcommand %q — use: entry, close, update-stop", sub)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printResult(r ops.OrderResult, asJSON bool) {
	if asJSON {
		printJSON(r)
		return
	}
	mode := r.Mode
	if r.Price > 0 {
		fmt.Printf("[%s] %s %d %s %s @ $%.4f → order %s\n",
			mode, r.Action, r.Qty, r.Symbol, r.Type, r.Price, r.OrderID)
	} else {
		fmt.Printf("[%s] %s %d %s %s → order %s\n",
			mode, r.Action, r.Qty, r.Symbol, r.Type, r.OrderID)
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
		fatalf("usage: tiger-cli %s", usage)
	}
}

func parseInt(s, name string) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		fatalf("%s must be a positive integer, got %q", name, s)
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
