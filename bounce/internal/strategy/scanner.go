package strategy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

// FetchSetup returns current price, RSI(14), and avg volume for symbol.
// Uses Yahoo Finance v8 chart (no auth required).
func FetchSetup(symbol string) (Setup, error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=30d", symbol)
	body, err := get(url)
	if err != nil {
		return Setup{}, err
	}

	var resp struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
				} `json:"meta"`
				Indicators struct {
					Quote []struct {
						Close  []float64 `json:"close"`
						Volume []float64 `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Setup{}, fmt.Errorf("parse: %w", err)
	}
	if len(resp.Chart.Result) == 0 {
		return Setup{}, fmt.Errorf("no data for %s", symbol)
	}
	r := resp.Chart.Result[0]
	price := r.Meta.RegularMarketPrice

	var closes, vols []float64
	if len(r.Indicators.Quote) > 0 {
		for _, v := range r.Indicators.Quote[0].Close {
			if v != 0 {
				closes = append(closes, v)
			}
		}
		for _, v := range r.Indicators.Quote[0].Volume {
			if v != 0 {
				vols = append(vols, v)
			}
		}
	}

	rsi := computeRSI(closes, 14)

	var sumVol float64
	for _, v := range vols {
		sumVol += v
	}
	var avgVol float64
	if len(vols) > 0 {
		avgVol = sumVol / float64(len(vols))
	}

	return Setup{Symbol: symbol, Price: price, RSI: rsi, AvgVolume: avgVol}, nil
}

// computeRSI calculates the N-period RSI from a slice of closing prices.
// Returns 50 (neutral) if there is insufficient data.
func computeRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50
	}
	recent := closes[len(closes)-period-1:]
	var gains, losses float64
	for i := 1; i <= period; i++ {
		diff := recent[i] - recent[i-1]
		if diff > 0 {
			gains += diff
		} else {
			losses -= diff
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func get(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
