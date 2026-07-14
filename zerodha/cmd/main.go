// zerodha-cli — execute Zerodha Kite Connect trades from the command line.
//
// Usage:
//
//	zerodha-cli [--paper|--live] [--json] [--intraday] <command> [args]
//
// Build:
//
//	cd zerodha && go build -o zerodha-cli ./cmd/
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

	"zerodha-cli/client"
	"zerodha-cli/internal/tlog"
	"zerodha-cli/ops"
)

var _ ops.Caller = (*client.ZerodhaClient)(nil)

const usage = `zerodha-cli — Zerodha Kite Connect trade execution CLI

Usage:
  zerodha-cli [--paper|--live] [--json] [--intraday] <command> [args]

Global flags:
  --paper     Paper mode (default — logs orders without sending)
  --live      Live trading mode (sends real orders to Kite)
  --json      Output as JSON
  --intraday  Use MIS product (intraday) instead of CNC (delivery)

Read commands:
  positions                         List net positions
  holdings                          List DEMAT holdings
  account                           Account margins (equity segment)
  quote  <SYMBOL>                   Price snapshot (via Yahoo Finance)
  orders                            List all orders for the day

Write commands:
  buy    <SYMBOL> <QTY>             Market buy (CNC)
         --limit  <price>           Limit buy

  sell   <SYMBOL> <QTY>             Market sell (CNC)
         --limit  <price>           Limit sell

  cancel <ORDER_ID>                 Cancel an open order

Symbol format:
  RELIANCE                          NSE (default exchange)
  BSE:RELIANCE                      BSE exchange
  NFO:NIFTY23JUNFUT                 F&O segment

Examples:
  zerodha-cli positions
  zerodha-cli holdings
  zerodha-cli quote RELIANCE
  zerodha-cli buy RELIANCE 10
  zerodha-cli buy TCS 5 --limit 3500
  zerodha-cli --intraday buy INFY 20
  zerodha-cli sell RELIANCE 10
  zerodha-cli cancel 220713000001
  zerodha-cli --live buy RELIANCE 10
`

func main() {
	paperFlag := flag.Bool("paper", false, "Paper mode (default)")
	liveFlag := flag.Bool("live", false, "Live trading")
	jsonFlag := flag.Bool("json", false, "Output as JSON")
	intradayFlag := flag.Bool("intraday", false, "Use MIS product (intraday)")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	paper := !*liveFlag || *paperFlag
	if paper {
		tlog.Info("paper mode — orders will be logged, not sent")
	}

	c, err := client.New(paper)
	if err != nil {
		tlog.Error("connect: %v", err)
		os.Exit(1)
	}

	mode := "PAPER"
	if !paper {
		mode = "LIVE"
	}
	tlog.Info("connected (mode: %s)", mode)

	cmd := args[0]
	rest := args[1:]

	switch cmd {
	case "positions":
		cmdPositions(c, *jsonFlag)
	case "holdings":
		cmdHoldings(c, *jsonFlag)
	case "account":
		cmdAccount(c, *jsonFlag)
	case "quote":
		requireArgs(rest, 1, "quote <SYMBOL>")
		cmdQuote(c, rest[0], *jsonFlag)
	case "orders":
		cmdOrders(c, *jsonFlag)
	case "buy":
		requireArgs(rest, 2, "buy <SYMBOL> <QTY>")
		cmdBuy(c, rest, *jsonFlag, *liveFlag, *intradayFlag)
	case "sell":
		requireArgs(rest, 2, "sell <SYMBOL> <QTY>")
		cmdSell(c, rest, *jsonFlag, *liveFlag)
	case "cancel":
		requireArgs(rest, 1, "cancel <ORDER_ID>")
		cmdCancel(c, rest[0], *jsonFlag, *liveFlag)
	default:
		fatalf("unknown command %q\n\n%s", cmd, usage)
	}
}

func cmdPositions(c *client.ZerodhaClient, asJSON bool) {
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
	fmt.Fprintln(w, "SYMBOL\tEXCH\tQTY\tAVG PRICE\tLTP\tP&L\tPRODUCT")
	fmt.Fprintln(w, "------\t----\t---\t---------\t---\t---\t-------")
	for _, p := range positions {
		fmt.Fprintf(w, "%s\t%s\t%d\t%.2f\t%.2f\t%.2f\t%s\n",
			p.Symbol, p.Exchange, p.Quantity, p.AvgPrice, p.LastPrice, p.PnL, p.Product)
	}
	w.Flush()
}

