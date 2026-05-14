package strategy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

func FetchQuote(symbol string) (Quote, error) {
	body, err := get(fmt.Sprintf(
		"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", symbol))
	if err != nil {
		return Quote{}, err
	}

	var resp struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					ChartPreviousClose float64 `json:"chartPreviousClose"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return Quote{}, fmt.Errorf("parse %s: %w", symbol, err)
	}
	if len(resp.Chart.Result) == 0 {
		return Quote{}, fmt.Errorf("no data for %s", symbol)
	}
	r := resp.Chart.Result[0]
	price := r.Meta.RegularMarketPrice
	prev := r.Meta.ChartPreviousClose

	var changePct float64
	if prev > 0 {
		changePct = (price - prev) / prev * 100
	}
	return Quote{
		Symbol:    symbol,
		Price:     price,
		PrevClose: prev,
		ChangePct: changePct,
	}, nil
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
