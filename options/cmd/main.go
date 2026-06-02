// options — options chain viewer
//
// Displays calls and puts for any US optionable stock using Yahoo Finance.
// No API key or broker connection required.
//
// Usage:
//
//	options chain AAPL                    → nearest expiry, calls + puts
//	options chain AAPL --expiry 2026-06-20
//	options chain AAPL --calls
//	options chain AAPL --puts
//	options chain AAPL --json
//	options expiries AAPL                 → list available expiry dates
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ─── YAHOO TYPES ─────────────────────────────────────────────────────────────

type yahooOptions struct {
	OptionChain struct {
		Result []struct {
			UnderlyingSymbol string    `json:"underlyingSymbol"`
			ExpirationDates  []int64   `json:"expirationDates"`
			Options          []expiry  `json:"options"`
			Quote            quote     `json:"quote"`
		} `json:"result"`
		Error *struct{ Description string } `json:"error"`
	} `json:"optionChain"`
}

type expiry struct {
	ExpirationDate int64    `json:"expirationDate"`
	Calls          []option `json:"calls"`
	Puts           []option `json:"puts"`
}

type option struct {
	ContractSymbol    string  `json:"contractSymbol"`
	Strike            float64 `json:"strike"`
	Bid               float64 `json:"bid"`
	Ask               float64 `json:"ask"`
	LastPrice         float64 `json:"lastPrice"`
	ImpliedVolatility float64 `json:"impliedVolatility"`
	OpenInterest      int     `json:"openInterest"`
	Volume            int     `json:"volume"`
	InTheMoney        bool    `json:"inTheMoney"`
	Delta             float64 `json:"delta,omitempty"`
}

type quote struct {
	RegularMarketPrice float64 `json:"regularMarketPrice"`
	Symbol             string  `json:"symbol"`
}

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	subCmd := os.Args[1]
	symbol := strings.ToUpper(os.Args[2])

	switch subCmd {
	case "chain":
		cmdChain(symbol, os.Args[3:])
	case "expiries":
		cmdExpiries(symbol)
	default:
		usage()
		os.Exit(1)
	}
}

// ─── EXPIRIES ────────────────────────────────────────────────────────────────

func cmdExpiries(symbol string) {
	data, err := fetchOptions(symbol, 0)
	if err != nil {
		fatalf("%v", err)
	}
	if len(data.OptionChain.Result) == 0 {
		fatalf("no options data for %s", symbol)
	}
	res := data.OptionChain.Result[0]
	fmt.Printf("Expiry dates for %s (%d available)\n\n", symbol, len(res.ExpirationDates))
	for _, ts := range res.ExpirationDates {
		t := time.Unix(ts, 0).UTC()
		dte := int(time.Until(t).Hours() / 24)
		fmt.Printf("  %s  (%d DTE)\n", t.Format("2006-01-02"), dte)
	}
}

// ─── CHAIN ───────────────────────────────────────────────────────────────────

func cmdChain(symbol string, args []string) {
	expiry := ""
	showCalls := true
	showPuts := true
	asJSON := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--expiry":
			if i+1 < len(args) {
				expiry = args[i+1]
				i++
			}
		case "--calls":
			showPuts = false
		case "--puts":
			showCalls = false
		case "--json":
			asJSON = true
		}
	}

	// Resolve expiry timestamp
	var expiryTs int64
	if expiry != "" {
		t, err := time.Parse("2006-01-02", expiry)
		if err != nil {
			fatalf("--expiry must be YYYY-MM-DD, got %q", expiry)
		}
		expiryTs = t.Unix()
	}

	data, err := fetchOptions(symbol, expiryTs)
	if err != nil {
		fatalf("%v", err)
	}
	if data.OptionChain.Error != nil {
		fatalf("yahoo: %s", data.OptionChain.Error.Description)
	}
	if len(data.OptionChain.Result) == 0 {
		fatalf("no options data for %s — check symbol or market hours", symbol)
	}

	res := data.OptionChain.Result[0]
	if len(res.Options) == 0 {
		fatalf("no contracts returned for %s", symbol)
	}

	underlying := res.Quote.RegularMarketPrice
	exp := res.Options[0]
	expDate := time.Unix(exp.ExpirationDate, 0).UTC()
	dte := int(time.Until(expDate).Hours() / 24)

	if asJSON {
		type output struct {
			Symbol     string   `json:"symbol"`
			Price      float64  `json:"price"`
			Expiry     string   `json:"expiry"`
			DTE        int      `json:"dte"`
			Calls      []option `json:"calls,omitempty"`
			Puts       []option `json:"puts,omitempty"`
		}
		out := output{
			Symbol: symbol,
			Price:  underlying,
			Expiry: expDate.Format("2006-01-02"),
			DTE:    dte,
		}
		if showCalls {
			out.Calls = exp.Calls
		}
		if showPuts {
			out.Puts = exp.Puts
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
		return
	}

	fmt.Printf("  %s  |  Underlying: $%.2f  |  Expiry: %s (%d DTE)\n\n",
		symbol, underlying, expDate.Format("2006-01-02"), dte)

	if showCalls && len(exp.Calls) > 0 {
		printChain("CALLS", exp.Calls, underlying)
	}
	if showPuts && len(exp.Puts) > 0 {
		if showCalls {
			fmt.Println()
		}
		printChain("PUTS", exp.Puts, underlying)
	}
}

