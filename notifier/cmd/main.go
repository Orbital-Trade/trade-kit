// notifier — trade signal delivery for trade-kit
//
// Sends trade signals to Telegram and/or Discord. All scanner bots call
// notifier to push signals to the user's phone. If notifier is not found
// in PATH the bots fall back to stdout logging — no crash.
//
// Usage:
//
//	notifier send "LUNR gap +8.3% — BUY signal"
//	notifier send --symbol LUNR --signal BUY --price 35.50 --stop 32.00 --qty 5
//	notifier send --symbol LUNR --signal BUY --price 35.50 --stop 32.00 --qty 5 --strategy daytrader
//	notifier test
//	notifier status
//
// Config: notifier.json in the working directory or ~/.trade-kit/notifier/notifier.json
//
// Free tier:  leave telegram_bot_token / discord_webhook_url blank → stdout only.
// Paid tier:  configure a private Telegram channel token for subscriber delivery.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Config holds notifier channel configuration.
type Config struct {
	TelegramBotToken  string `json:"telegram_bot_token"`
	TelegramChatID    string `json:"telegram_chat_id"`
	DiscordWebhookURL string `json:"discord_webhook_url"`
	Enabled           bool   `json:"enabled"`
}

// Result is the delivery outcome per channel.
type Result struct {
	Channel string
	OK      bool
	Error   string
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	cfg, err := loadConfig()
	if err != nil {
		fatalf("config: %v", err)
	}

	switch os.Args[1] {
	case "send":
		msg := buildMessage(os.Args[2:])
		if msg == "" {
			fatalf("send: no message provided\nUsage: notifier send \"<text>\" or notifier send --symbol SYM --signal BUY --price P --stop S")
		}
		results := deliver(cfg, msg)
		printResults(results)

	case "test":
		msg := fmt.Sprintf("✅ trade-kit notifier test — %s SGT", nowSGT())
		results := deliver(cfg, msg)
		printResults(results)
		allOK := true
		for _, r := range results {
			if !r.OK {
				allOK = false
			}
		}
		if len(results) == 0 {
			fmt.Println("No channels configured — add telegram_bot_token or discord_webhook_url to notifier.json")
		} else if allOK {
			fmt.Println("All channels delivered successfully.")
		} else {
			os.Exit(1)
		}

	case "status":
		cmdStatus(cfg)

	default:
		usage()
		os.Exit(1)
	}
}

// ─── MESSAGE BUILDER ─────────────────────────────────────────────────────────

// buildMessage parses args and returns a formatted signal message.
// Supports two forms:
//
//	notifier send "free text"
//	notifier send --symbol SYM --signal BUY|SELL --price P --stop S [--qty N] [--strategy name] [--note text]
func buildMessage(args []string) string {
	if len(args) == 0 {
		return ""
	}

	// Free-text form: first arg does not start with --
	if !strings.HasPrefix(args[0], "--") {
		return args[0]
	}

	// Structured form
	var symbol, signal, strategy, note string
	var price, stop, target float64
	var qty int

	for i := 0; i < len(args)-1; i++ {
		switch args[i] {
		case "--symbol":
			symbol = strings.ToUpper(args[i+1])
			i++
		case "--signal":
			signal = strings.ToUpper(args[i+1])
			i++
		case "--price":
			price, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		case "--stop":
			stop, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		case "--target":
			target, _ = strconv.ParseFloat(args[i+1], 64)
			i++
		case "--qty":
			qty, _ = strconv.Atoi(args[i+1])
			i++
		case "--strategy":
			strategy = args[i+1]
			i++
		case "--note":
			note = args[i+1]
			i++
		}
	}

	if symbol == "" || signal == "" {
		return ""
	}

	// Emoji per signal direction
	icon := "📊"
	switch signal {
	case "BUY", "LONG":
		icon = "🟢"
	case "SELL", "SHORT", "EXIT":
		icon = "🔴"
	case "STOP":
		icon = "🛑"
	case "ALERT":
		icon = "⚠️"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%s *%s* — %s", icon, symbol, signal))
	if qty > 0 {
		sb.WriteString(fmt.Sprintf(" x%d", qty))
	}
	if price > 0 {
		sb.WriteString(fmt.Sprintf(" @ $%.2f", price))
	}
	sb.WriteString("\n")

	if stop > 0 {
		sb.WriteString(fmt.Sprintf("  Stop: $%.2f", stop))
		if price > 0 && stop > 0 {
			risk := abs(price-stop) / price * 100
			sb.WriteString(fmt.Sprintf("  (–%.1f%%)", risk))
		}
		sb.WriteString("\n")
	}
	if target > 0 {
		sb.WriteString(fmt.Sprintf("  Target: $%.2f", target))
		if price > 0 && target > 0 {
			gain := abs(target-price) / price * 100
			sb.WriteString(fmt.Sprintf("  (+%.1f%%)", gain))
		}
		if stop > 0 && price > 0 && target > 0 {
			rr := abs(target-price) / abs(price-stop)
			sb.WriteString(fmt.Sprintf("  R:R %.1fx", rr))
		}
		sb.WriteString("\n")
	}
	if strategy != "" {
		sb.WriteString(fmt.Sprintf("  Strategy: %s\n", strategy))
	}
	if note != "" {
		sb.WriteString(fmt.Sprintf("  %s\n", note))
	}
	sb.WriteString(fmt.Sprintf("  %s SGT", nowSGT()))
	return sb.String()
}

// ─── DELIVERY ────────────────────────────────────────────────────────────────

// deliver sends msg to all configured channels. Always logs to stdout.
func deliver(cfg Config, msg string) []Result {
	// Always print to stdout regardless of channel config.
	fmt.Println(msg)

	if !cfg.Enabled {
		return nil
	}

	var results []Result

	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		err := sendTelegram(cfg.TelegramBotToken, cfg.TelegramChatID, msg)
		r := Result{Channel: "telegram"}
		if err != nil {
			r.Error = err.Error()
			fmt.Fprintf(os.Stderr, "notifier: telegram: %v\n", err)
		} else {
			r.OK = true
		}
		results = append(results, r)
	}

	if cfg.DiscordWebhookURL != "" {
		err := sendDiscord(cfg.DiscordWebhookURL, msg)
		r := Result{Channel: "discord"}
		if err != nil {
			r.Error = err.Error()
			fmt.Fprintf(os.Stderr, "notifier: discord: %v\n", err)
		} else {
			r.OK = true
		}
		results = append(results, r)
	}

	return results
}

