package ops

import (
	"math"
	"testing"
)

// Tiger "brief" endpoint returns a JSON array of quote objects.

func briefResponse(latestPrice, preClose, openPrice, highPrice, lowPrice, volume float64) []map[string]interface{} {
	return []map[string]interface{}{{
		"symbol":       "NOK",
		"latest_price": latestPrice,
		"pre_close":    preClose,
		"open_price":   openPrice,
		"high_price":   highPrice,
		"low_price":    lowPrice,
		"volume":       volume,
	}}
}

func TestGetQuote_basic(t *testing.T) {
	m := newMock(false).on("brief", briefResponse(4.80, 4.50, 4.51, 4.90, 4.48, 2_000_000), nil)

	q, err := GetQuote(m, "NOK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Symbol != "NOK" {
		t.Errorf("symbol: want NOK, got %s", q.Symbol)
	}
	if q.Market != "US" {
		t.Errorf("market: want US, got %s", q.Market)
	}
	if q.Currency != "USD" {
		t.Errorf("currency: want USD, got %s", q.Currency)
	}
	if q.Price != 4.80 {
		t.Errorf("price: want 4.80, got %f", q.Price)
	}
	if q.PrevClose != 4.50 {
		t.Errorf("prev_close: want 4.50, got %f", q.PrevClose)
	}
	wantPct := (4.80 - 4.50) / 4.50 * 100
	if math.Abs(q.ChangePct-wantPct) > 0.001 {
		t.Errorf("change_pct: want %.4f, got %.4f", wantPct, q.ChangePct)
	}
	if q.Volume != 2_000_000 {
		t.Errorf("volume: want 2000000, got %f", q.Volume)
	}
}

func TestGetQuote_zeroPrevClose(t *testing.T) {
	m := newMock(false).on("brief", briefResponse(4.50, 0, 4.40, 4.55, 4.38, 500_000), nil)

	q, err := GetQuote(m, "NOK")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Price != 4.50 {
		t.Errorf("price: want 4.50, got %f", q.Price)
	}
	if q.ChangePct != 0 {
		t.Errorf("change_pct: want 0 when pre_close=0, got %f", q.ChangePct)
	}
}

func TestGetQuote_sgxSymbol(t *testing.T) {
	m := newMock(false).on("brief", []map[string]interface{}{{
		"symbol":       "ES3",
		"latest_price": 5.03,
		"pre_close":    5.015,
	}}, nil)

	q, err := GetQuote(m, "ES3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Market != "SG" {
		t.Errorf("market: want SG, got %s", q.Market)
	}
	if q.Currency != "SGD" {
		t.Errorf("currency: want SGD, got %s", q.Currency)
	}
}

func TestGetQuote_emptyResponse(t *testing.T) {
	m := newMock(false).on("brief", []map[string]interface{}{}, nil)
	_, err := GetQuote(m, "UNKNOWN")
	if err == nil {
		t.Fatal("expected error for empty response, got nil")
	}
}

func TestGetQuote_apiError(t *testing.T) {
	m := newMock(false).onErr("brief", "symbol not found")
	_, err := GetQuote(m, "INVALID")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
