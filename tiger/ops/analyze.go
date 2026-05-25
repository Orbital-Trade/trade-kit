package ops

// Analyze — MCP tool: tiger_analyze
//
// Pulls OHLCV bars from Tiger kline/future_kline for up to three timeframes
// (1D, 1H, 15m) and computes RSI(14), MACD(12/26/9), BB(20,2), EMA(20/50/200).
// Returns a per-timeframe bias (BULLISH / BEARISH / NEUTRAL) and an overall
// multi-timeframe alignment score — replacing TradingView for single-symbol work.

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"time"
)

// bar is a single OHLCV candle.
type bar struct {
	Time   int64
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume float64
}

// TFAnalysis holds computed indicators for one timeframe.
type TFAnalysis struct {
	Timeframe string `json:"timeframe"`
	Bars      int    `json:"bars"`
	Err       string `json:"error,omitempty"`

	Price  float64 `json:"price"`
	Volume float64 `json:"volume"`

	RSI float64 `json:"rsi"`

	MACDLine   float64 `json:"macd_line"`
	MACDSignal float64 `json:"macd_signal"`
	MACDHist   float64 `json:"macd_hist"`

	BBUpper  float64 `json:"bb_upper"`
	BBMiddle float64 `json:"bb_middle"`
	BBLower  float64 `json:"bb_lower"`
	BBPct    float64 `json:"bb_pct"`

	EMA20  float64 `json:"ema20"`
	EMA50  float64 `json:"ema50"`
	EMA200 float64 `json:"ema200"`

	Bias  string `json:"bias"`  // "BULLISH", "BEARISH", "NEUTRAL", "INSUFFICIENT_DATA", "ERROR"
	Score int    `json:"score"` // -5..+5
}

// AnalyzeResult is returned by Analyze.
type AnalyzeResult struct {
	Symbol    string       `json:"symbol"`
	Market    string       `json:"market"`
	Currency  string       `json:"currency"`
	Timestamp string       `json:"timestamp"`
	IsFutures bool         `json:"is_futures"`
	Contract  string       `json:"contract,omitempty"`

	Timeframes []TFAnalysis `json:"timeframes"`
	Alignment  string       `json:"alignment"`  // "BULLISH", "BEARISH", "MIXED"
	BullCount  int          `json:"bull_count"`
	BearCount  int          `json:"bear_count"`
}

// Analyze fetches multi-timeframe data and computes technical indicators.
// Pass isFutures=true for futures (MNQ, MES, ES, NQ …); symbol is the root.
func Analyze(c Caller, symbol string, isFutures bool) (AnalyzeResult, error) {
	info := DetectMarket(symbol)

	res := AnalyzeResult{
		Symbol:    info.Symbol,
		Market:    info.Market,
		Currency:  info.Currency,
		Timestamp: time.Now().Format("2006-01-02 15:04 MST"),
		IsFutures: isFutures,
	}

	// Resolve futures contract code once.
	contractCode := ""
	if isFutures {
		var err error
		contractCode, err = c.ResolveFuturesContract(info.Symbol)
		if err != nil {
			return res, fmt.Errorf("analyze: resolve contract %s: %w", info.Symbol, err)
		}
		res.Contract = contractCode
	}

	// Timeframes: need 200 bars for daily (EMA200), 100 for intraday.
	type tfSpec struct {
		label  string
		period string
		limit  int
	}
	specs := []tfSpec{
		{"1D", "day", 200},
		{"1H", "60min", 100},
		{"15m", "15min", 100},
	}

	for _, s := range specs {
		bars, err := fetchBars(c, info, contractCode, s.period, s.limit, isFutures)
		if err != nil {
			res.Timeframes = append(res.Timeframes, TFAnalysis{Timeframe: s.label, Bias: "ERROR", Err: err.Error()})
			continue
		}
		if len(bars) < 30 {
			res.Timeframes = append(res.Timeframes, TFAnalysis{
				Timeframe: s.label,
				Bars:      len(bars),
				Bias:      "INSUFFICIENT_DATA",
			})
			continue
		}
		res.Timeframes = append(res.Timeframes, indicators(s.label, bars))
	}

	// Overall alignment.
	for _, tf := range res.Timeframes {
		switch tf.Bias {
		case "BULLISH":
			res.BullCount++
		case "BEARISH":
			res.BearCount++
		}
	}
	n := len(res.Timeframes)
	switch {
	case res.BullCount > n/2:
		res.Alignment = "BULLISH"
	case res.BearCount > n/2:
		res.Alignment = "BEARISH"
	default:
		res.Alignment = "MIXED"
	}

	return res, nil
}

