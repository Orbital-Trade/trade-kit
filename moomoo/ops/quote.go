package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// NormalizeSymbol converts user input to a Moomoo code (SG.Z74, US.AAPL, HK.00700).
func NormalizeSymbol(sym string) string {
	sym = strings.ToUpper(strings.TrimSpace(sym))
	// Already prefixed
	if matched, _ := regexp.MatchString(`^(US|SG|HK|CN)\.`, sym); matched {
		return sym
	}
	// .SI suffix → SGX
	if strings.HasSuffix(sym, ".SI") {
		return "SG." + sym[:len(sym)-3]
	}
	// .HK suffix → HK
	if strings.HasSuffix(sym, ".HK") {
		return "HK." + sym[:len(sym)-3]
	}
	// Pure digits → SGX
	matched, _ := regexp.MatchString(`^\d+$`, sym)
	if matched {
		return "SG." + sym
	}
	// SGX mixed pattern: 1-3 alpha + 1-2 digits + optional alpha (Z74, G13, D05, A17U)
	matched, _ = regexp.MatchString(`^[A-Z]{1,3}[0-9]{1,2}[A-Z]?$`, sym)
	if matched {
		return "SG." + sym
	}
	// Default: US equity
	return "US." + sym
}

// toYahoo converts a Moomoo code or raw symbol to Yahoo Finance ticker.
func toYahoo(sym string) string {
	sym = strings.ToUpper(strings.TrimSpace(sym))
	if strings.HasPrefix(sym, "SG.") {
		return sym[3:] + ".SI"
	}
	if strings.HasPrefix(sym, "HK.") {
		base := strings.TrimLeft(sym[3:], "0")
		if base == "" {
			base = "0"
		}
		for len(base) < 4 {
			base = "0" + base
		}
		return base + ".HK"
	}
	if strings.HasPrefix(sym, "US.") {
		return sym[3:]
	}
	// Raw input — apply same routing as NormalizeSymbol
	if strings.HasSuffix(sym, ".SI") || strings.HasSuffix(sym, ".HK") {
		return sym
	}
	matched, _ := regexp.MatchString(`^\d+$`, sym)
	if matched {
		return sym + ".SI"
	}
	matched, _ = regexp.MatchString(`^[A-Z]{1,3}[0-9]{1,2}[A-Z]?$`, sym)
	if matched {
		return sym + ".SI"
	}
	return sym
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// GetQuote fetches a real-time price snapshot from Yahoo Finance.
// Works for all markets (SGX, US, HK) — Moomoo OpenD has no SGX data permission.
func GetQuote(sym string) (Quote, error) {
	mCode := NormalizeSymbol(sym)
	yahoo := toYahoo(mCode)

	url := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", yahoo)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Quote{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return Quote{}, fmt.Errorf("Yahoo Finance request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Quote{}, fmt.Errorf("read response: %w", err)
	}

	var payload struct {
		Chart struct {
			Result []struct {
				Meta struct {
					Currency              string  `json:"currency"`
					RegularMarketPrice    float64 `json:"regularMarketPrice"`
					RegularMarketOpen     float64 `json:"regularMarketOpen"`
					RegularMarketDayHigh  float64 `json:"regularMarketDayHigh"`
					RegularMarketDayLow   float64 `json:"regularMarketDayLow"`
					RegularMarketVolume   int64   `json:"regularMarketVolume"`
					ChartPreviousClose    float64 `json:"chartPreviousClose"`
					PreviousClose         float64 `json:"previousClose"`
				} `json:"meta"`
			} `json:"result"`
			Error *struct{ Code, Description string } `json:"error"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return Quote{}, fmt.Errorf("parse Yahoo response: %w", err)
	}
	if payload.Chart.Error != nil {
		return Quote{}, fmt.Errorf("Yahoo error: %s", payload.Chart.Error.Description)
	}
	if len(payload.Chart.Result) == 0 {
		return Quote{}, fmt.Errorf("no data for %s (Yahoo: %s)", mCode, yahoo)
	}

	meta := payload.Chart.Result[0].Meta
	prev := meta.ChartPreviousClose
	if prev == 0 {
		prev = meta.PreviousClose
	}
	var chg float64
	if prev > 0 {
		chg = (meta.RegularMarketPrice - prev) / prev * 100
	}

	return Quote{
		Symbol:    mCode,
		Yahoo:     yahoo,
		Price:     meta.RegularMarketPrice,
		Open:      meta.RegularMarketOpen,
		High:      meta.RegularMarketDayHigh,
		Low:       meta.RegularMarketDayLow,
		PrevClose: prev,
		Volume:    meta.RegularMarketVolume,
		ChangePct: chg,
		Currency:  meta.Currency,
	}, nil
}
