package ops

// GetQuote — MCP tool: tiger_quote
//
// Returns a real-time price snapshot for a stock symbol using daily bars.
// During market hours, the current bar's close updates continuously.
// Calls Tiger REST method: kline (period=day, limit=2).

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Quote is a real-time price snapshot for a single symbol.
type Quote struct {
	Symbol    string  `json:"symbol"`
	Market    string  `json:"market"`     // "US" or "SG"
	Currency  string  `json:"currency"`   // "USD" or "SGD"
	Price     float64 `json:"price"`      // latest close (real-time during market hours)
	Open      float64 `json:"open"`       // today's open
	High      float64 `json:"high"`       // today's high
	Low       float64 `json:"low"`        // today's low
	Volume    float64 `json:"volume"`     // today's volume
	PrevClose float64 `json:"prev_close"` // previous session close
	ChangePct float64 `json:"change_pct"` // % change from prev_close; 0 if unavailable
}

// GetQuote returns a real-time price snapshot.
// Tries Tiger "brief" first; falls back to Yahoo Finance on permission errors.
// This handles two Tiger limitations: kline has a 20-symbol quota; brief requires
// specific market subscriptions. Yahoo Finance works for both US and SGX symbols.
func GetQuote(c Caller, symbol string) (Quote, error) {
	info := DetectMarket(symbol)

	// Try Tiger brief.
	data, err := c.Call("brief", map[string]interface{}{
		"symbols": []string{info.Symbol},
		"market":  info.Market,
		"lang":    "en_US",
	})
	if err == nil {
		var items []struct {
			LatestPrice float64 `json:"latest_price"`
			PreClose    float64 `json:"pre_close"`
			OpenPrice   float64 `json:"open_price"`
			HighPrice   float64 `json:"high_price"`
			LowPrice    float64 `json:"low_price"`
			Volume      float64 `json:"volume"`
		}
		if json.Unmarshal(data, &items) == nil && len(items) > 0 {
			q := items[0]
			var changePct float64
			if q.PreClose > 0 {
				changePct = (q.LatestPrice - q.PreClose) / q.PreClose * 100
			}
			return Quote{
				Symbol:    info.Symbol,
				Market:    info.Market,
				Currency:  info.Currency,
				Price:     q.LatestPrice,
				Open:      q.OpenPrice,
				High:      q.HighPrice,
				Low:       q.LowPrice,
				Volume:    q.Volume,
				PrevClose: q.PreClose,
				ChangePct: changePct,
			}, nil
		}
	}

	// Fall back to Yahoo Finance — no quotas, works for US and SGX.
	return yahooQuote(info)
}

func yahooQuote(info SymbolInfo) (Quote, error) {
	ticker := info.Symbol
	if info.Market == "SG" {
		ticker = info.Symbol + ".SI"
	}
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", ticker)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: build request: %w", info.Symbol, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: %w", info.Symbol, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: read response: %w", info.Symbol, err)
	}

	var yr struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					ChartPreviousClose float64 `json:"chartPreviousClose"`
					RegularMarketOpen  float64 `json:"regularMarketOpen"`
					DayHigh            float64 `json:"regularMarketDayHigh"`
					DayLow             float64 `json:"regularMarketDayLow"`
					RegularMarketVol   float64 `json:"regularMarketVolume"`
				} `json:"meta"`
			} `json:"result"`
			Error interface{} `json:"error"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: parse: %w", info.Symbol, err)
	}
	if len(yr.Chart.Result) == 0 {
		return Quote{}, fmt.Errorf("yahoo_quote %s: no data", info.Symbol)
	}
	m := yr.Chart.Result[0].Meta
	var changePct float64
	if m.ChartPreviousClose > 0 {
		changePct = (m.RegularMarketPrice - m.ChartPreviousClose) / m.ChartPreviousClose * 100
	}
	return Quote{
		Symbol:    info.Symbol,
		Market:    info.Market,
		Currency:  info.Currency,
		Price:     m.RegularMarketPrice,
		Open:      m.RegularMarketOpen,
		High:      m.DayHigh,
		Low:       m.DayLow,
		Volume:    m.RegularMarketVol,
		PrevClose: m.ChartPreviousClose,
		ChangePct: changePct,
	}, nil
}