// ── Data fetching ─────────────────────────────────────────────────────────────

func fetchBars(c Caller, info SymbolInfo, contractCode, period string, limit int, isFutures bool) ([]bar, error) {
	if isFutures {
		return fetchFutureBars(c, contractCode, period, limit)
	}
	return fetchStockBars(c, info.Symbol, period, limit)
}

func fetchStockBars(c Caller, symbol, period string, limit int) ([]bar, error) {
	// Try Tiger kline first (fastest, no rate limit for subscribed symbols).
	data, tigerErr := c.Call("kline", map[string]interface{}{
		"symbols": []string{symbol},
		"period":  period,
		"limit":   limit,
		"lang":    "en_US",
	})
	if tigerErr == nil {
		bars, parseErr := parseTigerKline(data)
		if parseErr == nil && len(bars) > 0 {
			return bars, nil
		}
	}
	// Fall back to Yahoo Finance (works for all US + SGX symbols, no quota).
	return yahooKline(symbol, period)
}

func parseTigerKline(data json.RawMessage) ([]bar, error) {
	type klineItem struct {
		Time   int64   `json:"time"`
		Open   float64 `json:"open"`
		High   float64 `json:"high"`
		Low    float64 `json:"low"`
		Close  float64 `json:"close"`
		Volume float64 `json:"volume"`
	}
	var items []klineItem
	if err := json.Unmarshal(data, &items); err != nil {
		var wrapped struct {
			Items []klineItem `json:"items"`
		}
		if err2 := json.Unmarshal(data, &wrapped); err2 != nil {
			return nil, fmt.Errorf("kline parse: %w", err)
		}
		items = wrapped.Items
	}
	out := make([]bar, 0, len(items))
	for _, it := range items {
		if it.Open == 0 && it.Close == 0 {
			continue
		}
		out = append(out, bar{Time: it.Time, Open: it.Open, High: it.High, Low: it.Low, Close: it.Close, Volume: it.Volume})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time < out[j].Time })
	return out, nil
}

// yahooIntervalRange maps Tiger period names to Yahoo interval + range params.
var yahooIntervalRange = map[string][2]string{
	"day":   {"1d", "2y"},
	"60min": {"60m", "60d"},
	"15min": {"15m", "5d"},
	"5min":  {"5m", "5d"},
}

func yahooKline(symbol, period string) ([]bar, error) {
	ir, ok := yahooIntervalRange[period]
	if !ok {
		return nil, fmt.Errorf("yahoo_kline: unsupported period %q", period)
	}
	interval, rng := ir[0], ir[1]

	ticker := symbol
	info := DetectMarket(symbol)
	if info.Market == "SG" {
		ticker = symbol + ".SI"
	}

	url := fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=%s&range=%s", ticker, interval, rng)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("yahoo_kline %s %s: build request: %w", symbol, period, err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("yahoo_kline %s %s: %w", symbol, period, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("yahoo_kline %s %s: read response: %w", symbol, period, err)
	}

	var yr struct {
		Chart struct {
			Result []struct {
				Timestamp  []int64 `json:"timestamp"`
				Indicators struct {
					Quote []struct {
						Open   []float64 `json:"open"`
						High   []float64 `json:"high"`
						Low    []float64 `json:"low"`
						Close  []float64 `json:"close"`
						Volume []float64 `json:"volume"`
					} `json:"quote"`
				} `json:"indicators"`
			} `json:"result"`
			Error interface{} `json:"error"`
		} `json:"chart"`
	}
	if err := json.Unmarshal(body, &yr); err != nil {
		return nil, fmt.Errorf("yahoo_kline parse %s %s: %w", symbol, period, err)
	}
	if len(yr.Chart.Result) == 0 || len(yr.Chart.Result[0].Indicators.Quote) == 0 {
		return nil, fmt.Errorf("yahoo_kline %s %s: no data", symbol, period)
	}
	res := yr.Chart.Result[0]
	q := res.Indicators.Quote[0]
	n := len(res.Timestamp)

	out := make([]bar, 0, n)
	for i := 0; i < n; i++ {
		if i >= len(q.Close) || q.Close[i] == 0 {
			continue
		}
		open, high, low, vol := 0.0, 0.0, 0.0, 0.0
		if i < len(q.Open) {
			open = q.Open[i]
		}
		if i < len(q.High) {
			high = q.High[i]
		}
		if i < len(q.Low) {
			low = q.Low[i]
		}
		if i < len(q.Volume) {
			vol = q.Volume[i]
		}
		out = append(out, bar{
			Time:   res.Timestamp[i] * 1000, // Yahoo gives seconds, convert to ms
			Open:   open,
			High:   high,
			Low:    low,
			Close:  q.Close[i],
			Volume: vol,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time < out[j].Time })
	return out, nil
}

