package scanql

import (
	"testing"
	"time"
)

func TestParseBasic(t *testing.T) {
	input := `SCAN rsi_bounce
  EVERY 300s
  SYMBOLS AAPL, NVDA, TSLA
  FETCH quote, rsi(14)
  WHERE rsi <= 25
    AND volume >= 500000
  ENTER LONG
    STOP 5%
    TARGET 3R
    BUDGET 150`

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if plan.Name != "rsi_bounce" {
		t.Errorf("Name = %q, want rsi_bounce", plan.Name)
	}
	if plan.Interval != 300*time.Second {
		t.Errorf("Interval = %v, want 300s", plan.Interval)
	}
	if len(plan.Symbols) != 3 {
		t.Errorf("Symbols = %v, want 3", plan.Symbols)
	}
	if plan.Symbols[0] != "AAPL" {
		t.Errorf("Symbols[0] = %q, want AAPL", plan.Symbols[0])
	}
	if len(plan.Fetch) != 2 {
		t.Errorf("Fetch = %d, want 2", len(plan.Fetch))
	}
	if plan.Fetch[1].Name != "rsi" || plan.Fetch[1].Params[0] != 14 {
		t.Errorf("Fetch[1] = %+v, want rsi(14)", plan.Fetch[1])
	}
	if len(plan.Where) != 2 {
		t.Errorf("Where = %d conditions, want 2", len(plan.Where))
	}
	if plan.Where[0].Field != "rsi" || plan.Where[0].Op != "<=" || plan.Where[0].Value != 25 {
		t.Errorf("Where[0] = %+v, want rsi <= 25", plan.Where[0])
	}
	if plan.Action.Side != "long" {
		t.Errorf("Side = %q, want long", plan.Action.Side)
	}
	if plan.Action.StopPct != 5 {
		t.Errorf("StopPct = %v, want 5", plan.Action.StopPct)
	}
	if plan.Action.TargetR != 3 {
		t.Errorf("TargetR = %v, want 3", plan.Action.TargetR)
	}
	if plan.Action.Budget != 150 {
		t.Errorf("Budget = %v, want 150", plan.Action.Budget)
	}
}

func TestParseBetween(t *testing.T) {
	input := `SCAN gap
  EVERY 60s
  SYMBOLS AAPL
  FETCH quote
  WHERE gap_pct BETWEEN 3.0 AND 20.0
  ENTER LONG
    STOP 2%
    BUDGET 200`

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(plan.Where) != 1 {
		t.Fatalf("Where = %d, want 1", len(plan.Where))
	}
	c := plan.Where[0]
	if c.Op != "between" || c.Value != 3.0 || c.ValueEnd != 20.0 {
		t.Errorf("Between = %+v, want 3.0..20.0", c)
	}
}

func TestParseMinutes(t *testing.T) {
	input := `SCAN test
  EVERY 5m
  SYMBOLS AAPL
  FETCH quote
  WHERE price >= 100
  ENTER LONG
    STOP 3%
    BUDGET 200`

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if plan.Interval != 5*time.Minute {
		t.Errorf("Interval = %v, want 5m", plan.Interval)
	}
}

func TestParseExitWhen(t *testing.T) {
	input := `SCAN bounce
  EVERY 300s
  SYMBOLS AAPL
  FETCH quote, rsi(14)
  WHERE rsi <= 20
  ENTER LONG
    STOP 5%
    EXIT WHEN rsi >= 50
    HOLD MAX 5 DAYS
    BUDGET 150`

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if plan.Exit == nil {
		t.Fatal("Exit is nil")
	}
	if plan.Exit.WhenField != "rsi" || plan.Exit.WhenOp != ">=" || plan.Exit.WhenValue != 50 {
		t.Errorf("Exit when = %+v, want rsi >= 50", plan.Exit)
	}
	if plan.Exit.MaxHold != 5 {
		t.Errorf("MaxHold = %d, want 5", plan.Exit.MaxHold)
	}
}

func TestParseShort(t *testing.T) {
	input := `SCAN short_test
  EVERY 30s
  SYMBOLS QQQ
  FETCH quote
  WHERE change_pct <= -0.3
  ENTER SHORT SQQQ 4 SHARES
    STOP 5%
    TARGET 6%
    EXIT BY 12:30 ET`

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if plan.Action.Side != "short" {
		t.Errorf("Side = %q, want short", plan.Action.Side)
	}
	if plan.Action.Symbol != "SQQQ" {
		t.Errorf("Symbol = %q, want SQQQ", plan.Action.Symbol)
	}
	if plan.Action.Shares != 4 {
		t.Errorf("Shares = %d, want 4", plan.Action.Shares)
	}
	if plan.Action.TargetPct != 6 {
		t.Errorf("TargetPct = %v, want 6", plan.Action.TargetPct)
	}
	if plan.Exit == nil || plan.Exit.ExitBy != "12:30" {
		t.Errorf("ExitBy = %v, want 12:30", plan.Exit)
	}
}

func TestParseMACDFetch(t *testing.T) {
	input := `SCAN macd
  EVERY 60s
  SYMBOLS AAPL
  FETCH quote, macd(12, 26, 9)
  WHERE macd_histogram >= 0
  ENTER LONG
    STOP 3%
    BUDGET 300`

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(plan.Fetch) != 2 {
		t.Fatalf("Fetch = %d, want 2", len(plan.Fetch))
	}
	macd := plan.Fetch[1]
	if macd.Name != "macd" {
		t.Errorf("Name = %q, want macd", macd.Name)
	}
	if len(macd.Params) != 3 || macd.Params[0] != 12 || macd.Params[1] != 26 || macd.Params[2] != 9 {
		t.Errorf("Params = %v, want [12 26 9]", macd.Params)
	}
}

func TestParseComments(t *testing.T) {
	input := `# RSI bounce strategy
SCAN bounce
  EVERY 300s
  -- scan these symbols
  SYMBOLS AAPL, NVDA
  FETCH quote, rsi(14)
  WHERE rsi <= 25
  ENTER LONG
    STOP 5%
    BUDGET 150`

	plan, err := Parse(input)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if plan.Name != "bounce" {
		t.Errorf("Name = %q, want bounce", plan.Name)
	}
}

func TestParseValidationErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"no name", "EVERY 60s\nSYMBOLS AAPL\nFETCH quote\nWHERE price >= 1\nENTER LONG\n  STOP 3%\n  BUDGET 100"},
		{"no interval", "SCAN x\nSYMBOLS AAPL\nFETCH quote\nWHERE price >= 1\nENTER LONG\n  STOP 3%\n  BUDGET 100"},
		{"no symbols", "SCAN x\nEVERY 60s\nFETCH quote\nWHERE price >= 1\nENTER LONG\n  STOP 3%\n  BUDGET 100"},
		{"no fetch", "SCAN x\nEVERY 60s\nSYMBOLS AAPL\nWHERE price >= 1\nENTER LONG\n  STOP 3%\n  BUDGET 100"},
		{"no enter", "SCAN x\nEVERY 60s\nSYMBOLS AAPL\nFETCH quote\nWHERE price >= 1"},
	}
	for _, tt := range tests {
		_, err := Parse(tt.input)
		if err == nil {
			t.Errorf("%s: expected error, got nil", tt.name)
		}
	}
}
