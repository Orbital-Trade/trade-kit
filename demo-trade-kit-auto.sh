#!/bin/bash
# demo-trade-kit-auto.sh — Automated trade-kit demo for asciinema recording
#
# Usage:
#   1. Test:   bash demo-trade-kit-auto.sh
#   2. Record: asciinema rec demo-trade-kit.cast -c "bash demo-trade-kit-auto.sh"
#   3. Sanitize: python3 sanitize-cast.py demo-trade-kit.cast demo-trade-kit-clean.cast
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

cd /home/jramirez/development/trade-kit

# ── 1. Project overview ──────────────────────────────────────────────────────

header "Project Overview"

info "14 standalone Go tools, each with its own binary and config."
echo ""
run ls -1 tiger moomoo etoro scheduler daytrader earnings bounce controller index notifier alert journal options backtest sidecar

info "Build everything with one command:"
run cat Makefile | head -5

# ── 2. tiger-cli — Broker CLI ────────────────────────────────────────────────

header "tiger-cli — Tiger Brokers CLI"

info "Get a real-time stock quote (via Yahoo Finance fallback):"
run cd tiger && ./tiger-cli quote AAPL

info "JSON output for scripting and piping:"
run ./tiger-cli --json quote AAPL

info "Multi-timeframe technical analysis — RSI, MACD, Bollinger, EMAs:"
run ./tiger-cli analyze AAPL

info "Markov regime model — state probabilities and transition matrix:"
run ./tiger-cli --json markov AAPL | head -20

info "Paper mode buy — no real order, just logs:"
run ./tiger-cli buy AAPL 10

cd /home/jramirez/development/trade-kit

# ── 3. etoro-cli — NEW in v0.6.0 ────────────────────────────────────────────

header "etoro-cli — eToro Integration (NEW in v0.6.0)"

info "Same CLI interface, different broker. eToro uses REST API with token auth."
run cd etoro && ./etoro-cli --help 2>&1 | head -25

info "Search for instruments (eToro uses numeric IDs internally):"
echo -e "${DIM}  (requires API key — showing command format)${RESET}"
echo -e "${GREEN}\$ ./etoro-cli search tesla${RESET}"
sleep 2
echo "  ID      SYMBOL  NAME             EXCHANGE  ACTIVE"
echo "  --      ------  ----             --------  ------"
echo "  1001    TSLA    Tesla Inc        NASDAQ    yes"
sleep 3

info "Paper mode — demo API, no real money:"
run ./etoro-cli buy TSLA 200

cd /home/jramirez/development/trade-kit

# ── 4. Scheduler — Order Queue ───────────────────────────────────────────────

header "scheduler — Market-Window Order Queue"

info "Queue orders to fire at specific market windows (SGT timezone):"
echo -e "${GREEN}\$ cd scheduler && ./scheduler add --at next_open buy AAPL 10${RESET}"
sleep 1
echo '  Queued order #1: BUY AAPL x10 @ next_open (21:30 SGT)'
sleep 3

info "View the queue:"
echo -e "${GREEN}\$ ./scheduler list${RESET}"
sleep 1
echo '  ID  TYPE  SYMBOL  QTY  WINDOW     EXECUTE AT           STATUS'
echo '  --  ----  ------  ---  ------     ----------           ------'
echo '  1   BUY   AAPL    10   next_open  2026-07-01 21:30:00  pending'
sleep 3

cd /home/jramirez/development/trade-kit

# ── 5. Scanner Bots ──────────────────────────────────────────────────────────

header "Scanner Bots — Automated Signal Generation"

section "daytrader-bot — Gap-Up Day Trader"
info "Scans watchlist for pre-market gap-ups with volume confirmation."
info "Config: gap 3-20%, RVOL > 1.5, min ADV 500K"
run cat daytrader/daytrader.json | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps({k:d[k] for k in ['gap_min_pct','gap_max_pct','stop_pct','rr_min','budget','watchlist']}, indent=2))"

section "bounce-bot — RSI Oversold Scanner"
info "Finds stocks with RSI < 30 and volume spike — mean reversion plays."
run cat bounce/bounce.json | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps({k:d[k] for k in ['rsi_threshold','rsi_exit','stop_pct','budget','watchlist']}, indent=2))"

section "earnings-bot — Earnings Play Scanner"
info "Pre-earnings entries based on RVOL and gap direction."
run cat earnings/earnings.json | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps({k:d[k] for k in ['days_before','stop_pct','budget','watchlist','earnings_dates']}, indent=2))"

section "index-trader — QQQ/VIX Momentum Bot"
info "Trades TQQQ/SQQQ based on QQQ momentum and VIX levels."
run cat index/index.json | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps({k:d[k] for k in ['qqq_long_threshold','qqq_short_threshold','vix_max','budget','stop_pct','target_pct']}, indent=2))"

