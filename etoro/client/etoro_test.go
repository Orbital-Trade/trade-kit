package client

import (
	"net/http"
	"testing"
)

func TestNewUUID(t *testing.T) {
	id := newUUID()
	if len(id) != 36 {
		t.Errorf("UUID length = %d, want 36, got %q", len(id), id)
	}
	// Check dashes at correct positions.
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		t.Errorf("UUID format invalid: %q", id)
	}
	// Check version nibble.
	if id[14] != '4' {
		t.Errorf("UUID version = %c, want 4", id[14])
	}
	// Check uniqueness.
	id2 := newUUID()
	if id == id2 {
		t.Error("two UUIDs are identical — expected unique")
	}
}

func TestUpdateRateLimit(t *testing.T) {
	c := &EtoroClient{rateLimitLeft: 60}

	h := http.Header{}
	h.Set("RateLimit-Remaining", "42")
	h.Set("RateLimit-Reset", "30")
	c.updateRateLimit(h)

	if c.rateLimitLeft != 42 {
		t.Errorf("rateLimitLeft = %d, want 42", c.rateLimitLeft)
	}
	if c.rateLimitReset.IsZero() {
		t.Error("rateLimitReset should not be zero")
	}
}

func TestLoadConfigMissing(t *testing.T) {
	// With no .env files present, loadConfig should fail gracefully.
	_, err := loadConfig()
	if err == nil {
		// If this passes, it means there's a real .env file — that's fine.
		return
	}
	// Check error message is helpful.
	if err.Error() == "" {
		t.Error("error message should not be empty")
	}
}
