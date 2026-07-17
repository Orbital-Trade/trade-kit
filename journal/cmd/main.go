// journal — trade journal with SQLite storage
//
// Records every fill into a local database and provides P&L reporting.
// Duplicate order IDs (same broker + order_id) are silently ignored on import.
//
// Usage:
//
//	journal add BUY LUNR 10 35.50 [--broker tiger] [--order-id X] [--strategy daytrader] [--note text]
//	journal list [--symbol SYM] [--days N]
//	journal pnl  [--symbol SYM]
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"journal/internal/db"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	dbPath := db.DefaultPath()
	for i := 1; i < len(os.Args)-1; i++ {
		if os.Args[i] == "--db" {
			dbPath = os.Args[i+1]
		}
	}

	store, err := db.Open(dbPath)
	if err != nil {
		fatalf("open db: %v", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "journal: close db: %v\n", err)
		}
	}()

	switch os.Args[1] {
	case "add":
		cmdAdd(store, os.Args[2:])
	case "list", "ls":
		cmdList(store, os.Args[2:])
	case "pnl":
		cmdPnL(store, os.Args[2:])
	case "export":
		cmdExport(store, os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

// ─── ADD ─────────────────────────────────────────────────────────────────────

func cmdAdd(store *db.DB, args []string) {
	// journal add BUY LUNR 10 35.50 [flags]
	if len(args) < 4 {
		fatalf("usage: add <BUY|SELL> <SYMBOL> <QTY> <PRICE> [--broker x] [--order-id x] [--strategy x] [--note x] [--date YYYY-MM-DD]")
	}

	side := strings.ToUpper(args[0])
	if side != "BUY" && side != "SELL" {
		fatalf("side must be BUY or SELL, got %q", args[0])
	}
	symbol := strings.ToUpper(strings.TrimSpace(args[1]))
	if symbol == "" {
		fatalf("symbol must not be empty")
	}
	qty, err := strconv.Atoi(args[2])
	if err != nil || qty <= 0 {
		fatalf("qty must be a positive integer, got %q", args[2])
	}
	price, err := strconv.ParseFloat(args[3], 64)
	if err != nil || price <= 0 {
		fatalf("price must be a positive number, got %q", args[3])
	}

	t := db.Trade{
		Symbol:   symbol,
		Side:     side,
		Qty:      qty,
		Price:    price,
		FilledAt: time.Now().UTC(),
	}

	for i := 4; i < len(args)-1; i++ {
		switch args[i] {
		case "--broker":
			t.Broker = args[i+1]
			i++
		case "--order-id":
			t.OrderID = args[i+1]
			i++
		case "--strategy":
			t.Strategy = args[i+1]
			i++
		case "--note":
			t.Note = args[i+1]
			i++
		case "--date":
			parsed, perr := time.Parse("2006-01-02", args[i+1])
			if perr != nil {
				fatalf("--date must be YYYY-MM-DD, got %q", args[i+1])
			}
			t.FilledAt = parsed.UTC()
			i++
		}
	}

	id, err := store.Add(t)
	if err != nil {
		fatalf("add: %v", err)
	}
	if id == 0 {
		fmt.Printf("skipped — duplicate order already recorded\n")
		return
	}
	fmt.Printf("recorded #%d  %s %s %d @ $%.4f\n", id, t.Side, t.Symbol, t.Qty, t.Price)
}

// ─── LIST ────────────────────────────────────────────────────────────────────

func cmdList(store *db.DB, args []string) {
	symbol := ""
	days := 0
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--symbol":
			symbol = strings.ToUpper(args[i+1])
			i++
		case "--days":
			days, _ = strconv.Atoi(args[i+1])
			i++
		}
	}

	trades, err := store.List(symbol, days)
	if err != nil {
		fatalf("list: %v", err)
	}
	if len(trades) == 0 {
		fmt.Println("No trades recorded.")
		return
	}

	fmt.Printf("%-6s  %-4s  %-8s  %6s  %10s  %-10s  %-12s  %-10s  %s\n",
		"#", "SIDE", "SYMBOL", "QTY", "PRICE", "DATE", "BROKER", "STRATEGY", "NOTE")
	fmt.Println(strings.Repeat("─", 90))
	for _, t := range trades {
		fmt.Printf("%-6d  %-4s  %-8s  %6d  %10.4f  %-10s  %-12s  %-10s  %s\n",
			t.ID, t.Side, t.Symbol, t.Qty, t.Price,
			t.FilledAt.Format("2006-01-02"),
			t.Broker, t.Strategy, t.Note)
	}
}

