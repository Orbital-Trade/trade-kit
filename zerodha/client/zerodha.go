// Package client provides a standalone Zerodha Kite Connect REST API client.
//
// No external dependencies — only Go stdlib.
//
// Auth: Authorization: token {api_key}:{access_token} + X-Kite-Version: 3
//
// Base URL: https://api.kite.trade
//
// Usage:
//
//	c, err := client.New(paperMode)
//	data, err := c.Get("/user/profile", nil)
package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var debugAPI = strings.ToLower(os.Getenv("ZERODHA_LOG_LEVEL")) == "debug"

const baseURL = "https://api.kite.trade"

// ZerodhaClient is the low-level REST transport.
type ZerodhaClient struct {
	apiKey      string
	accessToken string
	http        *http.Client
	paper       bool
	baseURL     string
}

// New loads credentials from .env and returns a ready-to-use client.
func New(paper bool) (*ZerodhaClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	return newClient(cfg.apiKey, cfg.accessToken, paper), nil
}

// NewFromCreds constructs a client from explicit credentials.
// Used by the sidecar server.
func NewFromCreds(apiKey, accessToken string, paper bool) (*ZerodhaClient, error) {
	if apiKey == "" || accessToken == "" {
		return nil, fmt.Errorf("zerodha: api_key and access_token are required")
	}
	return newClient(apiKey, accessToken, paper), nil
}

func newClient(apiKey, accessToken string, paper bool) *ZerodhaClient {
	return &ZerodhaClient{
		apiKey:      apiKey,
		accessToken: accessToken,
		http:        &http.Client{Timeout: 15 * time.Second},
		paper:       paper,
		baseURL:     baseURL,
	}
}

// IsPaper returns true when running in paper mode (no real orders).
func (c *ZerodhaClient) IsPaper() bool { return c.paper }

// Get makes an authenticated GET request.
func (c *ZerodhaClient) Get(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("GET", path, query, nil)
}

// PostForm makes an authenticated POST request with form-encoded body.
func (c *ZerodhaClient) PostForm(path string, params map[string]string) (json.RawMessage, error) {
	return c.doForm("POST", path, params)
}

// PutForm makes an authenticated PUT request with form-encoded body.
func (c *ZerodhaClient) PutForm(path string, params map[string]string) (json.RawMessage, error) {
	return c.doForm("PUT", path, params)
}

// Delete makes an authenticated DELETE request.
func (c *ZerodhaClient) Delete(path string, query map[string]string) (json.RawMessage, error) {
	return c.do("DELETE", path, query, nil)
}

func (c *ZerodhaClient) do(method, path string, query map[string]string, body io.Reader) (json.RawMessage, error) {
	u := c.baseURL + path
	if len(query) > 0 {
		parts := make([]string, 0, len(query))
		for k, v := range query {
			parts = append(parts, k+"="+v)
		}
		u += "?" + strings.Join(parts, "&")
	}

	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "token "+c.apiKey+":"+c.accessToken)
	req.Header.Set("X-Kite-Version", "3")
	req.Header.Set("User-Agent", "zerodha-cli/1.0")

	if debugAPI {
		fmt.Fprintf(os.Stderr, "[DEBUG] zerodha %s %s\n", method, path)
	}

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
		fmt.Fprintf(os.Stderr, "[DEBUG] zerodha %s %s status=%d body=%.500s\n", method, path, resp.StatusCode, string(raw))
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429): retry later")
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("auth failed (%d): check ZERODHA_API_KEY and ZERODHA_ACCESS_TOKEN", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found (404): %s %s", method, path)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %.500s", resp.StatusCode, string(raw))
	}
	if resp.StatusCode == 204 || len(raw) == 0 {
		return json.RawMessage("null"), nil
	}

	return json.RawMessage(raw), nil
}

func (c *ZerodhaClient) doForm(method, path string, params map[string]string) (json.RawMessage, error) {
	form := url.Values{}
	for k, v := range params {
		form.Set(k, v)
	}
	encoded := form.Encode()

	if debugAPI {
		fmt.Fprintf(os.Stderr, "[DEBUG] zerodha %s %s body=%s\n", method, path, encoded)
	}

	req, err := http.NewRequest(method, c.baseURL+path, strings.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "token "+c.apiKey+":"+c.accessToken)
	req.Header.Set("X-Kite-Version", "3")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "zerodha-cli/1.0")

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
		fmt.Fprintf(os.Stderr, "[DEBUG] zerodha %s %s status=%d body=%.500s\n", method, path, resp.StatusCode, string(raw))
	}

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429): retry later")
	}
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return nil, fmt.Errorf("auth failed (%d): check ZERODHA_API_KEY and ZERODHA_ACCESS_TOKEN", resp.StatusCode)
	}
	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("not found (404): %s %s", method, path)
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

type zerodhaCreds struct {
	apiKey      string
	accessToken string
}

func loadConfig() (zerodhaCreds, error) {
	searchDirs := []string{
		filepath.Join("..", "..", "brokers", "Zerodha"),
		filepath.Join("brokers", "Zerodha"),
		filepath.Join(os.Getenv("HOME"), ".trade-kit", "zerodha"),
	}

	cfg := zerodhaCreds{}
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
			case "ZERODHA_API_KEY":
				cfg.apiKey = v
			case "ZERODHA_ACCESS_TOKEN":
				cfg.accessToken = v
			}
		}
		if cfg.apiKey != "" && cfg.accessToken != "" {
			break
		}
	}

	if cfg.apiKey == "" || cfg.accessToken == "" {
		return cfg, fmt.Errorf(
			"Zerodha credentials not found.\n"+
				"Expected: brokers/Zerodha/.env with ZERODHA_API_KEY and ZERODHA_ACCESS_TOKEN\n"+
				"Search paths: ../../brokers/Zerodha/.env, brokers/Zerodha/.env, ~/.trade-kit/zerodha/.env\n"+
				"Get your keys at: https://developers.kite.trade\n"+
				"Have: api_key=%v access_token=%v",
			cfg.apiKey != "", cfg.accessToken != "",
		)
	}
	return cfg, nil
}
