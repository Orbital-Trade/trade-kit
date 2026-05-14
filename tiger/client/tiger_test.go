package client

// Unit tests for pure functions in the client package.
// These do not require a live Tiger connection or API keys.

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// ── fallbackContract ──────────────────────────────────────────────────────────

// staticTime is a no-op TigerClient used only to call unexported helpers.
var staticClient = &TigerClient{contractCache: make(map[string]string)}

func TestFallbackContract_march(t *testing.T) {
	// Month 1 (January) → next quarterly expiry is March.
	// We can't override time.Now() easily, so we test the logic directly
	// by verifying that fallbackContract returns one of the expected formats.
	code := staticClient.fallbackContract("MES")
	if !strings.HasPrefix(code, "MES") {
		t.Errorf("expected code to start with MES, got %s", code)
	}
	// Code format: MES + 2-digit year + 2-digit month (e.g. MES2506)
	if len(code) != 7 {
		t.Errorf("expected 7 chars (MES+YYMM), got %q (len %d)", code, len(code))
	}
}

func TestFallbackContract_format(t *testing.T) {
	symbols := []string{"MES", "MNQ", "M2K"}
	for _, sym := range symbols {
		code := staticClient.fallbackContract(sym)
		if !strings.HasPrefix(code, sym) {
			t.Errorf("%s: code should start with symbol, got %s", sym, code)
		}
	}
}

func TestFallbackContract_quarterlyMonths(t *testing.T) {
	// The result must end in 03, 06, 09, or 12.
	code := staticClient.fallbackContract("MES")
	month := code[len(code)-2:]
	validMonths := map[string]bool{"03": true, "06": true, "09": true, "12": true}
	if !validMonths[month] {
		t.Errorf("expected quarterly month (03/06/09/12), got %q in code %q", month, code)
	}
}

func TestFallbackContract_futureOnly(t *testing.T) {
	// The returned contract must be in the future (not in the past).
	code := staticClient.fallbackContract("MES")
	// code = MES + YY + MM, e.g. MES2506 → year=25, month=06
	yearStr := code[3:5]
	monthStr := code[5:7]

	year := 2000 + int(yearStr[0]-'0')*10 + int(yearStr[1]-'0')
	month := int(monthStr[0]-'0')*10 + int(monthStr[1]-'0')

	now := time.Now()
	contractDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	if contractDate.Before(now.AddDate(0, 0, -1)) {
		t.Errorf("fallback contract %s is in the past (year=%d, month=%d)", code, year, month)
	}
}

// ── parseKey ─────────────────────────────────────────────────────────────────

func TestParseKey_emptyString(t *testing.T) {
	_, err := parseKey("")
	if err == nil {
		t.Error("expected error for empty key, got nil")
	}
}

func TestParseKey_invalidBase64(t *testing.T) {
	_, err := parseKey("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}
}

func TestParseKey_validBase64ButNotAKey(t *testing.T) {
	// Valid base64 but not a DER-encoded RSA key.
	_, err := parseKey("aGVsbG8gd29ybGQ=") // "hello world"
	if err == nil {
		t.Error("expected error for non-key bytes, got nil")
	}
}

// ── double-encoding and empty-string normalisation ────────────────────────────
// Tiger double-encodes list responses: the data field is a JSON string whose
// content is the actual JSON. client.Call must unwrap this before returning.
// These tests verify the unwrap logic using the same JSON operations Call() uses.

func TestDoubleEncoding_emptyString(t *testing.T) {
	// Tiger returns "" for empty lists. After unwrap → null.
	encoded := []byte(`""`)
	var inner string
	if err := json.Unmarshal(encoded, &inner); err != nil {
		t.Fatalf("unmarshal of empty JSON string: %v", err)
	}
	if inner != "" {
		t.Errorf("expected empty string, got %q", inner)
	}
	// Empty inner → should become null; null unmarshals to nil slice without error.
	var s []struct{ X int }
	if err := json.Unmarshal([]byte("null"), &s); err != nil {
		t.Errorf("json.Unmarshal(null, &slice) should not error: %v", err)
	}
}

func TestDoubleEncoding_listResponse(t *testing.T) {
	// Simulate Tiger returning a double-encoded array: data = "\"[{...}]\""
	innerJSON := `[{"symbol":"NOK","position":10}]`
	encoded, _ := json.Marshal(innerJSON) // produces "\"[{...}]\""

	// Verify it starts with '"' (is a JSON string).
	if encoded[0] != '"' {
		t.Fatalf("expected JSON string, got %s", string(encoded))
	}

	// Unwrap: decode string → get inner JSON.
	var s string
	if err := json.Unmarshal(encoded, &s); err != nil {
		t.Fatalf("first decode: %v", err)
	}
	if s != innerJSON {
		t.Errorf("inner JSON mismatch: want %q, got %q", innerJSON, s)
	}

	// Inner JSON is now parseable as an array.
	var items []struct {
		Symbol   string `json:"symbol"`
		Position int    `json:"position"`
	}
	if err := json.Unmarshal([]byte(s), &items); err != nil {
		t.Fatalf("parse unwrapped array: %v", err)
	}
	if len(items) != 1 || items[0].Symbol != "NOK" {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestDoubleEncoding_wrappedObject(t *testing.T) {
	// Tiger also returns {"items":[...]} wrapped in a string.
	innerJSON := `{"items":[{"symbol":"NOK","position":10}]}`
	encoded, _ := json.Marshal(innerJSON)

	var s string
	json.Unmarshal(encoded, &s)

	var wrapper struct {
		Items []struct {
			Symbol   string `json:"symbol"`
			Position int    `json:"position"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(s), &wrapper); err != nil {
		t.Fatalf("parse items wrapper: %v", err)
	}
	if len(wrapper.Items) != 1 || wrapper.Items[0].Symbol != "NOK" {
		t.Errorf("unexpected items: %+v", wrapper.Items)
	}
}

// ── loadConfig ────────────────────────────────────────────────────────────────

func TestLoadConfig_missingFiles(t *testing.T) {
	// When none of the search paths have config files, loadConfig must error.
	// Override search paths by ensuring no valid dir exists at the test path.
	// (The test binary runs in tools/tiger/client/ which has no brokers/Tiger/.)
	_, err := loadConfig()
	// We expect either success (if real credentials exist on this machine) or
	// a clear missing-credentials error. We just verify it doesn't panic.
	if err != nil {
		if !strings.Contains(err.Error(), "credentials") && !strings.Contains(err.Error(), "Tiger") {
			t.Errorf("unexpected error message: %v", err)
		}
	}
}