// sendTelegram posts a message via the Telegram Bot API (sendMessage).
func sendTelegram(token, chatID, text string) error {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload, err := json.Marshal(map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	resp, err := httpPost(url, "application/json", payload)
	if err != nil {
		return err
	}

	var tgResp struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(resp, &tgResp); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if !tgResp.OK {
		return fmt.Errorf("telegram API error: %s", tgResp.Description)
	}
	return nil
}

// sendDiscord posts a message to a Discord webhook.
func sendDiscord(webhookURL, text string) error {
	// Discord markdown uses ** for bold; convert Telegram *bold* to **bold**
	discordText := strings.ReplaceAll(text, "*", "**")
	payload, err := json.Marshal(map[string]string{"content": discordText})
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	resp, err := httpPost(webhookURL, "application/json", payload)
	if err != nil {
		return err
	}

	// Discord returns 204 No Content on success; anything in body is an error
	if len(resp) > 0 {
		var discordErr struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		}
		if json.Unmarshal(resp, &discordErr) == nil && discordErr.Message != "" {
			return fmt.Errorf("discord error %d: %s", discordErr.Code, discordErr.Message)
		}
	}
	return nil
}

// httpPost is a thin wrapper around http.Post with a timeout and body reader.
func httpPost(url, contentType string, body []byte) ([]byte, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, contentType, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP POST: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return raw, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return raw, nil
}

// ─── STATUS ──────────────────────────────────────────────────────────────────

func cmdStatus(cfg Config) {
	fmt.Println("notifier status")
	fmt.Println()

	if !cfg.Enabled {
		fmt.Println("  ⏸  Disabled (enabled=false in config)")
		return
	}

	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		fmt.Printf("  ✅ Telegram   chat_id=%s\n", cfg.TelegramChatID)
	} else {
		fmt.Println("  ⬜ Telegram   not configured (set telegram_bot_token + telegram_chat_id)")
	}

	if cfg.DiscordWebhookURL != "" {
		short := cfg.DiscordWebhookURL
		if len(short) > 40 {
			short = short[:20] + "..." + short[len(short)-10:]
		}
		fmt.Printf("  ✅ Discord    webhook=%s\n", short)
	} else {
		fmt.Println("  ⬜ Discord    not configured (set discord_webhook_url)")
	}

	fmt.Println()
	if cfg.TelegramBotToken == "" && cfg.DiscordWebhookURL == "" {
		fmt.Println("  No channels configured — signals go to stdout only.")
		fmt.Println("  Edit notifier.json to add Telegram or Discord delivery.")
	}
}

// ─── CONFIG ──────────────────────────────────────────────────────────────────

func loadConfig() (Config, error) {
	cfg := Config{Enabled: true}

	searchPaths := []string{
		"notifier.json",
		filepath.Join(os.Getenv("HOME"), ".trade-kit", "notifier", "notifier.json"),
	}

	for _, p := range searchPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse %s: %w", p, err)
		}
		return cfg, nil
	}

	// No config file found — stdout-only mode is fine.
	return cfg, nil
}

// ─── HELPERS ─────────────────────────────────────────────────────────────────

func printResults(results []Result) {
	for _, r := range results {
		if r.OK {
			fmt.Printf("  ✅ %s delivered\n", r.Channel)
		} else {
			fmt.Printf("  ❌ %s failed: %s\n", r.Channel, r.Error)
		}
	}
}

func nowSGT() string {
	return time.Now().UTC().Add(8 * time.Hour).Format("02 Jan 15:04")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "notifier: "+format+"\n", args...)
	os.Exit(1)
}

func usage() {
	fmt.Print(`notifier — trade signal delivery for trade-kit

Sends signals to Telegram and/or Discord. Falls back to stdout if
no channels are configured (free tier).

COMMANDS
  send "<text>"                              Send free-text message
  send --symbol SYM --signal BUY|SELL        Send structured signal
       [--price P] [--stop S] [--target T]
       [--qty N] [--strategy name] [--note text]
  test                                       Send test message to all channels
  status                                     Show configured channels

EXAMPLES
  notifier send "LUNR gap +8.3% — BUY signal"
  notifier send --symbol LUNR --signal BUY --price 35.50 --stop 32.00 --qty 5
  notifier send --symbol NVDA --signal SELL --price 920.00 --strategy earnings
  notifier test
  notifier status

CONFIG  (notifier.json)
  {
    "telegram_bot_token": "",   ← BotFather token
    "telegram_chat_id":  "",   ← channel or chat ID (use @channelusername or numeric)
    "discord_webhook_url": "",  ← Discord channel webhook URL
    "enabled": true
  }

`)
}
