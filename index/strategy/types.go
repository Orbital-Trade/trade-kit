package strategy

type Config struct {
	QQQLongThreshold  float64 `json:"qqq_long_threshold"`
	QQQShortThreshold float64 `json:"qqq_short_threshold"`
	VIXMax            float64 `json:"vix_max"`
	VIXSpikeMin       float64 `json:"vix_spike_min"`
	Budget            float64 `json:"budget"`
	TQQQShares        int     `json:"tqqq_shares"`
	SQQQShares        int     `json:"sqqq_shares"`
	StopPct           float64 `json:"stop_pct"`
	TargetPct         float64 `json:"target_pct"`
	GracePeriodMin    int     `json:"grace_period_min"`
	ExitByMin         int     `json:"exit_by_min"`
	ScanIntervalSec   int     `json:"scan_interval_sec"`
}

type Quote struct {
	Symbol    string
	Price     float64
	PrevClose float64
	ChangePct float64
}

type Direction int

const (
	Flat  Direction = 0
	Long  Direction = 1  // TQQQ
	Short Direction = -1 // SQQQ
)

type Signal struct {
	Direction Direction
	Symbol    string
	Shares    int
	Price     float64
	Reason    string
}

type Position struct {
	Symbol    string
	Shares    int
	EntryPrice float64
	Direction Direction
}

func (p *Position) PnL(currentPrice float64) float64 {
	return (currentPrice - p.EntryPrice) * float64(p.Shares)
}

func (p *Position) PnLPct(currentPrice float64) float64 {
	return (currentPrice - p.EntryPrice) / p.EntryPrice * 100
}
