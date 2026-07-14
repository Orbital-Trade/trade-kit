// ibkr-cli — execute Interactive Brokers trades from the command line.
//
// Usage:
//
//	ibkr-cli [--paper|--live] [--json] <command> [args]
//
// Build:
//
//	cd ibkr && go build -o ibkr-cli ./cmd/
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"ibkr-cli/client"
	"ibkr-cli/internal/tlog"
	"ibkr-cli/ops"
)

var _ ops.Caller = (*client.IBKRClient)(nil)

const usage = `ibkr-cli — Interactive Brokers Client Portal trade execution CLI

Usage:
  ibkr-cli [--paper|--live] [--json] <command> [args]

Global flags:
  --paper   Paper trading mode (default — informational only, gateway determines actual mode)
  --live    Live trading mode
  --json    Output as JSON

Read commands:
  positions                         List open positions
  account                           Account summary (equity, cash, buying power)
  quote  <SYMBOL>                   Real-time price snapshot
  orders                            List open/pending orders

Write commands:
  buy    <SYMBOL> <QTY>             Market buy
         --limit  <price>           Limit buy

  sell   <SYMBOL> <QTY>             Market sell
         --limit  <price>           Limit sell (GTC)

  close  <SYMBOL>                   Close entire position
  cancel <ORDER_ID>                 Cancel an open order

Examples:
  ibkr-cli positions
  ibkr-cli quote AAPL
  ibkr-cli buy AAPL 10
  ibkr-cli buy AAPL 10 --limit 150
  ibkr-cli sell AAPL 5
  ibkr-cli close AAPL
  ibkr-cli --live buy AAPL 10
`

func main() {
	paperFlag := flag.Bool("paper", false, "Paper mode (default)")
	liveFlag := flag.Bool("live", false, "Live trading")
	jsonFlag := flag.Bool("json", false, "Output as JSON")
	flag.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	paper := !*liveFlag || *paperFlag
	if paper {
		tlog.Info("paper mode — gateway determines actual account type")
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
	tlog.Info("connected (mode: %s, account: %s)", mode, c.AccountID())

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
	case "buy":
		requireArgs(rest, 2, "buy <SYMBOL> <QTY>")
		cmdBuy(c, rest, *jsonFlag, *liveFlag)
	case "sell":
		requireArgs(rest, 2, "sell <SYMBOL> <QTY>")
		cmdSell(c, rest, *jsonFlag, *liveFlag)
	case "close":
		requireArgs(rest, 1, "close <SYMBOL>")
		cmdClose(c, rest[0], *jsonFlag, *liveFlag)
	case "cancel":
		requireArgs(rest, 1, "cancel <ORDER_ID>")
		cmdCancel(c, rest[0], *jsonFlag, *liveFlag)
	default:
		fatalf("unknown command %q\n\n%s", cmd, usage)
	}
}

func cmdPositions(c *client.IBKRClient, asJSON bool) {
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
	fmt.Fprintln(w, "SYMBOL\tSIDE\tQTY\tAVG COST\tMKT PRICE\tMKT VALUE\tUNREAL P&L\tREAL P&L")
	fmt.Fprintln(w, "------\t----\t---\t--------\t---------\t---------\t----------\t--------")
	for _, p := range positions {
		fmt.Fprintf(w, "%s\t%s\t%.0f\t$%.4f\t$%.4f\t$%.2f\t$%.2f\t$%.2f\n",
			p.Symbol, strings.ToUpper(p.Side), math.Abs(p.Qty), p.AvgCost, p.MktPrice,
			p.MktValue, p.UnrealizedPL, p.RealizedPL)
	}
	w.Flush()
}

func cmdAccount(c *client.IBKRClient, asJSON bool) {
	acct, err := ops.GetAccount(c)
	check(err)
	if asJSON {
		printJSON(acct)
		return
	}
	fmt.Printf("Account:         %s (%s)\n", acct.ID, acct.Status)
	fmt.Printf("Equity:          $%.2f\n", acct.Equity)
	fmt.Printf("Cash:            $%.2f\n", acct.Cash)
	fmt.Printf("Buying Power:    $%.2f\n", acct.BuyingPower)
	fmt.Printf("Net Liquidation: $%.2f\n", acct.NetLiquidation)
	fmt.Printf("Gross Position:  $%.2f\n", acct.GrossPosition)
}

func cmdQuote(c *client.IBKRClient, symbol string, asJSON bool) {
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

func cmdOrders(c *client.IBKRClient, asJSON bool) {
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
	fmt.Fprintln(w, "ORDER ID\tSYMBOL\tSIDE\tTYPE\tQTY\tFILLED\tPRICE\tTIF\tSTATUS")
	fmt.Fprintln(w, "--------\t------\t----\t----\t---\t------\t-----\t---\t------")
	for _, o := range orders {
		price := "-"
		if o.Price > 0 {
			price = fmt.Sprintf("$%.4f", o.Price)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%.0f\t%.0f\t%s\t%s\t%s\n",
			o.OrderID, o.Symbol, o.Side, o.OrderType, o.Qty, o.FilledQty,
			price, o.TimeInForce, o.Status)
	}
	w.Flush()
}

func cmdBuy(c *client.IBKRClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("buy", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "Limit price")
	fs.Parse(args[2:])

	symbol := strings.ToUpper(args[0])
	qty := parseInt(args[1], "qty")

	if *limitPrice > 0 {
		confirmLive(live, "BUY %d %s @ LIMIT $%.4f", qty, symbol, *limitPrice)
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

func cmdSell(c *client.IBKRClient, args []string, asJSON, live bool) {
	fs := flag.NewFlagSet("sell", flag.ExitOnError)
	limitPrice := fs.Float64("limit", 0, "Limit price (GTC)")
	fs.Parse(args[2:])

	symbol := strings.ToUpper(args[0])
	qty := parseInt(args[1], "qty")

	if *limitPrice > 0 {
		confirmLive(live, "SELL %d %s @ LIMIT $%.4f (GTC)", qty, symbol, *limitPrice)
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

func cmdClose(c *client.IBKRClient, symbol string, asJSON, live bool) {
	symbol = strings.ToUpper(symbol)

	// Get position to determine quantity.
	positions, err := ops.GetPositions(c)
	check(err)

	var posQty int
	for _, p := range positions {
		if p.Symbol == symbol {
			posQty = int(math.Abs(p.Qty))
			break
		}
	}
	if posQty == 0 {
		fatalf("no open position for %s", symbol)
	}

	confirmLive(live, "CLOSE %d %s", posQty, symbol)
	err = ops.ClosePosition(c, symbol, posQty)
	check(err)
	if asJSON {
		printJSON(map[string]string{"symbol": symbol, "status": "closed"})
	} else {
		fmt.Printf("Closed all %s positions (%d shares)\n", symbol, posQty)
	}
}

func cmdCancel(c *client.IBKRClient, orderID string, asJSON, live bool) {
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
		fmt.Printf("[%s] %s %d %s %s @ $%.4f → order %s\n",
			r.Mode, r.Action, r.Qty, r.Symbol, r.Type, r.Price, r.OrderID)
	} else {
		fmt.Printf("[%s] %s %d %s %s → order %s\n",
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
		fatalf("usage: ibkr-cli %s", usage)
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
