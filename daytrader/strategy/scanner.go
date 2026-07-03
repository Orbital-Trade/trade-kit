package strategy

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

// FetchSetup returns price, prev close, gap %, avg daily volume, and RVOL.
// RVOL = today's accumulated volume / expected volume at this hour (based on ADV).
func FetchSetup(symbol string) (Setup, error) {
	// Fetch daily bars for ADV + prev close
	dailyBody, err := get(fmt.Sprintf(
		"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=10d", symbol))
	if err != nil {
		return Setup{}, err
	}

	var daily struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice   float64 `json:"regularMarketPrice"`
					ChartPreviousClose   float64 `json:"chartPreviousClose"`
					RegularMarketVolume  float64 `json:"regularMarketVolume"`
				} `json:"meta"`
				Indicators struct {
					Quote []struct {
						Volume []float64 `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(dailyBody, &daily); err != nil {
		return Setup{}, fmt.Errorf("parse daily: %w", err)
	}
	if len(daily.Chart.Result) == 0 {
		return Setup{}, fmt.Errorf("no data for %s", symbol)
	}
	r := daily.Chart.Result[0]
	price := r.Meta.RegularMarketPrice
	prevClose := r.Meta.ChartPreviousClose
	todayVol := r.Meta.RegularMarketVolume

	var gapPct float64
	if prevClose > 0 {
		gapPct = (price - prevClose) / prevClose * 100
	}

	// ADV from last 10 daily bars (excluding today)
	var sumVol float64
	var n int
	if len(r.Indicators.Quote) > 0 {
		vols := r.Indicators.Quote[0].Volume
		// skip last entry (today) for ADV calculation
		end := len(vols) - 1
		if end < 0 {
			end = 0
		}
		for _, v := range vols[:end] {
			if v > 0 {
				sumVol += v
				n++
			}
		}
	}
	var avgVol float64
	if n > 0 {
		avgVol = sumVol / float64(n)
	}

	// RVOL: compare today's volume to expected volume at this time of day.
	// US market is 6.5 hours (390 min). Scale ADV by fraction of session elapsed.
	rvol := computeRVOL(todayVol, avgVol)

	return Setup{
		Symbol:    symbol,
		Price:     price,
		PrevClose: prevClose,
		GapPct:    gapPct,
		AvgVolume: avgVol,
		RVOL:      rvol,
	}, nil
}

// computeRVOL calculates relative volume: today's volume vs expected volume
// at this point in the trading session, based on historical ADV.
// Returns 0 if market is not open or ADV is unknown.
func computeRVOL(todayVol, adv float64) float64 {
	if adv <= 0 || todayVol <= 0 {
		return 0
	}
	etLoc, err := time.LoadLocation("America/New_York")
	if err != nil {
		etLoc = time.FixedZone("EST", -5*60*60)
	}
	et := time.Now().In(etLoc)
	h, m, _ := et.Clock()
	minsIntoSession := h*60 + m - (9*60 + 30) // minutes since open
	if minsIntoSession <= 0 {
		return 0
	}
	const sessionMins = 390.0 // 6.5 hours
	fractionElapsed := float64(minsIntoSession) / sessionMins
	if fractionElapsed > 1 {
		fractionElapsed = 1
	}
	expectedVol := adv * fractionElapsed
	return todayVol / expectedVol
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
