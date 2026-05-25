// Package client provides a standalone Tiger Brokers REST API client.
//
// No external dependencies — only Go stdlib. Credentials are loaded from the
// existing brokers/Tiger/.env and tiger_openapi_config.properties files.
//
// Usage:
//
//	c, err := client.New(paperMode)
//	data, err := c.Call("positions", map[string]interface{}{"account": c.Account(), ...})
package client

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// debugAPI enables raw API response logging when TIGER_LOG_LEVEL=debug.
var debugAPI = strings.ToLower(os.Getenv("TIGER_LOG_LEVEL")) == "debug"

const baseURL = "https://openapi.tigerfintech.com/hkg/gateway"

// TigerClient is the low-level REST transport. Operations live in ops/.
type TigerClient struct {
	tigerID       string
	account       string
	tradePassword string
	privateKey    *rsa.PrivateKey
	http          *http.Client
	paper         bool
	contractCache map[string]string
}

// New loads credentials and returns a ready-to-use client.
// Set paper=true to log orders without sending them.
func New(paper bool) (*TigerClient, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	key, err := parseKey(cfg.privateKey)
	if err != nil {
		return nil, fmt.Errorf("private key: %w", err)
	}
	return &TigerClient{
		tigerID:       cfg.tigerID,
		account:       cfg.account,
		tradePassword: cfg.tradePassword,
		privateKey:    key,
		http:          &http.Client{Timeout: 15 * time.Second},
		paper:         paper,
		contractCache: make(map[string]string),
	}, nil
}

// Account returns the Tiger account number.
func (c *TigerClient) Account() string { return c.account }

// IsPaper returns true when running in paper/simulation mode.
func (c *TigerClient) IsPaper() bool { return c.paper }

