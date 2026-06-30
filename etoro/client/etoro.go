// Package client provides a standalone eToro REST API client.
//
// No external dependencies — only Go stdlib. Credentials are loaded from
// .env files in standard trade-kit search paths.
//
// eToro API requires three headers on every request:
//   - x-api-key:    API key from https://api-portal.etoro.com
//   - x-user-key:   User-specific authentication token
//   - x-request-id: UUID per request for tracing
//
// Base URL: https://api.etoro.com
//
// Rate limits:
//   - Default:     60 req / 60s (shared)
//   - Trading:     20 req / 60s
//   - Market data: 120 req / 60s (shared)
//
// Usage:
//
//	c, err := client.New(paperMode)
//	data, err := c.Get("/api/v1/balances", nil)
//	data, err := c.Post("/api/v1/trading/demo/orders", payload)
package client

import (
	"crypto/rand"
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

var debugAPI = strings.ToLower(os.Getenv("ETORO_LOG_LEVEL")) == "debug"

const baseURL = "https://api.etoro.com"

// EtoroClient is the low-level REST transport. Operations live in ops/.
type EtoroClient struct {
	apiKey  string
	userKey string
	http    *http.Client
	paper   bool

	// Rate limiter state.
	mu            sync.Mutex
	rateLimitLeft int
	rateLimitReset time.Time
}

// New loads credentials and returns a ready-to-use client.
// Set paper=true to route orders through the demo API.
func New(paper bool) (*EtoroClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return &EtoroClient{
		apiKey:         cfg.apiKey,
		userKey:        cfg.userKey,
		http:           &http.Client{Timeout: 15 * time.Second},
		paper:          paper,
		rateLimitLeft:  60,
	}, nil
}

// NewFromCreds constructs an EtoroClient from explicit credentials,
// bypassing .env file loading. Used by the sidecar server.
func NewFromCreds(apiKey, userKey string, paper bool) (*EtoroClient, error) {
	if apiKey == "" || userKey == "" {
		return nil, fmt.Errorf("etoro: api_key and user_key are required")
	}
	return &EtoroClient{
		apiKey:        apiKey,
		userKey:       userKey,
		http:          &http.Client{Timeout: 15 * time.Second},
		paper:         paper,
		rateLimitLeft: 60,
	}, nil
}

// IsPaper returns true when running in paper/demo mode.
func (c *EtoroClient) IsPaper() bool { return c.paper }

// tradingPrefix returns the API path prefix for trading endpoints.
// Demo mode uses /api/v1/trading/demo, live uses /api/v1/trading/real.
func (c *EtoroClient) tradingPrefix() string {
	if c.paper {
		return "/api/v1/trading/demo"
	}
	return "/api/v1/trading/real"
}

// Get makes an authenticated GET request to the eToro API.
func (c *EtoroClient) Get(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("GET", path, query, nil)
}

// Post makes an authenticated POST request to the eToro API.
func (c *EtoroClient) Post(path string, body interface{}) (json.RawMessage, error) {
	return c.do("POST", path, nil, body)
}

// Put makes an authenticated PUT request to the eToro API.
func (c *EtoroClient) Put(path string, body interface{}) (json.RawMessage, error) {
	return c.do("PUT", path, nil, body)
}

// Delete makes an authenticated DELETE request to the eToro API.
func (c *EtoroClient) Delete(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("DELETE", path, query, nil)
}

// Patch makes an authenticated PATCH request to the eToro API.
func (c *EtoroClient) Patch(path string, body interface{}) (json.RawMessage, error) {
	return c.do("PATCH", path, nil, body)
}

func (c *EtoroClient) do(method, path string, query map[string]string, body interface{}) (json.RawMessage, error) {
	// Check rate limit before sending.
	c.mu.Lock()
	if c.rateLimitLeft <= 0 && time.Now().Before(c.rateLimitReset) {
		wait := time.Until(c.rateLimitReset)
		c.mu.Unlock()
		if debugAPI {
			fmt.Fprintf(os.Stderr, "[DEBUG] eToro rate limited, waiting %s\n", wait)
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
			fmt.Fprintf(os.Stderr, "[DEBUG] eToro %s %s body=%s\n", method, path, string(b))
		}
	} else if debugAPI {
		fmt.Fprintf(os.Stderr, "[DEBUG] eToro %s %s\n", method, path)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("x-user-key", c.userKey)
	req.Header.Set("x-request-id", newUUID())
	req.Header.Set("User-Agent", "etoro-cli/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Update rate limit state from response headers.
	c.updateRateLimit(resp.Header)

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if debugAPI {
		fmt.Fprintf(os.Stderr, "[DEBUG] eToro %s %s status=%d body=%.500s\n", method, path, resp.StatusCode, string(raw))
	}

	// Handle error status codes.
	if resp.StatusCode == 429 {
		retryAfter := resp.Header.Get("Retry-After")
		return nil, fmt.Errorf("rate limited (429): retry after %s seconds", retryAfter)
	}
	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed (401): check ETORO_API_KEY and ETORO_USER_KEY")
	}
	if resp.StatusCode == 403 {
		return nil, fmt.Errorf("forbidden (403): insufficient permissions for %s %s", method, path)
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found (404): %s %s", method, path)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %.500s", resp.StatusCode, string(raw))
	}

	// 204 No Content — successful deletion/update with no body.
	if resp.StatusCode == 204 || len(raw) == 0 {
		return json.RawMessage("null"), nil
	}

	return json.RawMessage(raw), nil
}

func (c *EtoroClient) updateRateLimit(h http.Header) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if v := h.Get("RateLimit-Remaining"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.rateLimitLeft = n
		}
	}
	if v := h.Get("RateLimit-Reset"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			c.rateLimitReset = time.Now().Add(time.Duration(secs) * time.Second)
		}
	}
}

// ── Config loading ───────────────────────────────────────────────────────────

type etoroCreds struct {
	apiKey  string
	userKey string
}

func loadConfig() (etoroCreds, error) {
	searchDirs := []string{
		filepath.Join("..", "..", "brokers", "eToro"),
		filepath.Join("brokers", "eToro"),
		filepath.Join(os.Getenv("HOME"), ".trade-kit", "etoro"),
	}

	cfg := etoroCreds{}
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
			case "ETORO_API_KEY":
				cfg.apiKey = v
			case "ETORO_USER_KEY":
				cfg.userKey = v
			}
		}
		if cfg.apiKey != "" && cfg.userKey != "" {
			break
		}
	}

	if cfg.apiKey == "" || cfg.userKey == "" {
		return cfg, fmt.Errorf(
			"eToro credentials not found.\n"+
				"Expected: brokers/eToro/.env with ETORO_API_KEY and ETORO_USER_KEY\n"+
				"Search paths: ../../brokers/eToro/.env, brokers/eToro/.env, ~/.trade-kit/etoro/.env\n"+
				"Get your API key at: https://api-portal.etoro.com\n"+
				"Have: api_key=%v user_key=%v",
			cfg.apiKey != "", cfg.userKey != "",
		)
	}
	return cfg, nil
}

// newUUID generates a v4 UUID string for x-request-id headers.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to timestamp if crypto/rand fails (should never happen).
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
