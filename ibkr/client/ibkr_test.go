package client

import "testing"

func TestNewClient(t *testing.T) {
	c := newClient("localhost", "5000", "TEST123", true)
	if !c.IsPaper() {
		t.Error("expected paper mode")
	}
	if c.AccountID() != "TEST123" {
		t.Errorf("AccountID = %q, want TEST123", c.AccountID())
	}
	if c.baseURL != "https://localhost:5000" {
		t.Errorf("baseURL = %q, want https://localhost:5000", c.baseURL)
	}

	c2 := newClient("192.168.1.10", "5001", "LIVE456", false)
	if c2.IsPaper() {
		t.Error("expected live mode")
	}
	if c2.baseURL != "https://192.168.1.10:5001" {
		t.Errorf("baseURL = %q, want https://192.168.1.10:5001", c2.baseURL)
	}
}

func TestNewFromCredsMissingAccountID(t *testing.T) {
	_, err := NewFromCreds("localhost", "5000", "", true)
	if err == nil {
		t.Error("expected error for empty account_id")
	}
}

func TestNewFromCredsDefaults(t *testing.T) {
	c, err := NewFromCreds("", "", "ACCT123", true)
	if err != nil {
		t.Fatalf("NewFromCreds: %v", err)
	}
	if c.host != "localhost" {
		t.Errorf("host = %q, want localhost", c.host)
	}
	if c.port != "5000" {
		t.Errorf("port = %q, want 5000", c.port)
	}
	if c.AccountID() != "ACCT123" {
		t.Errorf("AccountID = %q, want ACCT123", c.AccountID())
	}
	if !c.IsPaper() {
		t.Error("expected paper mode")
	}
}

func TestNewFromCredsValid(t *testing.T) {
	c, err := NewFromCreds("myhost", "9000", "MY_ACCT", false)
	if err != nil {
		t.Fatalf("NewFromCreds: %v", err)
	}
	if c.host != "myhost" {
		t.Errorf("host = %q, want myhost", c.host)
	}
	if c.port != "9000" {
		t.Errorf("port = %q, want 9000", c.port)
	}
	if c.IsPaper() {
		t.Error("expected live mode")
	}
}

func TestConIDCache(t *testing.T) {
	c := newClient("localhost", "5000", "TEST123", true)
	// Manually populate cache.
	c.conid["AAPL"] = 265598
	c.conid["MSFT"] = 272093

	if len(c.conid) != 2 {
		t.Errorf("cache len = %d, want 2", len(c.conid))
	}
}
