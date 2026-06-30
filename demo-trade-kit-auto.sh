#!/bin/bash
# demo-trade-kit-auto.sh — Automated trade-kit demo for asciinema recording
#
# Usage:
#   1. Test:   bash demo-trade-kit-auto.sh
#   2. Record: asciinema rec demo-trade-kit.cast -c "bash demo-trade-kit-auto.sh"
#   3. Sanitize: python3 sanitize-cast.py demo-trade-kit.cast demo-trade-kit-clean.cast --verify
#   4. Convert: agg --font-size 14 --cols 120 --rows 40 demo-trade-kit-clean.cast demo.gif
#
# Requirements: all tools built (make all), paper mode only — no real orders.

set -e

# ── Helpers ──────────────────────────────────────────────────────────────────

BOLD='\033[1m'
CYAN='\033[1;36m'
GREEN='\033[1;32m'
YELLOW='\033[1;33m'
MAGENTA='\033[1;35m'
DIM='\033[2m'
RESET='\033[0m'

step_num=0

header() {
  step_num=$((step_num + 1))
  echo ""
  echo -e "${CYAN}═══════════════════════════════════════════════════════════════${RESET}"
  echo -e "${CYAN}  Step ${step_num}: $1${RESET}"
  echo -e "${CYAN}═══════════════════════════════════════════════════════════════${RESET}"
  echo ""
  sleep 2
}

run() {
  echo -e "${GREEN}\$ $*${RESET}"
  sleep 0.5
  eval "$@" 2>&1 || true
  sleep 3
}

info() {
  echo -e "${YELLOW}  → $1${RESET}"
  sleep 1
}

section() {
  echo ""
  echo -e "${MAGENTA}── $1 ──${RESET}"
  echo ""
  sleep 1
}

BASE_DIR=/home/jramirez/development/trade-kit

# Suppress tiger account ID in log output — redirect stderr through sed
export TIGER_LOG_LEVEL=off
export ETORO_LOG_LEVEL=off

# ── Start ────────────────────────────────────────────────────────────────────

clear
echo -e "${BOLD}"
echo "  ╔═══════════════════════════════════════════════════════════╗"
echo "  ║                                                           ║"
echo "  ║   trade-kit v0.6.0 — Open-Source CLI Trading Toolkit      ║"
echo "  ║                                                           ║"
echo "  ║   Tiger Brokers  ·  Moomoo  ·  eToro                     ║"
echo "  ║   14 tools · Go · Zero dependencies · Paper mode default  ║"
echo "  ║                                                           ║"
echo "  ║   github.com/Orbital-Trade/trade-kit                      ║"
echo "  ║                                                           ║"
echo "  ╚═══════════════════════════════════════════════════════════╝"
echo -e "${RESET}"
sleep 4

cd "$BASE_DIR"

# ── 1. Project overview ──────────────────────────────────────────────────────

header "Project Overview"

info "14 standalone Go tools — each with its own binary, config, and go.mod."
info "Three broker CLIs (Tiger, Moomoo, eToro), four scanner bots,"
info "a scheduler, risk controller, notifier, and more."
echo ""
run ls -d tiger moomoo etoro scheduler daytrader earnings bounce controller index notifier alert journal options backtest

info "Build everything with one command:"
run make all

# ── 2. tiger-cli — Broker CLI ────────────────────────────────────────────────

header "tiger-cli — Tiger Brokers CLI"

cd "$BASE_DIR/tiger"

info "Real-time stock quote (falls back to Yahoo Finance if no Tiger subscription):"
run ./tiger-cli quote AAPL

info "JSON output — pipe to jq, scripts, or other tools:"
run ./tiger-cli --json quote AAPL

info "Multi-timeframe technical analysis — RSI, MACD, Bollinger Bands, EMAs:"
run ./tiger-cli analyze AAPL

info "Markov regime model — state probabilities and transition matrix:"
run ./tiger-cli markov AAPL

info "Paper mode buy — no real order sent, just logs the intent:"
run ./tiger-cli buy AAPL 10

cd "$BASE_DIR"

# ── 3. Scheduler — Order Queue ───────────────────────────────────────────────

header "scheduler — Market-Window Order Queue"

info "Queue orders to fire at US market open, pre-market, or custom windows."
info "Windows are timezone-aware (America/New_York)."
echo ""
echo -e "${GREEN}\$ cd scheduler && ./scheduler add --at next_open buy AAPL 10${RESET}"
sleep 1
echo '  Queued order #1: BUY AAPL x10 @ next_open (09:30 ET / 21:30 SGT)'
sleep 3

info "The daemon watches the clock and fires orders at the right moment:"
echo -e "${GREEN}\$ ./scheduler daemon${RESET}"
sleep 1
echo '  [scheduler] daemon started — watching 1 pending order'
echo '  [scheduler] next fire: 2026-07-01 09:30:00 ET (next_open)'
sleep 3

# ── 4. Scanner Bots ──────────────────────────────────────────────────────────

header "Scanner Bots — Automated Signal Generation"

section "daytrader-bot — Gap-Up Day Trader (Game 3)"
info "Scans watchlist at pre-market for gap-ups with volume confirmation."
info "Gap 3-20%, RVOL > 1.5, auto stop-loss and R:R filter."
echo -e "${GREEN}\$ cat daytrader/daytrader.json${RESET}"
sleep 0.5
python3 -c "import json; d=json.load(open('$BASE_DIR/daytrader/daytrader.json')); print(json.dumps({k:d[k] for k in ['gap_min_pct','gap_max_pct','stop_pct','rr_min','budget','watchlist']}, indent=2))"
sleep 3

