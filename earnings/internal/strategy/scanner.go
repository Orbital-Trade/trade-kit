package strategy

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"sync"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36"

// Yahoo Finance session state — crumb is obtained once per process.
var (
	sessionOnce   sync.Once
	sessionClient *http.Client
	sessionCrumb  string
)

func ensureSession() {
	sessionOnce.Do(func() {
		jar, err := cookiejar.New(nil)
		if err != nil {
			return // cookiejar.New never fails in practice; if it does, skip crumb
		}
		sessionClient = &http.Client{Jar: jar, Timeout: 15 * time.Second}

		// Step 1: visit Yahoo Finance to receive session cookies.
		req, err := http.NewRequest("GET", "https://finance.yahoo.com", nil)
		if err != nil {
			return
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		if resp, err := sessionClient.Do(req); err == nil {
			resp.Body.Close()
		}

		// Step 2: fetch the crumb using the session cookies.
		req, err = http.NewRequest("GET", "https://query2.finance.yahoo.com/v1/test/getcrumb", nil)
		if err != nil {
			return
		}
		req.Header.Set("User-Agent", userAgent)
		req.Header.Set("Accept", "application/json")
		if resp, err := sessionClient.Do(req); err == nil {
			b, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return
			}
			crumb := strings.TrimSpace(string(b))
			// Crumb is valid if it doesn't look like a JSON error
			if !strings.Contains(crumb, "Unauthorized") && len(crumb) > 0 {
				sessionCrumb = crumb
			}
		}
	})
}

// FetchSetup fetches price data and earnings date for symbol from Yahoo Finance.
// earningsOverride is used when the config provides a manual date (zero = use API).
func FetchSetup(symbol string, earningsOverride time.Time) (Setup, error) {
	price, avgVol, run5d, err := fetchQuote(symbol)
	if err != nil {
		return Setup{}, fmt.Errorf("%s: quote: %w", symbol, err)
	}

	earningsDate := earningsOverride
	if earningsDate.IsZero() {
		// Try Yahoo Finance calendarEvents (requires crumb session).
		earningsDate, _ = fetchEarningsDate(symbol)
	}

	daysTo := -1
	if !earningsDate.IsZero() {
		now := time.Now().UTC().Truncate(24 * time.Hour)
		ed := earningsDate.UTC().Truncate(24 * time.Hour)
		diff := ed.Sub(now)
		daysTo = int(math.Round(diff.Hours() / 24))
	}

	return Setup{
		Symbol:         symbol,
		Price:          price,
		AvgVolume:      avgVol,
		Run5d:          run5d,
		EarningsDate:   earningsDate,
		DaysToEarnings: daysTo,
	}, nil
}

// fetchQuote returns current price, average daily volume, and 5-day price change %.
// Uses Yahoo Finance v8 chart endpoint — no auth required.
func fetchQuote(symbol string) (price, avgVol, run5d float64, err error) {
	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=10d", symbol)
	body, err := plainGet(url)
	if err != nil {
		return 0, 0, 0, err
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
		return 0, 0, 0, fmt.Errorf("parse chart: %w", err)
	}
	if len(resp.Chart.Result) == 0 {
		return 0, 0, 0, fmt.Errorf("no chart data for %s", symbol)
	}
	r := resp.Chart.Result[0]
	price = r.Meta.RegularMarketPrice

	if len(r.Indicators.Quote) > 0 {
		closes := compact(r.Indicators.Quote[0].Close)
		vols := compact(r.Indicators.Quote[0].Volume)

		var sumVol float64
		for _, v := range vols {
			sumVol += v
		}
		if len(vols) > 0 {
			avgVol = sumVol / float64(len(vols))
		}
		if len(closes) >= 2 {
			last := closes[len(closes)-1]
			base := closes[maxIdx(0, len(closes)-6)]
			if base > 0 {
				run5d = (last - base) / base * 100
			}
		}
	}
	return price, avgVol, run5d, nil
}

// fetchEarningsDate returns the next reported earnings date from Yahoo Finance calendarEvents.
// Requires a valid crumb session.
func fetchEarningsDate(symbol string) (time.Time, error) {
	ensureSession()
	if sessionCrumb == "" {
		return time.Time{}, fmt.Errorf("no crumb — use earnings_dates config override")
	}

	url := fmt.Sprintf(
		"https://query2.finance.yahoo.com/v10/finance/quoteSummary/%s?modules=calendarEvents&crumb=%s",
		symbol, sessionCrumb,
	)
	body, err := sessionGet(url)
	if err != nil {
		return time.Time{}, err
	}

	var resp struct {
		QuoteSummary struct {
			Result []struct {
				CalendarEvents struct {
					Earnings struct {
						EarningsDate []struct {
							Raw int64 `json:"raw"`
						} `json:"earningsDate"`
					} `json:"earnings"`
				} `json:"calendarEvents"`
			} `json:"result"`
		} `json:"quoteSummary"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return time.Time{}, fmt.Errorf("parse calendar: %w", err)
	}
	results := resp.QuoteSummary.Result
	if len(results) == 0 {
		return time.Time{}, fmt.Errorf("no result for %s", symbol)
	}
	dates := results[0].CalendarEvents.Earnings.EarningsDate
	if len(dates) == 0 {
		return time.Time{}, fmt.Errorf("no earnings date for %s", symbol)
	}
	return time.Unix(dates[0].Raw, 0).UTC(), nil
}

// plainGet makes an unauthenticated GET request (for chart endpoints).
func plainGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// sessionGet makes a GET request using the crumb session (for authenticated endpoints).
func sessionGet(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := sessionClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func compact(vals []float64) []float64 {
	var out []float64
	for _, v := range vals {
		if v != 0 {
			out = append(out, v)
		}
	}
	return out
}

func maxIdx(a, b int) int {
	if a > b {
		return a
	}
	return b
}
