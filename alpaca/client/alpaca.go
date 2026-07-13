// Package client provides a standalone Alpaca Markets REST API client.
//
// No external dependencies — only Go stdlib.
//
// Auth: APCA-API-KEY-ID + APCA-API-SECRET-KEY headers on every request.
//
// Base URLs:
//   - Paper: https://paper-api.alpaca.markets
//   - Live:  https://api.alpaca.markets
//   - Data:  https://data.alpaca.markets
//
// Usage:
//
//	c, err := client.New(paperMode)
//	data, err := c.Get("/v2/account", nil)
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var debugAPI = strings.ToLower(os.Getenv("ALPACA_LOG_LEVEL")) == "debug"

const (
	liveURL  = "https://api.alpaca.markets"
	paperURL = "https://paper-api.alpaca.markets"
	dataURL  = "https://data.alpaca.markets"
)

// AlpacaClient is the low-level REST transport.
type AlpacaClient struct {
	keyID     string
	secretKey string
	http      *http.Client
	paper     bool
	baseURL   string
	dataURL   string

	mu             sync.Mutex
	rateLimitLeft  int
	rateLimitReset time.Time
}

// New loads credentials from .env and returns a ready-to-use client.
func New(paper bool) (*AlpacaClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return newClient(cfg.keyID, cfg.secretKey, paper), nil
}

// NewFromCreds constructs a client from explicit credentials.
// Used by the sidecar server.
func NewFromCreds(keyID, secretKey string, paper bool) (*AlpacaClient, error) {
	if keyID == "" || secretKey == "" {
		return nil, fmt.Errorf("alpaca: key_id and secret_key are required")
	}
	return newClient(keyID, secretKey, paper), nil
}

func newClient(keyID, secretKey string, paper bool) *AlpacaClient {
	base := liveURL
	if paper {
		base = paperURL
	}
	return &AlpacaClient{
		keyID:         keyID,
		secretKey:     secretKey,
		http:          &http.Client{Timeout: 15 * time.Second},
		paper:         paper,
		baseURL:       base,
		dataURL:       dataURL,
		rateLimitLeft: 200,
	}
}

// IsPaper returns true when running against the paper trading API.
func (c *AlpacaClient) IsPaper() bool { return c.paper }

// Get makes an authenticated GET request to the trading API.
func (c *AlpacaClient) Get(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("GET", c.baseURL, path, query, nil)
}

// Post makes an authenticated POST request to the trading API.
func (c *AlpacaClient) Post(path string, body interface{}) (json.RawMessage, error) {
	return c.do("POST", c.baseURL, path, nil, body)
}

// Delete makes an authenticated DELETE request to the trading API.
func (c *AlpacaClient) Delete(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("DELETE", c.baseURL, path, query, nil)
}

// Patch makes an authenticated PATCH request to the trading API.
func (c *AlpacaClient) Patch(path string, body interface{}) (json.RawMessage, error) {
	return c.do("PATCH", c.baseURL, path, nil, body)
}

// DataGet makes an authenticated GET request to the market data API.
func (c *AlpacaClient) DataGet(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("GET", c.dataURL, path, query, nil)
}

func (c *AlpacaClient) do(method, baseURL, path string, query map[string]string, body interface{}) (json.RawMessage, error) {
	// Rate limit check.
	c.mu.Lock()
	if c.rateLimitLeft <= 0 && time.Now().Before(c.rateLimitReset) {
		wait := time.Until(c.rateLimitReset)
		c.mu.Unlock()
		if debugAPI {
			fmt.Fprintf(os.Stderr, "[DEBUG] alpaca rate limited, waiting %s\n", wait)
		}
		time.Sleep(wait)
	} else {
		c.mu.Unlock()
	}

	url := baseURL + path
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
			fmt.Fprintf(os.Stderr, "[DEBUG] alpaca %s %s body=%s\n", method, path, string(b))
		}
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("APCA-API-KEY-ID", c.keyID)
	req.Header.Set("APCA-API-SECRET-KEY", c.secretKey)
	req.Header.Set("User-Agent", "alpaca-cli/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	c.updateRateLimit(resp.Header)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if debugAPI {
		fmt.Fprintf(os.Stderr, "[DEBUG] alpaca %s %s status=%d body=%.500s\n", method, path, resp.StatusCode, string(raw))
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429): retry later")
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("auth failed (%d): check ALPACA_KEY_ID and ALPACA_SECRET_KEY", resp.StatusCode)
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

func (c *AlpacaClient) updateRateLimit(h http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if v := h.Get("X-Ratelimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.rateLimitLeft = n
		}
	}
	if v := h.Get("X-Ratelimit-Reset"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			c.rateLimitReset = time.Unix(ts, 0)
		}
	}
}

// ── Config loading ───────────────────────────────────────────────────────────

type alpacaCreds struct {
	keyID     string
	secretKey string
}

func loadConfig() (alpacaCreds, error) {
	searchDirs := []string{
		filepath.Join("..", "..", "brokers", "Alpaca"),
		filepath.Join("brokers", "Alpaca"),
		filepath.Join(os.Getenv("HOME"), ".trade-kit", "alpaca"),
	}

	cfg := alpacaCreds{}
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
			case "ALPACA_KEY_ID":
				cfg.keyID = v
			case "ALPACA_SECRET_KEY":
				cfg.secretKey = v
			}
		}
		if cfg.keyID != "" && cfg.secretKey != "" {
			break
		}
	}

	if cfg.keyID == "" || cfg.secretKey == "" {
		return cfg, fmt.Errorf(
			"Alpaca credentials not found.\n"+
				"Expected: brokers/Alpaca/.env with ALPACA_KEY_ID and ALPACA_SECRET_KEY\n"+
				"Search paths: ../../brokers/Alpaca/.env, brokers/Alpaca/.env, ~/.trade-kit/alpaca/.env\n"+
				"Get your keys at: https://app.alpaca.markets\n"+
				"Have: key_id=%v secret_key=%v",
			cfg.keyID != "", cfg.secretKey != "",
		)
	}
	return cfg, nil
}
