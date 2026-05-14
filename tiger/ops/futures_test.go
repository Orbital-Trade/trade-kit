package ops

import (
	"errors"
	"testing"
)

// ── FuturesEntry ─────────────────────────────────────────────────────────────

func TestFuturesEntry_paper(t *testing.T) {
	m := newMock(true)
	res, err := FuturesEntry(m, "MES", "LONG", 1, 5100.0, 5090.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("mode: want PAPER, got %s", res.Mode)
	}
	if res.EntryOrderID != "PAPER" || res.StopOrderID != "PAPER" {
		t.Errorf("expected PAPER order IDs, got entry=%s stop=%s", res.EntryOrderID, res.StopOrderID)
	}
	if m.called("place_order") != 0 {
		t.Error("paper mode must not call place_order")
	}
}

func TestFuturesEntry_liveSuccess(t *testing.T) {
	m := newMock(false).
		on("place_order", map[string]interface{}{"id": 100001}, nil). // entry
		on("place_order", map[string]interface{}{"id": 100002}, nil)  // stop

	res, err := FuturesEntry(m, "MES", "LONG", 1, 5100.0, 5090.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.EntryOrderID != "100001" {
		t.Errorf("entry_order_id: want 100001, got %s", res.EntryOrderID)
	}
	if res.StopOrderID != "100002" {
		t.Errorf("stop_order_id: want 100002, got %s", res.StopOrderID)
	}
	if res.Direction != "LONG" || res.Contracts != 1 {
		t.Errorf("direction/contracts: want LONG/1, got %s/%d", res.Direction, res.Contracts)
	}
	// Must make exactly 2 place_order calls.
	if m.called("place_order") != 2 {
		t.Errorf("expected 2 place_order calls, got %d", m.called("place_order"))
	}
}

func TestFuturesEntry_shortDirection(t *testing.T) {
	// SHORT direction: entry=SELL, stop=BUY.
	m := newMock(false).
		on("place_order", map[string]interface{}{"id": 200001}, nil). // entry (SELL)
		on("place_order", map[string]interface{}{"id": 200002}, nil)  // stop (BUY)

	res, err := FuturesEntry(m, "MES", "SHORT", 2, 5200.0, 5210.0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Direction != "SHORT" || res.Contracts != 2 {
		t.Errorf("direction/contracts: want SHORT/2, got %s/%d", res.Direction, res.Contracts)
	}
}

func TestFuturesEntry_stopFailure_emergencyClose(t *testing.T) {
	// Entry succeeds, stop fails → emergency close must be sent.
	m := newMock(false).
		on("place_order", map[string]interface{}{"id": 300001}, nil).             // entry OK
		on("place_order", nil, errors.New("stop rejected")).                      // stop FAILS
		on("place_order", map[string]interface{}{"id": 300003}, nil)              // emergency close OK

	_, err := FuturesEntry(m, "MES", "LONG", 1, 5100.0, 5090.0)
	if err == nil {
		t.Fatal("expected error when stop fails, got nil")
	}
	// 3 place_order calls: entry + stop + emergency close
	if m.called("place_order") != 3 {
		t.Errorf("expected 3 place_order calls (entry+stop+close), got %d", m.called("place_order"))
	}
}

func TestFuturesEntry_entryFailure(t *testing.T) {
	// Entry itself fails — no stop or close should be attempted.
	m := newMock(false).onErr("place_order", "market closed")

	_, err := FuturesEntry(m, "MES", "LONG", 1, 5100.0, 5090.0)
	if err == nil {
		t.Fatal("expected error for entry failure, got nil")
	}
	if m.called("place_order") != 1 {
		t.Errorf("expected 1 place_order call, got %d", m.called("place_order"))
	}
}

// ── FuturesClose ─────────────────────────────────────────────────────────────

func TestFuturesClose_paper(t *testing.T) {
	m := newMock(true)
	res, err := FuturesClose(m, "MES", "LONG", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("mode: want PAPER, got %s", res.Mode)
	}
	// LONG close = SELL
	if res.Action != "SELL" {
		t.Errorf("action: want SELL (close of LONG), got %s", res.Action)
	}
}

func TestFuturesClose_longIsMarketSell(t *testing.T) {
	m := newMock(false).on("place_order", map[string]interface{}{"id": 400001}, nil)
	res, err := FuturesClose(m, "MES", "LONG", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Action != "SELL" || res.Type != "MKT" {
		t.Errorf("want SELL/MKT, got %s/%s", res.Action, res.Type)
	}
}

func TestFuturesClose_shortIsMarketBuy(t *testing.T) {
	m := newMock(false).on("place_order", map[string]interface{}{"id": 400002}, nil)
	res, err := FuturesClose(m, "MES", "SHORT", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// SHORT close = BUY
	if res.Action != "BUY" {
		t.Errorf("action: want BUY (close of SHORT), got %s", res.Action)
	}
}

// ── FuturesUpdateStop ─────────────────────────────────────────────────────────

func TestFuturesUpdateStop_paper(t *testing.T) {
	m := newMock(true)
	res, err := FuturesUpdateStop(m, "MES", "LONG", 1, 5095.0, "OLD-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Mode != "PAPER" {
		t.Errorf("mode: want PAPER, got %s", res.Mode)
	}
	if res.Price != 5095.0 {
		t.Errorf("price: want 5095.0, got %f", res.Price)
	}
}

func TestFuturesUpdateStop_cancelThenPlace(t *testing.T) {
	m := newMock(false).
		on("cancel_order", map[string]interface{}{"result": "ok"}, nil).
		on("place_order", map[string]interface{}{"id": 500001}, nil)

	res, err := FuturesUpdateStop(m, "MES", "LONG", 1, 5095.0, "OLD-999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "500001" {
		t.Errorf("order_id: want 500001, got %s", res.OrderID)
	}
	if m.called("cancel_order") != 1 {
		t.Errorf("expected 1 cancel_order call, got %d", m.called("cancel_order"))
	}
	if m.called("place_order") != 1 {
		t.Errorf("expected 1 place_order call, got %d", m.called("place_order"))
	}
}

func TestFuturesUpdateStop_cancelFailContinues(t *testing.T) {
	// Cancel failure is non-fatal — the new stop must still be placed.
	m := newMock(false).
		onErr("cancel_order", "order already filled").
		on("place_order", map[string]interface{}{"id": 500002}, nil)

	res, err := FuturesUpdateStop(m, "MES", "LONG", 1, 5095.0, "OLD-XXX")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OrderID != "500002" {
		t.Errorf("order_id: want 500002, got %s", res.OrderID)
	}
}

// ── actionsFor ────────────────────────────────────────────────────────────────

func TestActionsFor(t *testing.T) {
	cases := []struct {
		direction  string
		wantEntry  string
		wantStop   string
	}{
		{"LONG", "BUY", "SELL"},
		{"SHORT", "SELL", "BUY"},
		{"long", "BUY", "SELL"},  // lowercase treated as LONG (default)
	}
	for _, c := range cases {
		entry, stop := actionsFor(c.direction)
		if entry != c.wantEntry || stop != c.wantStop {
			t.Errorf("actionsFor(%q): want %s/%s, got %s/%s",
				c.direction, c.wantEntry, c.wantStop, entry, stop)
		}
	}
}
