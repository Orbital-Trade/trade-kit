package client

import "testing"

func TestNewClient(t *testing.T) {
	c := newClient("test-key", "test-secret", true)
	if !c.IsPaper() {
		t.Error("expected paper mode")
	}
	if c.baseURL != paperURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, paperURL)
	}

	c2 := newClient("test-key", "test-secret", false)
	if c2.IsPaper() {
		t.Error("expected live mode")
	}
	if c2.baseURL != liveURL {
		t.Errorf("baseURL = %q, want %q", c2.baseURL, liveURL)
	}
}

func TestNewFromCredsMissing(t *testing.T) {
	_, err := NewFromCreds("", "secret", true)
	if err == nil {
		t.Error("expected error for empty key_id")
	}
	_, err = NewFromCreds("key", "", true)
	if err == nil {
		t.Error("expected error for empty secret")
	}
}

func TestNewFromCredsValid(t *testing.T) {
	c, err := NewFromCreds("my-key", "my-secret", true)
	if err != nil {
		t.Fatalf("NewFromCreds: %v", err)
	}
	if c.keyID != "my-key" {
		t.Errorf("keyID = %q, want my-key", c.keyID)
	}
	if !c.IsPaper() {
		t.Error("expected paper mode")
	}
}
