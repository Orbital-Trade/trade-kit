// Package client provides a standalone IBKR Client Portal REST API client.
//
// No external dependencies — only Go stdlib.
//
// Auth: Session-based. The Client Portal Gateway handles authentication
// via browser login. API calls use the gateway's session cookies.
//
// Base URL: https://{host}:{port}/v1/api (default: https://localhost:5000/v1/api)
//
// The gateway uses self-signed TLS certificates, so this client skips
// TLS verification.
//
// Usage:
//
//	c, err := client.New(paperMode)
//	data, err := c.Get("/v1/api/portfolio/accounts", nil)
package client

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var debugAPI = strings.ToLower(os.Getenv("IBKR_LOG_LEVEL")) == "debug"

// IBKRClient is the low-level REST transport for the IBKR Client Portal API.
type IBKRClient struct {
	host      string
	port      string
	accountID string
	http      *http.Client
	paper     bool
	baseURL   string

	mu    sync.Mutex
	conid map[string]int // symbol → conid cache
}

// New loads credentials from .env and returns a ready-to-use client.
func New(paper bool) (*IBKRClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return newClient(cfg.host, cfg.port, cfg.accountID, paper), nil
}

// NewFromCreds constructs a client from explicit credentials.
// Used by the sidecar server.
func NewFromCreds(host, port, accountID string, paper bool) (*IBKRClient, error) {
	if accountID == "" {
		return nil, fmt.Errorf("ibkr: account_id is required")
	}
	if host == "" {
		host = "localhost"
	}
	if port == "" {
		port = "5000"
	}
	return newClient(host, port, accountID, paper), nil
}

func newClient(host, port, accountID string, paper bool) *IBKRClient {
	return &IBKRClient{
		host:      host,
		port:      port,
		accountID: accountID,
		paper:     paper,
		baseURL:   fmt.Sprintf("https://%s:%s", host, port),
		http: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
		conid: make(map[string]int),
	}
}

// IsPaper returns true when running against a paper trading account.
func (c *IBKRClient) IsPaper() bool { return c.paper }

// AccountID returns the configured account ID.
func (c *IBKRClient) AccountID() string { return c.accountID }

// Get makes a GET request to the Client Portal API.
func (c *IBKRClient) Get(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("GET", path, query, nil)
}

// Post makes a POST request to the Client Portal API.
func (c *IBKRClient) Post(path string, body interface{}) (json.RawMessage, error) {
	return c.do("POST", path, nil, body)
}

// Delete makes a DELETE request to the Client Portal API.
func (c *IBKRClient) Delete(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("DELETE", path, query, nil)
}

// ResolveConID resolves a ticker symbol to an IBKR contract ID.
// Results are cached for the lifetime of the client.
func (c *IBKRClient) ResolveConID(symbol string) (int, error) {
	c.mu.Lock()
	if id, ok := c.conid[symbol]; ok {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	data, err := c.Get("/v1/api/iserver/secdef/search", map[string]string{
		"symbol": symbol,
	})
	if err != nil {
		return 0, fmt.Errorf("resolve_conid %s: %w", symbol, err)
	}

	var results []struct {
		ConID int    `json:"conid"`
		Desc  string `json:"companyName"`
	}
	if err := json.Unmarshal(data, &results); err != nil {
		return 0, fmt.Errorf("resolve_conid %s: parse: %w", symbol, err)
	}
	if len(results) == 0 {
		return 0, fmt.Errorf("resolve_conid %s: no results", symbol)
	}

	id := results[0].ConID
	c.mu.Lock()
	c.conid[symbol] = id
	c.mu.Unlock()

	if debugAPI {
		fmt.Fprintf(os.Stderr, "[DEBUG] ibkr resolved %s → conid %d\n", symbol, id)
	}
	return id, nil
}

// Tickle sends a keepalive to maintain the gateway session.
func (c *IBKRClient) Tickle() error {
	_, err := c.Post("/v1/api/tickle", nil)
	return err
}

// AuthStatus checks whether the gateway session is authenticated.
func (c *IBKRClient) AuthStatus() (json.RawMessage, error) {
	return c.Get("/v1/api/iserver/auth/status", nil)
}

func (c *IBKRClient) do(method, path string, query map[string]string, body interface{}) (json.RawMessage, error) {
	url := c.baseURL + path
	if len(query) > 0 {
		parts := make([]string, 0, len(query))
		for k, v := range query {
			parts = append(parts, k+"="+v)
		}
		url += "?" + strings.Join(parts, "&")
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = strings.NewReader(string(b))
		if debugAPI {
			fmt.Fprintf(os.Stderr, "[DEBUG] ibkr %s %s body=%s\n", method, path, string(b))
		}
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ibkr-cli/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if debugAPI {
		fmt.Fprintf(os.Stderr, "[DEBUG] ibkr %s %s status=%d body=%.500s\n", method, path, resp.StatusCode, string(raw))
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429): retry later")
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("auth failed (%d): ensure Client Portal Gateway is authenticated", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found (404): %s %s", method, path)
	}
	if resp.StatusCode == 422 {
		return nil, fmt.Errorf("unprocessable (422): %.500s", string(raw))
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %.500s", resp.StatusCode, string(raw))
	}
	if resp.StatusCode == 204 || len(raw) == 0 {
		return json.RawMessage("null"), nil
	}

	return json.RawMessage(raw), nil
}

// ── Config loading ───────────────────────────────────────────────────────────

type ibkrCreds struct {
	host      string
	port      string
	accountID string
}

func loadConfig() (ibkrCreds, error) {
	searchDirs := []string{
		filepath.Join("..", "..", "brokers", "IBKR"),
		filepath.Join("brokers", "IBKR"),
		filepath.Join(os.Getenv("HOME"), ".trade-kit", "ibkr"),
	}

	cfg := ibkrCreds{}
	for _, dir := range searchDirs {
		path := filepath.Join(dir, ".env")
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
			switch k {
			case "IBKR_GATEWAY_HOST":
				cfg.host = v
			case "IBKR_GATEWAY_PORT":
				cfg.port = v
			case "IBKR_ACCOUNT_ID":
				cfg.accountID = v
			}
		}
		if cfg.accountID != "" {
			break
		}
	}

	// Defaults for host and port.
	if cfg.host == "" {
		cfg.host = "localhost"
	}
	if cfg.port == "" {
		cfg.port = "5000"
	}

	if cfg.accountID == "" {
		return cfg, fmt.Errorf(
			"IBKR credentials not found.\n"+
				"Expected: brokers/IBKR/.env with IBKR_ACCOUNT_ID\n"+
				"Optional: IBKR_GATEWAY_HOST (default: localhost), IBKR_GATEWAY_PORT (default: 5000)\n"+
				"Search paths: ../../brokers/IBKR/.env, brokers/IBKR/.env, ~/.trade-kit/ibkr/.env\n"+
				"Ensure the Client Portal Gateway is running and authenticated.\n"+
				"Have: account_id=%v",
			cfg.accountID != "",
		)
	}
	return cfg, nil
}