func cmdHoldings(c *client.ZerodhaClient, asJSON bool) {
	holdings, err := ops.GetHoldings(c)
	check(err)
	if asJSON {
		printJSON(holdings)
		return
	}
	if len(holdings) == 0 {
		fmt.Println("No DEMAT holdings.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SYMBOL\tEXCH\tQTY\tAVG PRICE\tLTP\tP&L\tDAY CHG%")
	fmt.Fprintln(w, "------\t----\t---\t---------\t---\t---\t--------")
	for _, h := range holdings {
		fmt.Fprintf(w, "%s\t%s\t%d\t%.2f\t%.2f\t%.2f\t%.2f%%\n",
			h.Symbol, h.Exchange, h.Quantity, h.AveragePrice, h.LastPrice, h.PnL, h.DayChangePct)
	}
	w.Flush()
}

func cmdAccount(c *client.ZerodhaClient, asJSON bool) {
	acct, err := ops.GetAccount(c)
	check(err)
	if asJSON {
		printJSON(acct)
		return
	}
	fmt.Printf("Cash:          %.2f\n", acct.Cash)
	fmt.Printf("Live Balance:  %.2f\n", acct.LiveBal)
	fmt.Printf("Collateral:    %.2f\n", acct.Collateral)
	fmt.Printf("Debits:        %.2f\n", acct.Debits)
	fmt.Printf("Exposure:      %.2f\n", acct.Exposure)
	fmt.Printf("Net:           %.2f\n", acct.Net)
}

func cmdQuote(c *client.ZerodhaClient, symbol string, asJSON bool) {
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
	fmt.Printf("%s  %.2f  (%s%.2f%%)\n", q.Symbol, q.Price, sign, q.ChangePct)
	fmt.Printf("  O: %.2f  H: %.2f  L: %.2f  Vol: %.0f\n", q.Open, q.High, q.Low, q.Volume)
	fmt.Printf("  Prev close: %.2f\n", q.PrevClose)
}

func cmdOrders(c *client.ZerodhaClient, asJSON bool) {
	orders, err := ops.GetOrders(c)
	check(err)
	if asJSON {
		printJSON(orders)
		return
	}
	if len(orders) == 0 {
		fmt.Println("No orders for the day.")
		return
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ORDER ID\tSYMBOL\tEXCH\tSIDE\tTYPE\tQTY\tFILLED\tPRICE\tPRODUCT\tSTATUS")
	fmt.Fprintln(w, "--------\t------\t----\t----\t----\t---\t------\t-----\t-------\t------")
	for _, o := range orders {
		price := "-"
		if o.Price > 0 {
			price = fmt.Sprintf("%.2f", o.Price)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%d\t%d\t%s\t%s\t%s\n",
			o.OrderID, o.Symbol, o.Exchange, o.TransactionType, o.OrderType,
			o.Quantity, o.FilledQty, price, o.Product, o.Status)
	}
	w.Flush()
}

func cmdBuy(c *client.ZerodhaClient, args []string, asJSON, live, intraday bool) {
	fs := flag.NewFlagSet("buy", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "Limit price")
	fs.Parse(args[2:])

	symbol := strings.ToUpper(args[0])
	qty := parseInt(args[1], "qty")

	if intraday {
		confirmLive(live, "BUY %d %s @ MARKET (MIS intraday)", qty, symbol)
		res, err := ops.BuyIntraday(c, symbol, qty)
		check(err)
		printResult(res, asJSON)
	} else if *limitPrice > 0 {
		confirmLive(live, "BUY %d %s @ LIMIT %.2f", qty, symbol, *limitPrice)
		res, err := ops.BuyLimit(c, symbol, qty, *limitPrice)
		check(err)
		printResult(res, asJSON)
	} else {
		confirmLive(live, "BUY %d %s @ MARKET", qty, symbol)
		res, err := ops.BuyMarket(c, symbol, qty)
		check(err)
		printResult(res, asJSON)
	}
}

func cmdSell(c *client.ZerodhaClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("sell", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "Limit price")
	fs.Parse(args[2:])

	symbol := strings.ToUpper(args[0])
	qty := parseInt(args[1], "qty")

	if *limitPrice > 0 {
		confirmLive(live, "SELL %d %s @ LIMIT %.2f", qty, symbol, *limitPrice)
		res, err := ops.SellLimit(c, symbol, qty, *limitPrice)
		check(err)
		printResult(res, asJSON)
	} else {
		confirmLive(live, "SELL %d %s @ MARKET", qty, symbol)
		res, err := ops.SellMarket(c, symbol, qty)
		check(err)
		printResult(res, asJSON)
	}
}

func cmdCancel(c *client.ZerodhaClient, orderID string, asJSON, live bool) {
	confirmLive(live, "CANCEL order %s", orderID)
	err := ops.CancelOrder(c, orderID)
	check(err)
	if asJSON {
		printJSON(map[string]string{"order_id": orderID, "status": "cancelled"})
	} else {
		fmt.Printf("Cancelled order %s\n", orderID)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func printResult(r ops.OrderResult, asJSON bool) {
	if asJSON {
		printJSON(r)
		return
	}
	if r.Price > 0 {
		fmt.Printf("[%s] %s %d %s %s @ %.2f -> order %s\n",
			r.Mode, r.Action, r.Qty, r.Symbol, r.Type, r.Price, r.OrderID)
	} else {
		fmt.Printf("[%s] %s %d %s %s -> order %s\n",
			r.Mode, r.Action, r.Qty, r.Symbol, r.Type, r.OrderID)
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
		fatalf("usage: zerodha-cli %s", usage)
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