func fetchFutureBars(c Caller, contractCode, period string, limit int) ([]bar, error) {
	periodMinutes := map[string]int{
		"1min": 1, "5min": 5, "15min": 15, "30min": 30, "60min": 60, "day": 1440,
	}
	mins := periodMinutes[period]
	if mins == 0 {
		mins = 15
	}
	now := time.Now()
	beginTime := now.Add(-time.Duration(limit*mins+mins*20) * time.Minute).UnixMilli()

	data, err := c.Call("future_kline", map[string]interface{}{
		"contract_codes": []string{contractCode},
		"period":         period,
		"begin_time":     beginTime,
		"end_time":       now.UnixMilli(),
		"limit":          limit,
		"lang":           "en_US",
	})
	if err != nil {
		return nil, fmt.Errorf("future_kline %s %s: %w", contractCode, period, err)
	}

	// Response: [{items:[{time,open,high,low,close,volume},...]}]
	var wrapper []struct {
		Items []struct {
			Time   int64   `json:"time"`
			Open   float64 `json:"open"`
			High   float64 `json:"high"`
			Low    float64 `json:"low"`
			Close  float64 `json:"close"`
			Volume float64 `json:"volume"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("future_kline parse %s %s: %w", contractCode, period, err)
	}

	var out []bar
	for _, w := range wrapper {
		for _, it := range w.Items {
			if it.Open == 0 && it.Close == 0 {
				continue
			}
			out = append(out, bar{Time: it.Time, Open: it.Open, High: it.High, Low: it.Low, Close: it.Close, Volume: it.Volume})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time < out[j].Time })
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

// ── Indicator computation ─────────────────────────────────────────────────────

func indicators(label string, bars []bar) TFAnalysis {
	closes := make([]float64, len(bars))
	for i, b := range bars {
		closes[i] = b.Close
	}
	last := bars[len(bars)-1]

	tfa := TFAnalysis{
		Timeframe: label,
		Bars:      len(bars),
		Price:     last.Close,
		Volume:    last.Volume,
	}

	tfa.RSI = rsi(closes, 14)
	tfa.MACDLine, tfa.MACDSignal, tfa.MACDHist = macd(closes, 12, 26, 9)
	tfa.BBUpper, tfa.BBMiddle, tfa.BBLower = bollinger(closes, 20, 2.0)
	if tfa.BBUpper > tfa.BBLower {
		tfa.BBPct = (tfa.Price - tfa.BBLower) / (tfa.BBUpper - tfa.BBLower)
	}
	tfa.EMA20 = ema(closes, 20)
	tfa.EMA50 = ema(closes, 50)
	tfa.EMA200 = ema(closes, 200)

	// Scoring: each signal contributes ±1. Range: -5..+5.
	score := 0

	// RSI: overbought (>70) and oversold (<30) are NOT directional confirmations —
	// they signal mean-reversion risk. Only the 45–70 band counts as trend evidence.
	switch {
	case tfa.RSI >= 70:
		// Overbought: no bullish credit; momentum is extended, not confirmed.
	case tfa.RSI > 55:
		score++
	case tfa.RSI <= 30:
		// Oversold: no bearish credit; potential bounce, not continuation.
	case tfa.RSI < 45:
		score--
	}

	// MACD histogram: direction of momentum change is the most sensitive signal.
	if tfa.MACDHist > 0 {
		score++
	} else if tfa.MACDHist < 0 {
		score--
	}

	// Price vs EMA20 (short-term structure)
	if tfa.Price > tfa.EMA20 && tfa.EMA20 > 0 {
		score++
	} else if tfa.Price < tfa.EMA20 && tfa.EMA20 > 0 {
		score--
	}

	// Price vs EMA50 (medium-term structure)
	if tfa.Price > tfa.EMA50 && tfa.EMA50 > 0 {
		score++
	} else if tfa.Price < tfa.EMA50 && tfa.EMA50 > 0 {
		score--
	}

	// Bollinger %B: >1.0 means price pierced the upper band — overextension,
	// not strength. <0 means below lower band — also overextension downward.
	switch {
	case tfa.BBPct > 1.0:
		score-- // extended above upper band: mean-reversion risk
	case tfa.BBPct > 0.5:
		score++
	case tfa.BBPct < 0:
		score++ // extended below lower band: potential bounce
	default:
		score--
	}

	tfa.Score = score
	switch {
	case score >= 3:
		tfa.Bias = "BULLISH"
	case score <= -3:
		tfa.Bias = "BEARISH"
	default:
		tfa.Bias = "NEUTRAL"
	}

	return tfa
}

// ── Math helpers ──────────────────────────────────────────────────────────────

// emaValues computes EMA for all positions, seeded with SMA of first `period` bars.
// Returns nil if len(closes) < period.
func emaValues(closes []float64, period int) []float64 {
	if len(closes) < period {
		return nil
	}
	k := 2.0 / float64(period+1)
	out := make([]float64, len(closes))

	// Seed: SMA of first period values.
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += closes[i]
	}
	out[period-1] = sum / float64(period)

	for i := period; i < len(closes); i++ {
		out[i] = closes[i]*k + out[i-1]*(1-k)
	}
	return out
}

// ema returns the final EMA value for the given period.
func ema(closes []float64, period int) float64 {
	v := emaValues(closes, period)
	if v == nil {
		return 0
	}
	return v[len(v)-1]
}

// rsi computes RSI(period) using Wilder's smoothing on the final `period` bars.
func rsi(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50
	}
	// Use the last period+1 bars for simplicity (simple RSI, not Wilder's smoothing).
	n := len(closes)
	start := n - period - 1
	var gains, losses float64
	for i := start + 1; i <= start+period; i++ {
		d := closes[i] - closes[i-1]
		if d > 0 {
			gains += d
		} else {
			losses -= d
		}
	}
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)
	if avgLoss == 0 {
		return 100
	}
	rs := avgGain / avgLoss
	return 100 - 100/(1+rs)
}

// macd computes MACD line, signal line, and histogram.
func macd(closes []float64, fast, slow, signalPeriod int) (line, sig, hist float64) {
	ema12 := emaValues(closes, fast)
	ema26 := emaValues(closes, slow)
	if ema12 == nil || ema26 == nil {
		return 0, 0, 0
	}

	// MACD values start at index slow-1 (first valid EMA26).
	macdVals := make([]float64, len(closes)-slow+1)
	for i := slow - 1; i < len(closes); i++ {
		macdVals[i-(slow-1)] = ema12[i] - ema26[i]
	}

	sigVals := emaValues(macdVals, signalPeriod)
	if sigVals == nil {
		return macdVals[len(macdVals)-1], 0, 0
	}

	ml := macdVals[len(macdVals)-1]
	sl := sigVals[len(sigVals)-1]
	return ml, sl, ml - sl
}

// bollinger computes Bollinger Bands over the last `period` bars.
func bollinger(closes []float64, period int, mult float64) (upper, middle, lower float64) {
	if len(closes) < period {
		return 0, 0, 0
	}
	recent := closes[len(closes)-period:]
	var sum float64
	for _, v := range recent {
		sum += v
	}
	sma := sum / float64(period)

	var variance float64
	for _, v := range recent {
		d := v - sma
		variance += d * d
	}
	stdDev := math.Sqrt(variance / float64(period))
	return sma + mult*stdDev, sma, sma - mult*stdDev
}
