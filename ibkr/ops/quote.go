package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Quote is a price snapshot for a single symbol.
type Quote struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	PrevClose float64 `json:"prev_close"`
	ChangePct float64 `json:"change_pct"`
}

// GetQuote returns a price snapshot.
// Tries IBKR market data snapshot first, falls back to Yahoo Finance.
func GetQuote(c Caller, symbol string) (Quote, error) {
	conid, err := c.ResolveConID(symbol)
	if err == nil {
		// IBKR snapshot fields:
		// 31=last, 7295=open, 7296=high, 7297=low, 7762=volume, 7291=prev close
		fields := "31,7295,7296,7297,7762,7291"
		data, snapErr := c.Get("/v1/api/iserver/marketdata/snapshot", map[string]string{
			"conids": strconv.Itoa(conid),
			"fields": fields,
		})
		if snapErr == nil {
			var snaps []map[string]json.RawMessage
			if json.Unmarshal(data, &snaps) == nil && len(snaps) > 0 {
				snap := snaps[0]
				price := extractFloat(snap, "31")
				if price > 0 {
					open := extractFloat(snap, "7295")
					high := extractFloat(snap, "7296")
					low := extractFloat(snap, "7297")
					volume := extractFloat(snap, "7762")
					prevClose := extractFloat(snap, "7291")
					var changePct float64
					if prevClose > 0 {
						changePct = (price - prevClose) / prevClose * 100
					}
					return Quote{
						Symbol:    symbol,
						Price:     price,
						Open:      open,
						High:      high,
						Low:       low,
						Volume:    volume,
						PrevClose: prevClose,
						ChangePct: changePct,
					}, nil
				}
			}
		}
	}

	// Fall back to Yahoo Finance.
	return yahooQuote(symbol)
}

func extractFloat(m map[string]json.RawMessage, key string) float64 {
	v, ok := m[key]
	if !ok {
		return 0
	}
	var f float64
	if json.Unmarshal(v, &f) == nil {
		return f
	}
	// Try string.
	var s string
	if json.Unmarshal(v, &s) == nil {
		f, _ = strconv.ParseFloat(s, 64)
		return f
	}
	return 0
}

func yahooQuote(symbol string) (Quote, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", symbol)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: %w", symbol, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: read: %w", symbol, err)
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
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: parse: %w", symbol, err)
	}
	if len(yr.Chart.Result) == 0 {
		return Quote{}, fmt.Errorf("yahoo_quote %s: no data", symbol)
	}
	m := yr.Chart.Result[0].Meta
	var changePct float64
	if m.ChartPreviousClose > 0 {
		changePct = (m.RegularMarketPrice - m.ChartPreviousClose) / m.ChartPreviousClose * 100
	}
	return Quote{
		Symbol:    symbol,
		Price:     m.RegularMarketPrice,
		Open:      m.RegularMarketOpen,
		High:      m.DayHigh,
		Low:       m.DayLow,
		Volume:    m.RegularMarketVol,
		PrevClose: m.ChartPreviousClose,
		ChangePct: changePct,
	}, nil
}