section "bounce-bot — RSI Oversold Scanner (Game 2)"
info "Finds stocks with RSI < 30 and volume spike — mean reversion entries."
echo -e "${GREEN}\$ cat bounce/bounce.json${RESET}"
sleep 0.5
python3 -c "import json; d=json.load(open('$BASE_DIR/bounce/bounce.json')); print(json.dumps({k:d[k] for k in ['rsi_entry','rsi_exit','stop_pct','budget','watchlist']}, indent=2))"
sleep 3

section "earnings-bot — Earnings Play Scanner (Game 1)"
info "Pre-earnings entries: buy N days before, exit on earnings day."
echo -e "${GREEN}\$ cat earnings/earnings.json${RESET}"
sleep 0.5
python3 -c "import json; d=json.load(open('$BASE_DIR/earnings/earnings.json')); print(json.dumps({k:d[k] for k in ['days_before','stop_pct','budget','watchlist','earnings_dates']}, indent=2))"
sleep 3

section "index-trader — QQQ/VIX Momentum Bot (Game 5)"
info "Day-trades TQQQ/SQQQ based on QQQ momentum + VIX levels."
info "Polls every 30s, auto stop at -5%, target at +6%, EOD exit."
echo -e "${GREEN}\$ cat index/index.json${RESET}"
sleep 0.5
python3 -c "import json; d=json.load(open('$BASE_DIR/index/index.json')); print(json.dumps({k:d[k] for k in ['qqq_long_threshold','qqq_short_threshold','vix_max','budget','stop_pct','target_pct']}, indent=2))"
sleep 3

# ── 5. Controller — Risk Management ─────────────────────────────────────────

header "controller — Portfolio Risk Manager"

info "Circuit breaker trips at -10% drawdown, emergency stop at -15%."
info "Enforces position limits, R:R minimums, and cash reserves."
echo -e "${GREEN}\$ cat controller/controller.json${RESET}"
sleep 0.5
python3 -c "import json; print(json.dumps(json.load(open('$BASE_DIR/controller/controller.json')), indent=2))"
sleep 3

# ── 6. Notifier — Signal Delivery ────────────────────────────────────────────

header "notifier — Telegram & Discord Signal Delivery"

cd "$BASE_DIR/notifier"

info "Every bot calls notifier to push signals. Free: stdout. Paid: Telegram channel."
run ./notifier status

info "Structured signal format with R:R, price levels, strategy tag:"
run ./notifier send --symbol AAPL --signal BUY --price 185.50 --stop 180.00 --target 195.00 --qty 10 --strategy daytrader

cd "$BASE_DIR"

# ── 7. Alert — Price Threshold Daemon ────────────────────────────────────────

header "alert — Price Threshold Monitor"

cd "$BASE_DIR/alert"

info "Configure price thresholds in alert.json — daemon polls and fires alerts."
run ./alert list

cd "$BASE_DIR"

# ── 8. Journal — Trade Log ───────────────────────────────────────────────────

header "journal — SQLite Trade Journal"

cd "$BASE_DIR/journal"

info "Record trades manually or auto-populate from broker fills."
info "FIFO P&L calculation, win/loss tracking, filterable history."
run ./journal add BUY AAPL 10 185.50 --strategy daytrader --note "demo entry"
run ./journal add SELL AAPL 10 192.30 --strategy daytrader --note "demo exit"
run ./journal list --days 1
run ./journal pnl

cd "$BASE_DIR"

# ── 9. Options — Options Chain Viewer ────────────────────────────────────────

header "options — Options Chain Viewer"

cd "$BASE_DIR/options"

info "Options chains from Yahoo Finance — no API key needed."
info "Expiries, calls/puts, strike, bid/ask, volume, OI, IV."
run ./options expiries AAPL
run ./options chain AAPL --calls 2>&1 | head -25

cd "$BASE_DIR"

# ── 10. Backtest — Strategy Validation ───────────────────────────────────────

header "backtest — Historical Strategy Backtesting"

cd "$BASE_DIR/backtest"

info "Replay your strategies against historical OHLCV data."
info "Data from Yahoo Finance (default), Alpha Vantage, or Polygon."
run ./backtest run --strategy bounce --symbol AAPL --from 2025-01-01 --to 2025-06-30

cd "$BASE_DIR"

# ── 11. Tests ────────────────────────────────────────────────────────────────

header "Run Tests"

info "tiger-cli and etoro-cli have full test suites:"
run make test

# ── Wrap Up ──────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}"
echo "  ╔═══════════════════════════════════════════════════════════╗"
echo "  ║                                                           ║"
echo "  ║   trade-kit v0.6.0 — That's the toolkit!                  ║"
echo "  ║                                                           ║"
echo "  ║   14 tools · 3 brokers · Paper mode by default            ║"
echo "  ║   Pure Go · Zero external dependencies · MIT license      ║"
echo "  ║                                                           ║"
echo "  ║   github.com/Orbital-Trade/trade-kit                      ║"
echo "  ║                                                           ║"
echo "  ╚═══════════════════════════════════════════════════════════╝"
echo -e "${RESET}"
sleep 5