// Call makes a signed REST request to the Tiger gateway.
// method is the Tiger API method name (e.g. "place_order", "positions").
// bizContent is marshaled to JSON and included in the signed payload.
func (c *TigerClient) Call(method string, bizContent interface{}) (json.RawMessage, error) {
	body, err := json.Marshal(bizContent)
	if err != nil {
		return nil, err
	}

	ts := time.Now().Format("2006-01-02 15:04:05")
	params := map[string]string{
		"tiger_id":    c.tigerID,
		"method":      method,
		"charset":     "UTF-8",
		"sign_type":   "RSA",
		"timestamp":   ts,
		"version":     "1.0",
		"biz_content": string(body),
	}
	sig, err := c.sign(params)
	if err != nil {
		return nil, err
	}
	params["sign"] = sig

	if debugAPI {
		log.Printf("[DEBUG] Tiger %s biz_content=%s", method, string(body))
	}
	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	req, err := http.NewRequest("POST", baseURL, strings.NewReader(string(reqBody)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", "tiger-cli/1.0")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var tr struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &tr); err != nil {
		return nil, fmt.Errorf("parse response: %w (%.200s)", err, raw)
	}
	if tr.Code != 0 {
		return nil, fmt.Errorf("API error %d: %s", tr.Code, tr.Message)
	}

	if debugAPI {
		log.Printf("[DEBUG] Tiger %s biz=%s resp=%s", method, string(body), string(tr.Data))
	}

	// Tiger double-encodes list responses: the data field is a JSON string
	// whose content is the actual JSON (e.g. data = "\"[{...}]\"").
	// This happens for positions, orders, and similar list endpoints.
	// Detect any JSON string value and unwrap it once.
	//   ""           → null   (no results)
	//   "\"[...]\""  → [...]  (the actual list)
	//   "\"{...}\""  → {...}  (wrapped object)
	if len(tr.Data) > 0 && tr.Data[0] == '"' {
		var inner string
		if err := json.Unmarshal(tr.Data, &inner); err == nil {
			if inner == "" {
				return json.RawMessage("null"), nil
			}
			if debugAPI {
				log.Printf("[DEBUG] Tiger %s (unwrapped) → %s", method, inner)
			}
			return json.RawMessage(inner), nil
		}
	}

	return tr.Data, nil
}

// ResolveFuturesContract maps a root symbol (MES, MNQ, M2K) to the active
// contract code (e.g. MES2506). Falls back to a computed quarterly code.
func (c *TigerClient) ResolveFuturesContract(symbol string) (string, error) {
	if cached, ok := c.contractCache[symbol]; ok {
		return cached, nil
	}

	data, err := c.Call("future_contracts", map[string]interface{}{
		"type": symbol,
		"lang": "en_US",
	})
	if err != nil {
		fb := c.fallbackContract(symbol)
		c.contractCache[symbol] = fb
		return fb, nil
	}

	var result struct {
		Items []struct {
			ContractCode  string `json:"contractCode"`
			ContractMonth string `json:"contractMonth"`
			Trade         bool   `json:"trade"`
		} `json:"items"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		// also try direct array
		var items []struct {
			ContractCode  string `json:"contractCode"`
			ContractMonth string `json:"contractMonth"`
			Trade         bool   `json:"trade"`
		}
		if err2 := json.Unmarshal(data, &items); err2 == nil {
			result.Items = items
		}
	}

	now := time.Now().Format("200601")
	best, bestMonth := "", ""
	for _, item := range result.Items {
		if len(item.ContractMonth) != 6 {
			continue
		}
		if item.ContractMonth >= now && item.Trade {
			if bestMonth == "" || item.ContractMonth < bestMonth {
				bestMonth = item.ContractMonth
				best = item.ContractCode
			}
		}
	}
	if best == "" {
		best = c.fallbackContract(symbol)
	}
	c.contractCache[symbol] = best
	return best, nil
}

func (c *TigerClient) fallbackContract(symbol string) string {
	now := time.Now()
	for _, m := range []int{3, 6, 9, 12} {
		if int(now.Month()) <= m {
			return fmt.Sprintf("%s%02d%02d", symbol, now.Year()%100, m)
		}
	}
	return fmt.Sprintf("%s%02d03", symbol, (now.Year()+1)%100)
}

// ── RSA signing ──────────────────────────────────────────────────────────────

func (c *TigerClient) sign(params map[string]string) (string, error) {
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+params[k])
	}
	content := strings.Join(parts, "&")

	h := sha1.Sum([]byte(content))
	sig, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA1, h[:])
	if err != nil {
		return "", fmt.Errorf("RSA sign: %w", err)
	}
	return base64.StdEncoding.EncodeToString(sig), nil
}

// ── Config loading ───────────────────────────────────────────────────────────

type tigerCreds struct {
	tigerID       string
	account       string
	privateKey    string
	tradePassword string
}

func loadConfig() (tigerCreds, error) {
	// Search for brokers/Tiger/ relative to the project root or HOME.
	searchDirs := []string{
		filepath.Join("..", "..", "brokers", "Tiger"),
		filepath.Join("brokers", "Tiger"),
		filepath.Join(os.Getenv("HOME"), ".trade-kit", "tiger"),
	}

	cfg := tigerCreds{}
	for _, dir := range searchDirs {
		for _, fname := range []string{".env", "tiger_openapi_config.properties"} {
			path := filepath.Join(dir, fname)
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
				case "TIGER_ID", "tiger_id":
					cfg.tigerID = v
				case "account":
					cfg.account = v
				case "PRIVATE_KEY", "private_key_pk8":
					cfg.privateKey = v
				case "TRADE_PASSWORD":
					cfg.tradePassword = v
				}
			}
		}
		if cfg.tigerID != "" && cfg.account != "" && cfg.privateKey != "" {
			break
		}
	}

	if cfg.tigerID == "" || cfg.account == "" || cfg.privateKey == "" {
		return cfg, fmt.Errorf(
			"Tiger credentials not found.\n"+
				"Expected: brokers/Tiger/.env and tiger_openapi_config.properties\n"+
				"Have: tiger_id=%v account=%v key=%v",
			cfg.tigerID != "", cfg.account != "", cfg.privateKey != "",
		)
	}
	return cfg, nil
}

func parseKey(b64 string) (*rsa.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode: %w", err)
	}
	// Try PKCS8 first, fall back to PKCS1.
	if k, err := x509.ParsePKCS8PrivateKey(raw); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, fmt.Errorf("key is not RSA")
	}
	return x509.ParsePKCS1PrivateKey(raw)
}
