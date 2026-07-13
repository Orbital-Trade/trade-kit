package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
// Tries Alpaca data API first, falls back to Yahoo Finance.
func GetQuote(c Caller, symbol string) (Quote, error) {
	// Try Alpaca snapshot endpoint.
	data, err := c.DataGet("/v2/stocks/"+symbol+"/snapshot", nil)
	if err == nil {
		var snap struct {
			LatestTrade struct {
				Price float64 `json:"p"`
			} `json:"latestTrade"`
			DailyBar struct {
				Open   float64 `json:"o"`
				High   float64 `json:"h"`
				Low    float64 `json:"l"`
				Close  float64 `json:"c"`
				Volume float64 `json:"v"`
			} `json:"dailyBar"`
			PrevDailyBar struct {
				Close float64 `json:"c"`
			} `json:"prevDailyBar"`
		}
		if json.Unmarshal(data, &snap) == nil && snap.LatestTrade.Price > 0 {
			price := snap.LatestTrade.Price
			prevClose := snap.PrevDailyBar.Close
			var changePct float64
			if prevClose > 0 {
				changePct = (price - prevClose) / prevClose * 100
			}
			return Quote{
				Symbol:    symbol,
				Price:     price,
				Open:      snap.DailyBar.Open,
				High:      snap.DailyBar.High,
				Low:       snap.DailyBar.Low,
				Volume:    snap.DailyBar.Volume,
				PrevClose: prevClose,
				ChangePct: changePct,
			}, nil
		}
	}

	// Fall back to Yahoo Finance.
	return yahooQuote(symbol)
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
