// Package data fetches historical OHLCV bars from configurable sources.
package data

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Bar is a single OHLCV trading day.
type Bar struct {
	Date   time.Time
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}

// Fetch returns daily bars for symbol between from and to (inclusive).
// source: "yahoo" (default), "alphavantage", "polygon"
func Fetch(symbol, source, apiKey string, from, to time.Time) ([]Bar, error) {
	switch source {
	case "", "yahoo":
		return fetchYahoo(symbol, from, to)
	case "alphavantage":
		if apiKey == "" {
			return nil, fmt.Errorf("alphavantage requires api_key in config")
		}
		return fetchAlphaVantage(symbol, apiKey, from, to)
	case "polygon":
		if apiKey == "" {
			return nil, fmt.Errorf("polygon requires api_key in config")
		}
		return fetchPolygon(symbol, apiKey, from, to)
	default:
		return nil, fmt.Errorf("unknown data source %q — use: yahoo, alphavantage, polygon", source)
	}
}

// ─── YAHOO FINANCE ───────────────────────────────────────────────────────────

func fetchYahoo(symbol string, from, to time.Time) ([]Bar, error) {
	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&period1=%d&period2=%d",
		symbol, from.Unix(), to.Add(24*time.Hour).Unix(),
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo fetch %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo read %s: %w", symbol, err)
	}

	var yr struct {
		Chart struct {
			Result []struct {
				Timestamps []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Open   []float64 `json:"open"`
						High   []float64 `json:"high"`
						Low    []float64 `json:"low"`
						Close  []float64 `json:"close"`
						Volume []int64   `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error *struct{ Description string } `json:"error"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return nil, fmt.Errorf("yahoo parse %s: %w", symbol, err)
	}
	if yr.Chart.Error != nil {
		return nil, fmt.Errorf("yahoo %s: %s", symbol, yr.Chart.Error.Description)
	}
	if len(yr.Chart.Result) == 0 || len(yr.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("yahoo %s: no data returned", symbol)
	}

	res := yr.Chart.Result[0]
	q := res.Indicators.Quote[0]
	var bars []Bar
	for i, ts := range res.Timestamps {
		if i >= len(q.Close) || q.Close[i] == 0 {
			continue // skip incomplete bars
		}
		bars = append(bars, Bar{
			Date:   time.Unix(ts, 0).UTC(),
			Open:   safeIdx(q.Open, i),
			High:   safeIdx(q.High, i),
			Low:    safeIdx(q.Low, i),
			Close:  q.Close[i],
			Volume: safeIdxInt(q.Volume, i),
		})
	}
	return bars, nil
}

// ─── ALPHA VANTAGE ───────────────────────────────────────────────────────────

func fetchAlphaVantage(symbol, apiKey string, from, to time.Time) ([]Bar, error) {
	url := fmt.Sprintf(
		"https://www.alphavantage.co/query?function=TIME_SERIES_DAILY_ADJUSTED&symbol=%s&outputsize=full&apikey=%s",
		symbol, apiKey,
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("alphavantage build request: %w", err)
	}

	resp, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("alphavantage fetch %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("alphavantage read %s: %w", symbol, err)
	}

	var result struct {
		TimeSeries map[string]struct {
			Open   string `json:"1. open"`
			High   string `json:"2. high"`
			Low    string `json:"3. low"`
			Close  string `json:"5. adjusted close"`
			Volume string `json:"6. volume"`
		} `json:"Time Series (Daily)"`
		Note string `json:"Note"`
		Info string `json:"Information"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("alphavantage parse %s: %w", symbol, err)
	}
	if result.Note != "" || result.Info != "" {
		return nil, fmt.Errorf("alphavantage %s: rate limited — %s%s", symbol, result.Note, result.Info)
	}
	if result.TimeSeries == nil {
		return nil, fmt.Errorf("alphavantage %s: no data returned", symbol)
	}

	var bars []Bar
	for dateStr, v := range result.TimeSeries {
		t, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if t.Before(from) || t.After(to) {
			continue
		}
		bars = append(bars, Bar{
			Date:   t,
			Open:   parseF(v.Open),
			High:   parseF(v.High),
			Low:    parseF(v.Low),
			Close:  parseF(v.Close),
			Volume: parseInt(v.Volume),
		})
	}
	sortBars(bars)
	return bars, nil
}

// ─── POLYGON ─────────────────────────────────────────────────────────────────

func fetchPolygon(symbol, apiKey string, from, to time.Time) ([]Bar, error) {
	url := fmt.Sprintf(
		"https://api.polygon.io/v2/aggs/ticker/%s/range/1/day/%s/%s?adjusted=true&sort=asc&limit=5000&apiKey=%s",
		symbol,
		from.Format("2006-01-02"),
		to.Format("2006-01-02"),
		apiKey,
	)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("polygon build request: %w", err)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("polygon fetch %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("polygon read %s: %w", symbol, err)
	}

	var result struct {
		Status  string `json:"status"`
		Results []struct {
			T int64   `json:"t"` // unix ms
			O float64 `json:"o"`
			H float64 `json:"h"`
			L float64 `json:"l"`
			C float64 `json:"c"`
			V float64 `json:"v"`
		} `json:"results"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("polygon parse %s: %w", symbol, err)
	}
	if result.Error != "" {
		return nil, fmt.Errorf("polygon %s: %s", symbol, result.Error)
	}

	var bars []Bar
	for _, r := range result.Results {
		bars = append(bars, Bar{
			Date:   time.UnixMilli(r.T).UTC(),
			Open:   r.O,
			High:   r.H,
			Low:    r.L,
			Close:  r.C,
			Volume: int64(r.V),
		})
	}
	return bars, nil
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func safeIdx(s []float64, i int) float64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func safeIdxInt(s []int64, i int) int64 {
	if i < len(s) {
		return s[i]
	}
	return 0
}

func parseF(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseInt(s string) int64 {
	var n int64
	fmt.Sscanf(s, "%d", &n)
	return n
}

func sortBars(bars []Bar) {
	// insertion sort — bars are usually nearly sorted
	for i := 1; i < len(bars); i++ {
		for j := i; j > 0 && bars[j].Date.Before(bars[j-1].Date); j-- {
			bars[j], bars[j-1] = bars[j-1], bars[j]
		}
	}
}
