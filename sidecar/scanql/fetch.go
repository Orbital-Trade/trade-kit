package scanql

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

// FetchData computes all requested indicators for a symbol.
// Returns a map of field name → value (e.g. "rsi" → 18.3, "price" → 185.50).
func FetchData(symbol string, specs []FetchSpec) (map[string]float64, error) {
	data := make(map[string]float64)

	for _, spec := range specs {
		switch spec.Name {
		case "quote":
			if err := fetchQuote(symbol, data); err != nil {
				return nil, fmt.Errorf("quote %s: %w", symbol, err)
			}
		case "rsi":
			period := 14
			if len(spec.Params) > 0 {
				period = int(spec.Params[0])
			}
			if err := fetchRSI(symbol, period, data); err != nil {
				return nil, fmt.Errorf("rsi %s: %w", symbol, err)
			}
		case "rvol":
			if err := fetchRVOL(symbol, data); err != nil {
				return nil, fmt.Errorf("rvol %s: %w", symbol, err)
			}
		case "ema":
			period := 20
			if len(spec.Params) > 0 {
				period = int(spec.Params[0])
			}
			if err := fetchEMA(symbol, period, data); err != nil {
				return nil, fmt.Errorf("ema %s: %w", symbol, err)
			}
		default:
			// Unknown indicators are skipped — they'll fail at eval time if referenced in WHERE.
		}
	}
	return data, nil
}

func fetchQuote(symbol string, data map[string]float64) error {
	body, err := yahooGet(fmt.Sprintf(
		"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", symbol))
	if err != nil {
		return err
	}

	var yr struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Price     float64 `json:"regularMarketPrice"`
					PrevClose float64 `json:"chartPreviousClose"`
					Open      float64 `json:"regularMarketOpen"`
					High      float64 `json:"regularMarketDayHigh"`
					Low       float64 `json:"regularMarketDayLow"`
					Volume    float64 `json:"regularMarketVolume"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if len(yr.Chart.Result) == 0 {
		return fmt.Errorf("no data")
	}
	m := yr.Chart.Result[0].Meta

	data["price"] = m.Price
	data["prev_close"] = m.PrevClose
	data["open"] = m.Open
	data["high"] = m.High
	data["low"] = m.Low
	data["volume"] = m.Volume

	if m.PrevClose > 0 {
		data["change_pct"] = (m.Price - m.PrevClose) / m.PrevClose * 100
		data["gap_pct"] = data["change_pct"]
	}
	return nil
}

func fetchRSI(symbol string, period int, data map[string]float64) error {
	days := period*2 + 10
	body, err := yahooGet(fmt.Sprintf(
		"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=%dd", symbol, days))
	if err != nil {
		return err
	}

	var yr struct {
		Chart struct {
			Result []struct {
				Indicators struct {
					Quote []struct {
						Close []interface{} `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if len(yr.Chart.Result) == 0 || len(yr.Chart.Result[0].Indicators.Quote) == 0 {
		return fmt.Errorf("no data")
	}

	rawCloses := yr.Chart.Result[0].Indicators.Quote[0].Close
	closes := make([]float64, 0, len(rawCloses))
	for _, v := range rawCloses {
		if f, ok := v.(float64); ok {
			closes = append(closes, f)
		}
	}

	if len(closes) < period+1 {
		return fmt.Errorf("not enough data for RSI(%d): got %d bars", period, len(closes))
	}

	data["rsi"] = computeRSI(closes, period)
	return nil
}

func fetchRVOL(symbol string, data map[string]float64) error {
	body, err := yahooGet(fmt.Sprintf(
		"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=10d", symbol))
	if err != nil {
		return err
	}

	var yr struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Volume float64 `json:"regularMarketVolume"`
				} `json:"meta"`
				Indicators struct {
					Quote []struct {
						Volume []interface{} `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if len(yr.Chart.Result) == 0 {
		return fmt.Errorf("no data")
	}
	r := yr.Chart.Result[0]
	todayVol := r.Meta.Volume

	var vols []float64
	if len(r.Indicators.Quote) > 0 {
		for _, v := range r.Indicators.Quote[0].Volume {
			if f, ok := v.(float64); ok && f > 0 {
				vols = append(vols, f)
			}
		}
	}
	if len(vols) == 0 {
		data["rvol"] = 1.0
		return nil
	}

	var sum float64
	for _, v := range vols {
		sum += v
	}
	adv := sum / float64(len(vols))
	if adv > 0 {
		data["rvol"] = todayVol / adv
	} else {
		data["rvol"] = 1.0
	}
	return nil
}

func fetchEMA(symbol string, period int, data map[string]float64) error {
	days := period*2 + 10
	body, err := yahooGet(fmt.Sprintf(
		"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=%dd", symbol, days))
	if err != nil {
		return err
	}

	var yr struct {
		Chart struct {
			Result []struct {
				Indicators struct {
					Quote []struct {
						Close []interface{} `json:"close"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return fmt.Errorf("parse: %w", err)
	}
	if len(yr.Chart.Result) == 0 || len(yr.Chart.Result[0].Indicators.Quote) == 0 {
		return fmt.Errorf("no data")
	}

	rawCloses := yr.Chart.Result[0].Indicators.Quote[0].Close
	closes := make([]float64, 0, len(rawCloses))
	for _, v := range rawCloses {
		if f, ok := v.(float64); ok {
			closes = append(closes, f)
		}
	}

	if len(closes) < period {
		return fmt.Errorf("not enough data for EMA(%d)", period)
	}

	ema := computeEMA(closes, period)
	key := fmt.Sprintf("ema_%d", period)
	data[key] = ema
	return nil
}

// ── Math helpers ────────────────────────────────────────────────────────────

func computeRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50 // neutral if insufficient data
	}

	var gainSum, lossSum float64
	for i := 1; i <= period; i++ {
		diff := closes[len(closes)-period-1+i] - closes[len(closes)-period-1+i-1]
		if diff > 0 {
			gainSum += diff
		} else {
			lossSum += math.Abs(diff)
		}
	}

	avgGain := gainSum / float64(period)
	avgLoss := lossSum / float64(period)

	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func computeEMA(closes []float64, period int) float64 {
	if len(closes) < period {
		return closes[len(closes)-1]
	}
	k := 2.0 / float64(period+1)
	var sum float64
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	ema := sum / float64(period)
	for i := period; i < len(closes); i++ {
		ema = closes[i]*k + ema*(1-k)
	}
	return ema
}

func yahooGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
