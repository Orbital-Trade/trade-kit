package ops

// GetQuote — returns market data for a symbol via eToro Market Data API.
//
// Endpoints:
//   GET /api/v1/market-data/instruments/{instrumentId}/candles
//   GET /api/v1/market-data/instruments/{instrumentId}/closing-prices
//   GET /api/v1/market-data/rates
//
// Rate limit: 120 req/60s (shared pool).
//
// Falls back to Yahoo Finance if eToro market data is unavailable.

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
	InstrID   int     `json:"instrument_id,omitempty"`
	Price     float64 `json:"price"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Volume    float64 `json:"volume"`
	PrevClose float64 `json:"prev_close"`
	ChangePct float64 `json:"change_pct"`
}

// GetQuote returns a price snapshot for a symbol.
// Tries eToro Market Data API first, falls back to Yahoo Finance.
func GetQuote(c Caller, symbol string) (Quote, error) {
	// Try eToro first — resolve instrument and fetch market data.
	inst, err := ResolveInstrument(c, symbol)
	if err == nil {
		q, err := etoroQuote(c, inst)
		if err == nil {
			return q, nil
		}
	}

	// Fall back to Yahoo Finance.
	return yahooQuote(symbol)
}

func etoroQuote(c Caller, inst Instrument) (Quote, error) {
	path := fmt.Sprintf("/api/v1/market-data/instruments/%d/candles", inst.ID)
	data, err := c.Get(path, map[string]string{
		"period":   "OneDay",
		"interval": "OneDay",
		"limit":    "2",
	})
	if err != nil {
		return Quote{}, fmt.Errorf("etoro_quote %s: %w", inst.Symbol, err)
	}

	var candles []struct {
		Open   float64 `json:"Open"`
		High   float64 `json:"High"`
		Low    float64 `json:"Low"`
		Close  float64 `json:"Close"`
		Volume float64 `json:"Volume"`
	}
	if err := json.Unmarshal(data, &candles); err != nil {
		return Quote{}, fmt.Errorf("etoro_quote %s: parse: %w", inst.Symbol, err)
	}

	if len(candles) == 0 {
		return Quote{}, fmt.Errorf("etoro_quote %s: no candle data", inst.Symbol)
	}

	latest := candles[len(candles)-1]
	var prevClose float64
	if len(candles) > 1 {
		prevClose = candles[len(candles)-2].Close
	}

	var changePct float64
	if prevClose > 0 {
		changePct = (latest.Close - prevClose) / prevClose * 100
	}

	return Quote{
		Symbol:    inst.Symbol,
		InstrID:   inst.ID,
		Price:     latest.Close,
		Open:      latest.Open,
		High:      latest.High,
		Low:       latest.Low,
		Volume:    latest.Volume,
		PrevClose: prevClose,
		ChangePct: changePct,
	}, nil
}

func yahooQuote(symbol string) (Quote, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", symbol)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: build request: %w", symbol, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: %w", symbol, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quote{}, fmt.Errorf("yahoo_quote %s: read response: %w", symbol, err)
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
