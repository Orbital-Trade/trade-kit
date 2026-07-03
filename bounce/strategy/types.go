package strategy

// Config holds tunable parameters for the RSI bounce strategy.
type Config struct {
	RSIEntry        float64  `json:"rsi_entry"`         // enter when RSI drops below this
	RSIExit         float64  `json:"rsi_exit"`          // exit when RSI recovers to this
	StopPct         float64  `json:"stop_pct"`          // hard stop % below entry
	MaxHoldDays     int      `json:"max_hold_days"`     // forced exit after N trading days
	Budget          float64  `json:"budget"`            // max USD per trade
	MinADV          float64  `json:"min_adv"`           // min avg daily volume
	MinPrice        float64  `json:"min_price"`         // min stock price
	ScanIntervalSec int      `json:"scan_interval_sec"` // seconds between run-mode scans
	Watchlist       []string `json:"watchlist"`
}

// Setup is raw market data for one symbol.
type Setup struct {
	Symbol    string
	Price     float64
	RSI       float64 // 14-period daily RSI
	AvgVolume float64
}

// Signal is the bot's decision for one symbol.
type Signal struct {
	Symbol     string
	Action     string // "enter" | "exit" | "skip"
	Reason     string
	Qty        int
	LimitPrice float64
	StopPrice  float64
	Setup      Setup
}
