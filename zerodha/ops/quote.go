package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

// GetQuote returns a price snapshot via Yahoo Finance.
// Kite's quote API requires instrument tokens, so we fall back to Yahoo.
// Indian symbols get ".NS" suffix for Yahoo (e.g., RELIANCE -> RELIANCE.NS).
func GetQuote(_ Caller, symbol string) (Quote, error) {
	_, sym := parseExchangeForQuote(symbol)
	yahooSym := sym + ".NS"
	return yahooQuote(yahooSym, sym)
}

// parseExchangeForQuote strips exchange prefix for quote lookup.
func parseExchangeForQuote(symbol string) (string, string) {
	if idx := strings.Index(symbol, ":"); idx > 0 {
		return strings.ToUpper(symbol[:idx]), strings.ToUpper(symbol[idx+1:])
	}
	return "NSE", strings.ToUpper(symbol)
}

func yahooQuote(yahooSym, displaySym string) (Quote, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", yahooSym)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: %w", displaySym, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: %w", displaySym, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: read: %w", displaySym, err)
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
		return Quote{}, fmt.Errorf("yahoo_quote %s: parse: %w", displaySym, err)
	}
	if len(yr.Chart.Result) == 0 {
		return Quote{}, fmt.Errorf("yahoo_quote %s: no data", displaySym)
	}
	m := yr.Chart.Result[0].Meta
	var changePct float64
	if m.ChartPreviousClose > 0 {
		changePct = (m.RegularMarketPrice - m.ChartPreviousClose) / m.ChartPreviousClose * 100
	}
	return Quote{
		Symbol:    displaySym,
		Price:     m.RegularMarketPrice,
		Open:      m.RegularMarketOpen,
		High:      m.DayHigh,
		Low:       m.DayLow,
		Volume:    m.RegularMarketVol,
		PrevClose: m.ChartPreviousClose,
		ChangePct: changePct,
	}, nil
}