// ─── P&L ─────────────────────────────────────────────────────────────────────

func cmdPnL(store *db.DB, args []string) {
	symbol := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--symbol" {
			symbol = strings.ToUpper(args[i+1])
			i++
		}
	}

	results, err := store.ComputePnL(symbol)
	if err != nil {
		fatalf("pnl: %v", err)
	}
	if len(results) == 0 {
		fmt.Println("No trades recorded.")
		return
	}

	fmt.Printf("%-8s  %10s  %5s  %5s  %6s\n", "SYMBOL", "REALIZED", "WINS", "LOSS", "TRADES")
	fmt.Println(strings.Repeat("─", 42))

	totalRealized := 0.0
	for _, r := range results {
		sign := "+"
		if r.Realized < 0 {
			sign = ""
		}
		fmt.Printf("%-8s  %s%9.2f  %5d  %5d  %6d\n",
			r.Symbol, sign, r.Realized, r.WinCount, r.LoseCount, r.Trades)
		totalRealized += r.Realized
	}

	fmt.Println(strings.Repeat("─", 42))
	sign := "+"
	if totalRealized < 0 {
		sign = ""
	}
	fmt.Printf("%-8s  %s%9.2f\n", "TOTAL", sign, totalRealized)
}

// ─── EXPORT ─────────────────────────────────────────────────────────────────

func cmdExport(store *db.DB, args []string) {
	symbol := ""
	days := 0
	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--symbol":
			symbol = strings.ToUpper(args[i+1])
			i++
		case "--days":
			days, _ = strconv.Atoi(args[i+1])
			i++
		}
	}

	trades, err := store.List(symbol, days)
	if err != nil {
		fatalf("export: %v", err)
	}

	fmt.Println("id,side,symbol,qty,price,date,broker,order_id,strategy,note")
	for _, t := range trades {
		note := strings.ReplaceAll(t.Note, ",", ";")
		fmt.Printf("%d,%s,%s,%d,%.4f,%s,%s,%s,%s,%s\n",
			t.ID, t.Side, t.Symbol, t.Qty, t.Price,
			t.FilledAt.Format("2006-01-02"),
			t.Broker, t.OrderID, t.Strategy, note)
	}
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "journal: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`journal — trade journal with SQLite storage

Records fills and computes realized P&L (FIFO cost basis).
Database: ~/.trade-kit/journal/trades.db

COMMANDS
  add <BUY|SELL> <SYMBOL> <QTY> <PRICE>   Record a trade manually
  list [--symbol SYM] [--days N]           Show trade history
  pnl  [--symbol SYM]                      Realized P&L per symbol
  export [--symbol SYM] [--days N]         Export trades as CSV to stdout

ADD FLAGS
  --broker <name>       tiger | moomoo | manual (default: empty)
  --order-id <id>       Broker order ID — duplicates are silently ignored
  --strategy <name>     daytrader | bounce | earnings | manual
  --note <text>         Free-text note
  --date <YYYY-MM-DD>   Fill date (default: today)

GLOBAL FLAGS
  --db <path>           Override database path

EXAMPLES
  journal add BUY  LUNR 10 35.50 --broker tiger --strategy daytrader
  journal add SELL LUNR 10 38.20 --broker tiger --note "target hit"
  journal list
  journal list --symbol LUNR
  journal list --days 30
  journal pnl
  journal pnl --symbol LUNR
  journal export > trades.csv
  journal export --symbol LUNR --days 90 > lunr.csv

BUILD
  cd journal && go build -o journal ./cmd/

`)
}
