package client

import "testing"

func TestNewClient(t *testing.T) {
	c := newClient("test-key", "test-token", true)
	if !c.IsPaper() {
		t.Error("expected paper mode")
	}
	if c.baseURL != baseURL {
		t.Errorf("baseURL = %q, want %q", c.baseURL, baseURL)
	}

	c2 := newClient("test-key", "test-token", false)
	if c2.IsPaper() {
		t.Error("expected live mode")
	}
}

func TestNewFromCredsMissing(t *testing.T) {
	_, err := NewFromCreds("", "token", true)
	if err == nil {
		t.Error("expected error for empty api_key")
	}
	_, err = NewFromCreds("key", "", true)
	if err == nil {
		t.Error("expected error for empty access_token")
	}
}

func TestNewFromCredsValid(t *testing.T) {
	c, err := NewFromCreds("my-key", "my-token", true)
	if err != nil {
		t.Fatalf("NewFromCreds: %v", err)
	}
	if c.apiKey != "my-key" {
		t.Errorf("apiKey = %q, want my-key", c.apiKey)
	}
	if c.accessToken != "my-token" {
		t.Errorf("accessToken = %q, want my-token", c.accessToken)
	}
	if !c.IsPaper() {
		t.Error("expected paper mode")
	}
}
