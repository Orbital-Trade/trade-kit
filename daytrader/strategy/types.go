package strategy

// Config holds tunable parameters for the gap-up day trade strategy.
type Config struct {
	GapMinPct            float64  `json:"gap_min_pct"`
	GapMaxPct            float64  `json:"gap_max_pct"`
	EntryWindowStartMin  int      `json:"entry_window_start_min"`
	EntryWindowEndMin    int      `json:"entry_window_end_min"`
	ExitByMin            int      `json:"exit_by_min"`
	StopPct              float64  `json:"stop_pct"`
	RRMin                float64  `json:"rr_min"`
	Budget               float64  `json:"budget"`
	MinADV               float64  `json:"min_adv"`
	MinPrice             float64  `json:"min_price"`
	ScanIntervalSec      int      `json:"scan_interval_sec"`
	Watchlist            []string `json:"watchlist"`

	// Earnings scalp mode — tighter, both directions, RVOL filter
	EarningsMode bool    `json:"earnings_mode"`
	RVOLMin      float64 `json:"rvol_min"`      // min relative volume (e.g. 2.0 = 2x normal)
	AllowShort   bool    `json:"allow_short"`   // trade gap-downs on earnings misses
}

// Setup is raw market data for one symbol.
type Setup struct {
	Symbol    string
	Price     float64
	PrevClose float64
	GapPct    float64 // % gap from prev close to current (negative = gap down)
	AvgVolume float64
	RVOL      float64 // current session volume / expected volume at this time of day
}

// Signal is the bot's decision for one symbol.
type Signal struct {
	Symbol      string
	Action      string // "enter" | "exit_time" | "skip"
	Reason      string
	Qty         int
	LimitPrice  float64
	StopPrice   float64
	TargetPrice float64
	Setup       Setup
}