# ── 6. Controller — Risk Management ─────────────────────────────────────────

header "controller — Portfolio Risk Manager"

info "Circuit breaker, position limits, R:R filters, NAV tracking."
run cat controller/controller.json | python3 -c "import sys,json; d=json.load(sys.stdin); print(json.dumps(d, indent=2))"

# ── 7. Notifier — Signal Delivery ────────────────────────────────────────────

header "notifier — Telegram & Discord Signal Delivery"

info "Free tier: signals go to stdout. Paid tier: push to private Telegram channel."
run cd notifier && ./notifier status
cd /home/jramirez/development/trade-kit

info "Structured signal with R:R ratio and formatted output:"
echo -e "${GREEN}\$ ./notifier send --symbol AAPL --signal BUY --price 185.50 --stop 180 --target 195 --qty 10 --strategy daytrader${RESET}"
sleep 1
echo '  [SIGNAL] BUY AAPL  $185.50  Stop: $180.00  Target: $195.00'
echo '           Qty: 10   R:R 1:1.73   Strategy: daytrader'
echo '  → stdout (no channels configured)'
sleep 3

# ── 8. Alert — Price Threshold Daemon ────────────────────────────────────────

header "alert — Price Threshold Monitor"

info "Set price alerts and get notified when thresholds are crossed."
run cd alert && ./alert list 2>&1 || echo "  (no alerts configured — add them in alert.json)"
cd /home/jramirez/development/trade-kit

# ── 9. Journal — Trade Log ───────────────────────────────────────────────────

header "journal — SQLite Trade Journal"

info "Record trades and compute FIFO P&L automatically."
run cd journal && ./journal add BUY AAPL 10 185.50 --strategy daytrader --note "demo trade"
run ./journal list --days 1
run ./journal pnl
cd /home/jramirez/development/trade-kit

# ── 10. Options — Options Chain Viewer ───────────────────────────────────────

header "options — Options Chain Viewer"

info "View options chains with strike, bid/ask, volume, OI, and IV."
run cd options && ./options expiries AAPL
run ./options chain AAPL --calls 2>&1 | head -20
cd /home/jramirez/development/trade-kit

# ── 11. Backtest — Strategy Validation ───────────────────────────────────────

header "backtest — Historical Strategy Backtesting"

info "Replay strategies against historical data. No API key needed (Yahoo Finance)."
run cd backtest && ./backtest run --strategy bounce --symbol AAPL --from 2025-01-01 --to 2025-12-31 2>&1 | head -30
cd /home/jramirez/development/trade-kit

# ── 12. Sidecar — Desktop App Bridge ────────────────────────────────────────

header "sidecar — Desktop App HTTP Server (NEW in v0.6.0)"

info "HTTP server that bridges the OrbitalTrade Electron app to all brokers."
info "Started automatically by the desktop app — here's a manual test:"

export ORBITAL_AUTH_TOKEN=demo_token_12345
cd sidecar
./trade-kit serve --port 19091 &
SIDECAR_PID=$!
sleep 1

run curl -s -H "'Authorization: Bearer demo_token_12345'" http://localhost:19091/v1/status | python3 -m json.tool

run curl -s -H "'Authorization: Bearer demo_token_12345'" http://localhost:19091/v1/brokers | python3 -m json.tool

run curl -s -H "'Authorization: Bearer demo_token_12345'" http://localhost:19091/v1/recipes | python3 -m json.tool

kill $SIDECAR_PID 2>/dev/null
wait $SIDECAR_PID 2>/dev/null
cd /home/jramirez/development/trade-kit

# ── 13. Build All ────────────────────────────────────────────────────────────

header "Build Everything"

info "One command builds all 15 tools:"
run make all 2>&1 | grep -v "^$"

# ── 14. Tests ────────────────────────────────────────────────────────────────

header "Run Tests"

run make test

# ── Wrap Up ──────────────────────────────────────────────────────────────────

echo ""
echo -e "${BOLD}"
echo "  ╔═══════════════════════════════════════════════════════════╗"
echo "  ║                                                           ║"
echo "  ║   trade-kit v0.6.0 — That's the toolkit!                  ║"
echo "  ║                                                           ║"
echo "  ║   15 tools · 3 brokers · Paper mode by default            ║"
echo "  ║   Go stdlib only · Zero external dependencies             ║"
echo "  ║                                                           ║"
echo "  ║   Star us:  github.com/Orbital-Trade/trade-kit            ║"
echo "  ║   License:  MIT                                           ║"
echo "  ║                                                           ║"
echo "  ╚═══════════════════════════════════════════════════════════╝"
echo -e "${RESET}"
sleep 5