func printChain(label string, contracts []option, underlying float64) {
	// Sort by strike ascending
	sort.Slice(contracts, func(i, j int) bool {
		return contracts[i].Strike < contracts[j].Strike
	})

	fmt.Printf("  %s\n", label)
	fmt.Printf("  %-8s  %7s  %7s  %7s  %7s  %7s  %8s  %3s\n",
		"STRIKE", "BID", "ASK", "LAST", "IV%", "OI", "VOLUME", "ITM")
	fmt.Printf("  %s\n", strings.Repeat("─", 68))

	for _, c := range contracts {
		itm := " "
		if c.InTheMoney {
			itm = "*"
		}
		bid := fmtPrice(c.Bid)
		ask := fmtPrice(c.Ask)
		last := fmtPrice(c.LastPrice)
		iv := "─"
		if c.ImpliedVolatility > 0 {
			iv = fmt.Sprintf("%.1f%%", c.ImpliedVolatility*100)
		}
		oi := strconv.Itoa(c.OpenInterest)
		vol := strconv.Itoa(c.Volume)

		// Highlight near-the-money strikes (within 2% of underlying)
		nearMoney := math.Abs(c.Strike-underlying)/underlying < 0.02
		prefix := "  "
		if nearMoney {
			prefix = "> "
		}
		fmt.Printf("%s%-8.2f  %7s  %7s  %7s  %7s  %7s  %8s  %s\n",
			prefix, c.Strike, bid, ask, last, iv, oi, vol, itm)
	}
}

// ─── FETCH ───────────────────────────────────────────────────────────────────

// yahooClient holds a shared HTTP client with cookie jar and crumb for Yahoo Finance.
type yahooClient struct {
	hc    *http.Client
	crumb string
}

// newYahooClient bootstraps the Yahoo Finance session: visits fc.yahoo.com to
// pick up the A1 cookie, then fetches the crumb required for API calls.
func newYahooClient() (*yahooClient, error) {
	jar, _ := cookiejar.New(nil)
	hc := &http.Client{Jar: jar}
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"

	// 1. Seed cookie jar
	fcReq, err := http.NewRequest("GET", "https://fc.yahoo.com/", nil)
	if err != nil {
		return nil, err
	}
	fcReq.Header.Set("User-Agent", ua)
	fcResp, err := hc.Do(fcReq)
	if err != nil {
		return nil, fmt.Errorf("cookie seed: %w", err)
	}
	fcResp.Body.Close()

	// 2. Fetch crumb
	crumbReq, err := http.NewRequest("GET", "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
	if err != nil {
		return nil, err
	}
	crumbReq.Header.Set("User-Agent", ua)
	crumbReq.Header.Set("Referer", "https://finance.yahoo.com")
	crumbResp, err := hc.Do(crumbReq)
	if err != nil {
		return nil, fmt.Errorf("crumb: %w", err)
	}
	defer crumbResp.Body.Close()
	crumbBody, err := io.ReadAll(crumbResp.Body)
	if err != nil {
		return nil, fmt.Errorf("crumb read: %w", err)
	}
	crumb := strings.TrimSpace(string(crumbBody))
	if crumb == "" || strings.Contains(crumb, "Unauthorized") {
		return nil, fmt.Errorf("could not obtain Yahoo Finance session — try again in a few seconds")
	}
	return &yahooClient{hc: hc, crumb: crumb}, nil
}

func (yc *yahooClient) fetchOptions(symbol string, expiryTs int64) (*yahooOptions, error) {
	ua := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36"
	endpoint := "https://query2.finance.yahoo.com/v7/finance/options/" + symbol

	params := url.Values{"crumb": {yc.crumb}}
	if expiryTs > 0 {
		params.Set("date", strconv.FormatInt(expiryTs, 10))
	}
	endpoint += "?" + params.Encode()

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", ua)

	resp, err := yc.hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	var data yahooOptions
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	return &data, nil
}

// module-level client, initialised once per invocation
var yc *yahooClient

func getYahooClient() *yahooClient {
	if yc != nil {
		return yc
	}
	var err error
	yc, err = newYahooClient()
	if err != nil {
		fatalf("yahoo session: %v", err)
	}
	return yc
}

func fetchOptions(symbol string, expiryTs int64) (*yahooOptions, error) {
	return getYahooClient().fetchOptions(symbol, expiryTs)
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func fmtPrice(p float64) string {
	if p == 0 {
		return "─"
	}
	return fmt.Sprintf("$%.2f", p)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "options: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`options — options chain viewer

Displays calls and puts for any US optionable stock.
Data source: Yahoo Finance (no API key required).

COMMANDS
  chain <SYMBOL>       Show options chain (nearest expiry by default)
  expiries <SYMBOL>    List all available expiry dates

CHAIN FLAGS
  --expiry <YYYY-MM-DD>  Specific expiry date
  --calls                Show calls only
  --puts                 Show puts only
  --json                 Machine-readable JSON output

DISPLAY
  >  Near-the-money (within 2% of underlying price)
  *  In-the-money contract

EXAMPLES
  options chain AAPL
  options chain AAPL --calls
  options chain AAPL --expiry 2026-06-20
  options chain AAPL --expiry 2026-06-20 --puts --json
  options expiries NVDA

BUILD
  cd options && go build -o options ./cmd/

`)
}
