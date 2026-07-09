[![Go 1.21+](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/Orbital-Trade/trade-kit)](https://github.com/Orbital-Trade/trade-kit/releases)
[![Stars](https://img.shields.io/github/stars/Orbital-Trade/trade-kit?style=social)](https://github.com/Orbital-Trade/trade-kit)

```
  ████████╗██████╗  █████╗ ██████╗ ███████╗
  ╚══██╔══╝██╔══██╗██╔══██╗██╔══██╗██╔════╝
     ██║   ██████╔╝███████║██║  ██║█████╗
     ██║   ██╔══██╗██╔══██║██║  ██║██╔══╝
     ██║   ██║  ██║██║  ██║██████╔╝███████╗
     ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚═════╝ ╚══════╝
  ██╗  ██╗██╗████████╗
  ██║ ██╔╝██║╚══██╔══╝
  █████╔╝ ██║   ██║
  ██╔═██╗ ██║   ██║
  ██║  ██╗██║   ██║
  ╚═╝  ╚═╝╚═╝   ╚═╝
```

**Open-source multi-broker CLI toolkit for retail traders.**

Connect your Tiger, Moomoo, or eToro account. Scan, backtest, manage risk — from the terminal or through Claude.

- **15 tools** — 3 broker CLIs, 4 scanner bots, backtester, options viewer, scheduler, risk controller, notifier, journal, alerts
- **Paper mode by default** — nothing touches your broker until you pass `--live`
- **AI-ready** — Claude Code can run scans, backtests, and analysis through trade-kit directly
- **Zero external dependencies** — pure Go stdlib, single binary per tool
- **Shared signal bus** — bots coordinate via a single `signals.json` file
- **Strategy packs** — pre-built configs for dividends, earnings, index scalps

Built for retail traders in Singapore, Hong Kong, and the US who want to automate their workflow.

> ⚠️ See [DISCLAIMER.md](DISCLAIMER.md) — this is not financial advice.

---

## Tools

| Tool | Binary | What it does |
|---|---|---|
| tiger | `tiger-cli` | Tiger Brokers — positions, quotes, orders, technical analysis, Markov model |
| moomoo | `moomoo-cli` | Moomoo/Futu — same interface, pure Go TCP client via OpenD |
| etoro | `etoro-cli` | eToro — REST API, demo/live mode, watchlists, price alerts |
| scheduler | `scheduler` | Order queue daemon — fire orders at market windows (SGT/ET) |
| daytrader | `daytrader-bot` | Gap-up scanner — pre-market gap plays with RVOL filter |
| earnings | `earnings-bot` | Earnings scanner — pre-announcement run-up entries |
| bounce | `bounce-bot` | RSI oversold bounce — mean reversion with volume confirmation |
| index | `index-trader` | Index momentum — QQQ/VIX signals for TQQQ/SQQQ day trades |
| controller | `controller` | Portfolio risk manager — circuit breaker, NAV tracking, kill switch |
| backtest | `backtest` | Historical strategy validation — replay against OHLCV data |
| options | `options` | Options chain viewer — calls/puts, IV, OI via Yahoo Finance |
| notifier | `notifier` | Signal delivery — Telegram and Discord push notifications |
| alert | `alert` | Price threshold daemon — polls and fires on cross |
| journal | `journal` | Trade journal — SQLite log with FIFO P&L reporting |
| sidecar | `trade-kit` | HTTP server — bridges the OrbitalTrade desktop app to all brokers |

Jump to: [Setup](#setup) · [Strategy packs](#strategy-packs) · [Workflows](#workflow-playbooks) · [Signal bus](#the-signal-bus) · [Troubleshooting](#troubleshooting) · [Support this project](#support-this-project)

---

## Strategy packs

Ready-to-use config + watchlist bundles so you don't start from a blank file. Copy one, point a bot at it, and tune from there.

| Pack | For | Strategy |
|---|---|---|
| [SG Dividend](packs/sg-dividend/) | `alert`, `journal`, `controller` | SGX blue-chip dividend payers + large-cap S-REITs — income, long-term hold |
| [US Earnings](packs/us-earnings/) | `earnings-bot` | Liquid US large-caps — pre-announcement run-up |
| [Index Momentum](packs/index-momentum/) | `index-trader` | QQQ trend + VIX regime → TQQQ/SQQQ |

```bash
earnings-bot --config packs/us-earnings/earnings.json
```

See [packs/](packs/) for details.

---

## Setup

### Quick start

```bash
git clone https://github.com/Orbital-Trade/trade-kit
cd trade-kit
make all          # builds every tool
./tiger-cli --help
```

Requires Go 1.21+. Each tool compiles to a single binary with zero external dependencies.

<details>
<summary><strong>Tiger Brokers credentials</strong></summary>

1. Apply for Tiger Open API access at [openapi.tigersecurities.com](https://openapi.tigersecurities.com)
2. Download your RSA key pair from the developer portal
3. Create `~/.trade-kit/tiger/.env`:

```env
TIGER_ID=<your tiger developer ID>
PRIVATE_KEY=<base64-encoded PKCS8 RSA private key>
TRADE_PASSWORD=<your 6-digit trade PIN>
```

4. Test connection:

```bash
./tiger-cli positions       # paper mode — safe, reads only
./tiger-cli --live positions  # your real account
```

</details>

<details>
<summary><strong>Moomoo credentials</strong></summary>

1. Download and start [Futu OpenD](https://www.futunn.com/download/OpenD) — it runs as a local daemon on port 11111
2. Create `~/.trade-kit/moomoo/.env`:

```env
MOOMOO_HOST=127.0.0.1
MOOMOO_PORT=11111
TRADE_PASSWORD=<your 6-digit trade PIN>
ACC_ID=<your account ID>
```

3. Start OpenD, then test:

```bash
futu-opend &              # start OpenD in background
./moomoo-cli positions    # paper mode
```

</details>

---

## Paper · Semi · Live

Every write command has three modes. Paper is always the default.

| Mode | Flag | What happens |
|---|---|---|
| Paper | (default) | Prints what would happen — nothing sent to broker |
| Live | `--live` | Sends to broker after you type `Y` to confirm |
| Semi (bots only) | `--semi` | Bot prompts you before each trade signal |

```bash
./tiger-cli buy AAPL 10              # paper — safe, no order sent
./tiger-cli --live buy AAPL 10       # live — prompts: Execute? [y/N]
./daytrader-bot --semi run           # bot asks before each trade
./daytrader-bot --live run           # bot auto-executes every signal
```

---

## 1. tiger-cli

The Tiger Brokers command-line client. Reads positions, quotes, and orders. Places buy/sell/stop/modify orders. Runs technical analysis.

### Quick start

```bash
./tiger-cli positions                 # see open positions
./tiger-cli quote AAPL                # live price
./tiger-cli orders                    # open orders
./tiger-cli analyze AAPL              # multi-timeframe TA
```

### Read commands

These are always safe — no orders placed.

#### `positions`
List open stock positions.

```bash
./tiger-cli positions
./tiger-cli --json positions          # JSON output for scripting
```

Output: symbol, shares, avg cost, market price, market value, unrealised P&L, realised P&L.

#### `account`
Cash, buying power, and net liquidation value.

```bash
./tiger-cli account
```

#### `quote <SYMBOL>`
Real-time price snapshot.

```bash
./tiger-cli quote AAPL
./tiger-cli quote ES3.SI              # Singapore-listed stock
./tiger-cli --json quote NVDA
```

Output: price, change%, open, high, low, volume, previous close.

#### `orders`
All open and pending orders.

```bash
./tiger-cli orders
./tiger-cli --json orders
```

Output: order ID, symbol, action, type, qty, filled, limit, stop, TIF, status.

#### `analyze <SYMBOL> [--futures]`
Multi-timeframe technical analysis: RSI, MACD, Bollinger Bands, EMA across 1D / 1H / 15m.

```bash
./tiger-cli analyze AAPL              # US stock
./tiger-cli analyze ES3.SI            # Singapore stock
./tiger-cli analyze MNQ --futures     # Micro E-mini Nasdaq futures
./tiger-cli --json analyze AAPL       # machine-readable output
```

Output per timeframe:
- RSI(14) — overbought > 70, oversold < 30
- MACD line, signal, histogram
- Bollinger %B (position within the bands)
- EMA 20 / 50 / 200 and price relationship
- Bias score: **BULLISH** / **NEUTRAL** / **BEARISH**

Overall: bull count, bear count, alignment (**BULLISH** / **MIXED** / **BEARISH**).

> **Note:** RSI > 70 and %B > 1.0 are scored as overextension, not strength. This reflects mean-reversion risk at extremes.

#### `markov <SYMBOL>`
Markov regime model — tomorrow's state probabilities based on two years of daily returns.

```bash
./tiger-cli markov NVDA
./tiger-cli markov ES3.SI
./tiger-cli --json markov AAPL
```

Output:
- Current state: **BULL** / **SIDE** / **BEAR** (based on trailing 20-day return, ±5% thresholds)
- Historical distribution of days in each state
- 3×3 transition matrix (sticky states highlighted where P(self) > 50%)
- Forecasts at 1d / 5d / 10d via matrix exponentiation
- Signal: direction + confidence (HIGH / MEDIUM / LOW), P(bull) − P(bear)

### Write commands

These require `--live` for real execution. Paper mode logs the action only.

#### `buy <SYMBOL> <SHARES> [--limit <price>] [--stop <price>] [--target <price>]`

```bash
./tiger-cli --live buy AAPL 10                            # market buy
./tiger-cli --live buy AAPL 10 --limit 180.00             # limit buy
./tiger-cli --live buy AAPL 10 --limit 180 --stop 174     # buy + stop-loss
./tiger-cli --live buy AAPL 10 --limit 180 --stop 174 --target 198
```

- Market buy: DAY order, fills at best available price
- Limit buy: fills at your price or better (DAY)
- `--stop`: places a separate GTC stop-loss order after the entry fills
- `--target`: places a separate GTC take-profit limit order after entry fills

#### `sell <SYMBOL> <SHARES> [--limit <price>]`

```bash
./tiger-cli --live sell AAPL 10                           # market sell
./tiger-cli --live sell AAPL 10 --limit 195.00            # limit sell (GTC)
```

Limit sell stays open until filled or you cancel it.

#### `stop <SYMBOL> <SHARES> --price <price>`
Place a GTC stop-loss on an existing position.

```bash
./tiger-cli --live stop AAPL 100 --price 174.00
```

#### `target <SYMBOL> <SHARES> --price <price>`
Place a GTC take-profit limit on an existing position.

```bash
./tiger-cli --live target AAPL 100 --price 198.00
```

#### `cancel <ORDER_ID>`

```bash
./tiger-cli --live cancel 43116645746347008
```

#### `modify <ORDER_ID> [--limit <p>] [--stop <p>] [--qty <n>] [--tif DAY|GTC]`
Modify an open order in-place (no cancel + re-enter).

```bash
./tiger-cli --live modify 43116645746347008 --limit 182.00
./tiger-cli --live modify 43116645746347008 --stop 176.00 --qty 8
```

Requires at least one of: `--limit`, `--stop`, `--qty`, `--tif`.

### Futures commands

For micro futures: MES (Micro E-mini S&P 500), MNQ (Micro E-mini Nasdaq), M2K (Micro Russell 2000).

```bash
# Enter long MES at 5100, protective stop at 5090
./tiger-cli --live futures entry MES long 1 --entry 5100 --stop 5090

# Enter short MNQ at 16500, stop at 16600
./tiger-cli --live futures entry MNQ short 2 --entry 16500 --stop 16600

# Market-close an open position
./tiger-cli --live futures close MES long 1

# Trail the stop — cancel old, place new
./tiger-cli --live futures update-stop MES long 1 --stop 5095 --old-id 789012
```

`futures entry` always places the protective stop immediately after the entry. If the stop order fails, an emergency market close fires automatically.

### Playbook: Morning check ritual

Run this every morning before the open to see where you stand:

```bash
./tiger-cli account            # buying power available
./tiger-cli positions          # what you're holding
./tiger-cli orders             # any open limits/stops
./tiger-cli analyze AAPL       # quick TA on a position
```

---

## 2. moomoo-cli

Same command interface as tiger-cli. Uses the Futu OpenD TCP protocol — no Python, no bridge.

Futu OpenD must be running before any order commands.

### Symbol format

Auto-detected from the ticker pattern:

| Input | Interpreted as |
|---|---|
| `AAPL` | `US.AAPL` |
| `Z74`, `D05` | `SG.Z74`, `SG.D05` (alpha + 2 digits) |
| `558`, `1810` | `SG.558`, `HK.1810` |
| `HK.00700` | `HK.00700` (explicit prefix) |

### Commands (identical to tiger-cli)

```bash
./moomoo-cli positions
./moomoo-cli account
./moomoo-cli quote Z74
./moomoo-cli orders

./moomoo-cli --live buy Z74 100 --limit 4.50 --stop 4.20
./moomoo-cli --live sell Z74 100 --limit 4.80
./moomoo-cli --live stop Z74 100 --price 4.20
./moomoo-cli --live cancel FS1C71DE6D05BA1000
./moomoo-cli --live modify FS1C71DE6D05BA1000 --limit 4.60
```

### Playbook: SGX dividend stock management

```bash
# Check your Singapore blue-chip positions
./moomoo-cli positions
./moomoo-cli quote Z74         # Singtel
./moomoo-cli quote D05         # DBS Bank

# Add a trailing stop after a 5% gain
./moomoo-cli --live stop Z74 1000 --price 4.30

# Take partial profit
./moomoo-cli --live sell Z74 500 --limit 4.90
```

---

## 3. scheduler

A persistent order queue. You add orders at any time (even when the market is closed), and the daemon fires them automatically when the target market window opens.

Market windows:

| Window | Time | Use case |
|---|---|---|
| `next_open` | 9:30 AM ET / 21:30 SGT | Default — regular open |
| `pre_market` | 4:00 AM ET / 16:00 SGT | Pre-market executions |
| `pre_open` | 9:20 AM ET / 21:20 SGT | 10 min before open |
| `morning` | 7:00 AM SGT / 23:00 UTC | SGT morning scan trigger |
| `now` | Immediate | Next daemon tick |

### Add orders to the queue

```bash
# Basic buy/sell (defaults to next_open)
scheduler add buy AAPL 10
scheduler add sell AAPL 5
scheduler add buy SCHD 4 --limit 31.65

# Specify window
scheduler add --at pre_market sell QUBT 15
scheduler add --at next_open buy SCHD 4 --limit 31.65 --note "dividend reinvestment"

# Stop-loss and take-profit
scheduler add stop EXTR 9 --price 23.50
scheduler add target NVDA 5 --price 160.00

# Modify or cancel an existing Tiger order
scheduler add modify 43116645746347008 --stop 23.50
scheduler add modify 43116645746347008 --limit 155.00 --qty 50
scheduler add cancel 43116645746347008

# Run a shell command at market open
scheduler add exec "./daytrader-bot scan"
scheduler add --at pre_open exec "./daytrader-bot scan"

# Re-queue the same command every day (--daily)
scheduler add --at pre_open exec "./daytrader-bot --live scan" --daily
```

### Manage the queue

```bash
scheduler list             # show all pending and completed orders
scheduler ls               # alias for list

scheduler cancel a3f2b1    # remove a pending order by its 8-character ID
scheduler clear            # cancel all pending orders at once
```

### Start the daemon

The daemon checks every 30 seconds and fires any due orders.

```bash
scheduler daemon &                         # background
scheduler daemon --log ~/scheduler.log &   # with log file

# Check it's running
tail -f ~/scheduler.log
```

The daemon re-reads the queue file on every tick, so orders added after it starts are picked up automatically. You don't need to restart it.

### Time header

Every `scheduler` command (except `daemon`) prints a time header:

```
  Mon 25 May 21:32:14 SGT  |  09:32:14 ET  |  🟢 market open  →  market close in 6h 27m
```

### Playbook: Night-before queue

You're in Singapore and want to set up orders before you go to sleep:

```bash
# Queue tonight's orders (it's 11 PM SGT, US market opens in 10 hours)
scheduler add buy SCHD 4 --limit 31.65 --note "weekly buy"
scheduler add sell QUBT 15 --note "take profit"
scheduler add modify 43116645746347008 --stop 23.50 --note "trail stop"

# Start the daemon (if not already running)
scheduler daemon --log ~/scheduler.log &

# Confirm what's queued
scheduler list

# Go to sleep — daemon fires at 9:30 AM ET / 9:30 PM SGT
```

### Playbook: Automate daily scans

Run the gap scanner every day at pre-open (9:20 AM ET) automatically:

```bash
# Queue a daily exec order — re-queues itself after each run
scheduler add --at pre_open exec "./daytrader-bot scan" --daily

# Start the daemon
scheduler daemon --log ~/scheduler.log &
```

---

## 4. daytrader-bot

Gap-up day trade scanner. Scans your watchlist for stocks gapping 3–20% at the open with above-normal volume (RVOL > 1.5). Enters on the first pullback, exits hard by 11:00 AM ET.

### Config — `daytrader.json`

```json
{
  "gap_min_pct": 3.0,
  "gap_max_pct": 20.0,
  "stop_pct": 2.0,
  "rr_min": 3.0,
  "budget": 200.0,
  "min_adv": 500000,
  "scan_interval_sec": 60,
  "watchlist": ["LUNR", "RKLB", "ASTS", "KTOS", "IONQ", "RDW", "QQQ"]
}
```

| Field | What it controls |
|---|---|
| `gap_min_pct` | Minimum gap% to qualify (3% default) |
| `gap_max_pct` | Maximum gap% — filters runaway movers (20% default) |
| `stop_pct` | Stop-loss placed this % below entry (2% default) |
| `rr_min` | Skip if risk/reward ratio is below this (3.0 = 3:1) |
| `budget` | Max capital per trade in USD |
| `min_adv` | Minimum average daily volume to avoid illiquid stocks |
| `scan_interval_sec` | How often to re-scan during `run` mode (60s) |
| `watchlist` | Symbols to watch — add anything you follow |

### Commands

#### `scan`
One-time scan — prints signals, no action.

```bash
./daytrader-bot scan
./daytrader-bot --paper scan     # explicit paper (same as default)
./daytrader-bot --earnings scan  # use earnings watchlist + tighter params
```

Output for each symbol:
- `▶ LONG` — setup found: gap%, qty, entry price, stop, target
- `◎ WATCH` — interesting but criteria not fully met (shows reason)
- `—` — no setup, skipped

#### `run`
Continuous scan + execute loop. Runs until you stop it.

```bash
./daytrader-bot run              # paper (safe)
./daytrader-bot --semi run       # prompts before each trade
./daytrader-bot --live run       # auto-executes
```

What it does every tick:
1. Fetches price, gap%, and RVOL for each symbol in watchlist
2. Generates entry signals for qualifying setups
3. Checks pending signals: skips if price has drifted >2% from signal
4. Places buy limit + immediate stop-loss (live/semi only)
5. At `exit_by_min` (11:00 AM ET): market-sells all open positions

#### `monitor`
Print current signal bus state — useful in a second terminal while `run` is looping.

```bash
./daytrader-bot monitor
```

#### `status`
Print open trades and closed P&L.

```bash
./daytrader-bot status
```

#### `close <SYMBOL>`
Manually exit a position at market.

```bash
./daytrader-bot close LUNR
```

### Earnings mode

Adds today's earnings stocks to the watchlist (from `earnings.json`), tightens the RVOL filter, and allows shorts on gap-down:

```bash
./daytrader-bot --earnings scan
./daytrader-bot --earnings --live run
```

Earnings mode params:
- `gap_min_pct`: 5% (bigger gap required)
- `stop_pct`: 3% (tighter stop)
- `rvol_min`: 2.0x (higher volume bar)
- Exit by: 10:30 AM ET instead of 11:00 AM ET
- Shorts allowed on gap-down earnings misses

### Playbook: Pre-market gap trade

```bash
# 9:15 AM ET — pre-market, run a scan to see what's gapping
./daytrader-bot scan

# Found LUNR gapping +8.3% with RVOL 3.2x — looks good
# 9:30 AM ET — market opens, run in semi mode to approve each trade
./daytrader-bot --semi run

# Bot prompts: "LUNR: BUY 5 @ limit $35.50, stop $34.79? [y/N]"
# Type y — order placed
# Position closes automatically by 11:00 AM ET
```

---

## 5. earnings-bot

Earnings momentum scanner. Buys N days before an earnings announcement to capture the pre-announcement run-up. Sells on the day of earnings (before the report) to avoid event risk.

### Config — `earnings.json`

```json
{
  "days_before": 3,
  "stop_pct": 5.0,
  "budget": 200.0,
  "min_adv": 500000,
  "min_price": 3.0,
  "max_run_pct": 20.0,
  "scan_interval_sec": 300,
  "watchlist": ["NVDA", "LUNR", "RKLB", "OKLO"],
  "earnings_dates": {
    "LUNR": "2026-05-14",
    "NVDA": "2026-05-20"
  }
}
```

| Field | What it controls |
|---|---|
| `days_before` | Buy this many days before earnings date (3 default) |
| `stop_pct` | Stop-loss % below entry (5% default) |
| `max_run_pct` | Skip if stock already ran >20% into earnings |
| `budget` | Max capital per trade |
| `earnings_dates` | Override or add earnings dates (format: YYYY-MM-DD) |

Keep `earnings_dates` updated — the bot will not enter a trade for a symbol without a date.

### Commands

#### `scan`
Evaluate watchlist and print signals.

```bash
./earnings-bot scan
```

Output per symbol:
- `▶ ENTER` — N days before earnings, setup met: qty, entry, stop
- `◀ EXIT` — earnings day, sell now: position summary
- `◎ WATCH` — upcoming but criteria not met (too early, already ran)
- `—` — no earnings date, or not applicable

#### `run`
Continuous scan + execute loop.

```bash
./earnings-bot run
./earnings-bot --semi run
./earnings-bot --live run
```

Logic per tick:
1. Scan every `scan_interval_sec` (5 min default)
2. Generate ENTER signals when `N days_before` an earnings date
3. Generate EXIT signals on the earnings date itself
4. Execute fresh ENTER signals (checks drift <3% and no existing position)
5. Execute EXIT signals (sells at market)

#### `monitor` / `status` / `close`
Same as daytrader-bot:

```bash
./earnings-bot monitor        # signal bus
./earnings-bot status         # trade store + P&L
./earnings-bot close NVDA     # manual exit
```

### Playbook: Pre-earnings momentum trade

```bash
# Check what's coming up
./earnings-bot scan

# NVDA reports in 3 days — ENTER signal generated
# Run in semi mode (you confirm each entry)
./earnings-bot --semi run

# Bot: "NVDA: BUY 2 @ limit $882.50, stop $838.38? [y/N]"
# Type y

# Three days later — earnings day, EXIT signal fires automatically
# Bot sells before the report to avoid event risk
```

---

## 6. bounce-bot

RSI oversold bounce scanner. Buys when a stock's RSI drops below 20 with above-normal volume — a setup that historically reverts to the mean. Exits when RSI recovers to 50, or after 5 trading days, or at the stop-loss.

### Config — `bounce.json`

```json
{
  "rsi_entry": 20.0,
  "rsi_exit": 50.0,
  "stop_pct": 5.0,
  "max_hold_days": 5,
  "budget": 150.0,
  "min_adv": 500000,
  "min_price": 3.0,
  "scan_interval_sec": 300,
  "watchlist": ["RKLB", "LUNR", "ASTS", "IONQ", "KTOS", "RDW"]
}
```

| Field | What it controls |
|---|---|
| `rsi_entry` | Enter when RSI drops below this (20 default — deep oversold) |
| `rsi_exit` | Exit when RSI recovers above this (50 default — neutral) |
| `stop_pct` | Hard stop-loss below entry |
| `max_hold_days` | Force-close after this many trading days |
| `budget` | Max capital per trade |

### Commands

#### `scan`
One-time RSI scan.

```bash
./bounce-bot scan
```

Output: `▶ ENTER` with RSI value, qty, entry, stop — or `—` (not oversold enough).

#### `run`
Continuous scan, execute, and monitor exits.

```bash
./bounce-bot run
./bounce-bot --semi run
./bounce-bot --live run
```

Exit logic (checked every tick):
- RSI > 50: take profit, close position
- Stop hit: loss limited
- 5 trading days elapsed: time exit

#### `monitor` / `status` / `close`

```bash
./bounce-bot monitor
./bounce-bot status
./bounce-bot close LUNR
```

### Playbook: Catching a dip

```bash
# Scan for oversold setups in your watchlist
./bounce-bot scan

# RKLB RSI = 17.3, RVOL 2.1x — ENTER signal
# ASTS RSI = 22.1 — not oversold enough (above threshold of 20)

# Run in semi mode and approve the RKLB entry
./bounce-bot --semi run

# Bot will auto-exit when RSI recovers to 50 (usually 1–3 days)
# Or stops you out at -5% if the dip continues
```

---

## 7. index-trader

QQQ/VIX momentum signal generator. Monitors QQQ direction and VIX level every 30 seconds. Generates **LONG TQQQ** or **SHORT SQQQ** signals based on index momentum. Exits automatically at target (+6%), stop (-5%), or hard EOD (12:30 PM ET default).

This is a scalp tool — it trades 3x leveraged ETFs intraday and closes the same session.

### Config — `index.json`

```json
{
  "qqq_long_threshold": 0.3,
  "qqq_short_threshold": -0.3,
  "vix_max": 25.0,
  "vix_spike_min": 20.0,
  "budget": 200.0,
  "tqqq_shares": 2,
  "sqqq_shares": 4,
  "stop_pct": 5.0,
  "target_pct": 6.0,
  "grace_period_min": 10,
  "exit_by_min": 750,
  "scan_interval_sec": 30
}
```

| Field | What it controls |
|---|---|
| `qqq_long_threshold` | QQQ must be up >0.3% to signal LONG TQQQ |
| `qqq_short_threshold` | QQQ must be down >0.3% to signal SHORT SQQQ |
| `vix_max` | Skip all trades if VIX is above this (too volatile) |
| `tqqq_shares` | Shares to buy when going long |
| `sqqq_shares` | Shares to buy when going short (inverse, so more shares) |
| `stop_pct` | Hard stop: 5% below entry |
| `target_pct` | Take profit: 6% above entry |
| `grace_period_min` | No trades in first N minutes (default: first 10 min after open) |
| `exit_by_min` | Hard EOD exit time in minutes since midnight (750 = 12:30 PM ET) |

### Modes

```bash
./index-trader -mode watch      # observe signals, no action (default)
./index-trader -mode semi       # confirm before trade (10s timeout)
./index-trader -mode live       # auto-execute via tiger-cli
```

### Signal logic (every 30 seconds)

1. Fetch QQQ and VIX prices via Yahoo Finance
2. Skip if: pre-market, after 12:30 PM ET, VIX ≥ 25, in grace period (first 10 min)
3. If QQQ > +0.3% → signal: **BUY TQQQ**
4. If QQQ < -0.3% → signal: **BUY SQQQ**
5. If in position: check stop/target on every tick, print live P&L
6. Exit: when stop/target hit, or hard exit at 12:30 PM ET

### Playbook: Index scalp session

```bash
# 9:15 AM ET — fire up the watcher
./index-trader -mode watch

# 9:30 AM ET — market opens, grace period starts (no trades for 10 min)
# Output: [09:30:14] QQQ +0.21% | VIX 18.3 | 🟡 grace period (8m left)

# 9:40 AM ET — grace period ends
# Output: [09:40:14] QQQ +0.44% | VIX 17.8 | → LONG signal: BUY 2 TQQQ @ $82.50

# Switch to semi when you've watched enough patterns
./index-trader -mode semi

# Auto-exits at target +6% or stop -5%, latest 12:30 PM ET
```

### Tips

- Watch in `watch` mode for several sessions before going semi or live
- Raise `vix_max` threshold if you want to trade in choppier markets
- Lower `grace_period_min` to 5 on trend days, raise to 20 on gap-and-crap days
- `tqqq_shares` and `sqqq_shares` control position size — start small

---

## 8. controller

The portfolio risk manager. Does not place orders — it monitors your positions and signals, enforces risk limits, and gives you a single dashboard view before you make any decision.

Run `controller status` before every session.

### Config — `controller.json`

```json
{
  "t1_symbols": ["Z74", "ES3", "D05", "O39", "SGOV"],
  "position_max_pct": 30.0,
  "cash_reserve_pct": 15.0,
  "max_positions": 6,
  "circuit_breaker_alert_pct": 10.0,
  "circuit_breaker_estop_pct": 15.0,
  "rr_min_swing": 3.0,
  "rr_min_scalp": 2.0
}
```

| Field | What it controls |
|---|---|
| `t1_symbols` | Your "Track 1" stable/dividend stocks (blue-chips, ETFs) |
| `position_max_pct` | No single position can exceed this % of NAV (30%) |
| `cash_reserve_pct` | Always keep at least this % in cash (15%) |
| `max_positions` | Max concurrent open positions (6) |
| `circuit_breaker_alert_pct` | Alert when session drawdown exceeds this (10%) |
| `circuit_breaker_estop_pct` | Emergency stop threshold (15%) |
| `rr_min_swing` | Minimum R:R for swing signals (earnings, bounce) |
| `rr_min_scalp` | Minimum R:R for scalp signals (daytrader, index) |

### Commands

#### `status`
Full portfolio dashboard. Run this every morning.

```bash
./controller status
```

The dashboard shows ten panels:

1. **NAV TRAJECTORY** — Today's drawdown vs circuit breaker thresholds  
   `✅ OK` / `🔴 ALERT –10%` / `🚨 ESTOP –15%`

2. **PLAYBOOK GATE** — Can you open a new trade right now?  
   Shows: position count vs max, cash reserve %, max position size, max next-trade capital

3. **SIGNAL EXPIRY** — Pending signals with countdown bars  
   `💀 expired` / `🔴 < 2 hours` / `🟡 < 24 hours`

4. **PATTERN** — Market activity classification  
   `A ⚡` 3+ strategies firing (capital-constrained, prioritise by R:R)  
   `B 💤` No signals (cash is a position)  
   `C 🎯` Single signal (cleanest case, full focus)

5. **POSITIONS** — T1 (stable/dividend) and T2 (tactical) split

6. **TRACK 2 CAPITAL** — Deployment view: estimated T2 cash vs deployed

7. **OPEN ORDERS** — All pending stops and limit orders

8. **SIGNAL BUS** — All bot signals: pending, active, filled, expired  
   Each shows R:R check (`⚠️` if below minimum, `✅` if OK)

9. **BOOTSTRAP FILLED** — Recent profits with the 40/60 reinvestment split

10. **DAILY LOG** — Written to `logs/daily-YYYY-MM-DD.json`

#### `gate [cost]`
Quick pre-flight check before opening a trade.

```bash
./controller gate         # just show limits
./controller gate 200     # check if a $200 trade is allowed right now
```

Blocks if any of these are true:
- Already at max positions (6)
- Trade would exceed 30% of NAV
- Cash reserve would drop below 15%

#### `monitor`
Signal bus detail — sorted by status, newest first.

```bash
./controller monitor
```

Useful while bots are running — shows every pending/active/filled signal with R:R, expiry countdown, and notes.

#### `bootstrap <profit>`
Calculate how to split a closed profit (40% to savings, 60% reinvested).

```bash
./controller bootstrap 500
```

Output:
```
Net profit:        $500.00
40% → savings:     $200.00
60% → reinvest:    $300.00
   T1 (dividend):  $150.00
   T2 (tactical):  $150.00
```

#### `estop`
**Emergency stop.** Cancels all open orders and market-sells all positions.

```bash
./controller estop
```

You must type `CONFIRM` to proceed. Use this only in a crisis (flash crash, rogue daemon, unexpected news).

### Playbook: Morning risk review

```bash
# 9:00 AM ET — before anything else
./controller status

# Check the gate before entering any trade
./controller gate 200

# If gate is GREEN and pattern is C (single signal), proceed
# If gate is RED (at max positions), don't open anything

# End of day — log the session
./controller status    # writes to logs/daily-YYYY-MM-DD.json
```

### Playbook: Circuit breaker protocol

```bash
./controller status
# Panel 1 shows: 🔴 ALERT — NAV drawdown –10.5%

# Alert level: stop opening new positions
# Review all pending signals
./controller monitor

# If it hits –15%
./controller status
# Panel 1 shows: 🚨 ESTOP — NAV drawdown –15.2%

# Emergency stop
./controller estop
# Type CONFIRM
```

---

## Workflow playbooks

### Playbook: Full morning routine (SGT)

This is the complete morning-before-open routine. Takes about 5 minutes.

```bash
# 9:00 PM SGT (9:00 AM ET, 30 min before open)

# 1. Check your P&L and account
./tiger-cli account
./tiger-cli positions
./tiger-cli orders

# 2. Run portfolio risk check
./controller status

# 3. Scan for setups
./daytrader-bot scan        # gap plays (reads Yahoo Finance)
./bounce-bot scan           # RSI oversold setups
./earnings-bot scan         # pre-earnings entries (if any due)

# 4. Check the gate before acting on any signal
./controller gate 200

# 5. Open your chosen mode in a dedicated terminal
./daytrader-bot --semi run  # opens at 9:30 ET, prompts before each trade
```

### Playbook: Set it and sleep (automated)

Queue tonight's orders and let the daemon handle everything.

```bash
# 10:00 PM SGT (night before)

# Queue orders
scheduler add buy SCHD 4 --limit 31.65
scheduler add sell QUBT 10

# Set up daily pre-market scan (runs every day until you cancel it)
scheduler add --at pre_open exec "./daytrader-bot scan" --daily
scheduler add --at next_open exec "./daytrader-bot --live run" --daily

# Start daemon (if not already running)
scheduler daemon --log ~/scheduler.log &

# Confirm everything is queued correctly
scheduler list

# Go to sleep — daemon fires at 9:30 PM SGT (9:30 AM ET)
```

### Playbook: Multi-bot session

Running multiple strategies simultaneously with the controller watching over all of them.

```bash
# Terminal 1 — controller dashboard (refresh manually)
watch -n 30 ./controller status

# Terminal 2 — gap plays
./daytrader-bot --live run

# Terminal 3 — RSI bounces
./bounce-bot --live run

# Terminal 4 — index scalp (watch mode first, go semi after 30 min)
./index-trader -mode watch

# Terminal 5 — scheduler daemon (fires any queued orders)
scheduler daemon --log ~/scheduler.log
```

### Playbook: Earnings week

Set up for a week with heavy earnings reports:

```bash
# 1. Update earnings dates
# Edit earnings/earnings.json → add this week's reporters
{
  "earnings_dates": {
    "NVDA": "2026-05-20",
    "RKLB": "2026-05-19",
    "OKLO": "2026-05-18"
  }
}

# 2. Run earnings scan + daytrader scan together (earnings adds to daytrader watchlist)
./earnings-bot scan
./daytrader-bot --earnings scan

# 3. Run both bots in semi mode during the session
./earnings-bot --semi run &
./daytrader-bot --earnings --semi run &

# 4. Monitor signals
./controller monitor
```

---

## The signal bus

All scanner bots share a single `signals.json` file in the working directory. This is how the bots and controller communicate.

```
daytrader-bot  ─┐
earnings-bot   ─┼──→  signals.json  ←── controller monitors
bounce-bot     ─┘
```

**A signal** is a trade candidate written by a bot. It contains:

```json
{
  "id": "c08af385",
  "symbol": "LUNR",
  "strategy": "earnings",
  "action": "enter",
  "entry_limit": 34.20,
  "stop": 32.49,
  "qty": 6,
  "reason": "3 days before earnings, RVOL 1.9x",
  "expires_at": "2026-05-20T20:00:00Z",
  "status": "pending"
}
```

Status lifecycle: `pending` → `active` (entry order placed) → `filled` / `expired` / `rejected` / `declined`

You can manually add signals to `signals.json` with any text editor — the bots and controller will pick them up on the next scan.

---

## File reference

```
Working directory:
  signals.json          ← shared signal bus (auto-created)
  order-queue.json      ← scheduler queue (auto-created, gitignored)
  nav-history.json      ← controller session baseline (auto-created)
  logs/                 ← daily P&L logs (auto-created by controller)
  daytrader.json        ← daytrader config
  earnings.json         ← earnings config + dates
  bounce.json           ← bounce config
  controller.json       ← risk limits config
  index.json            ← index trader config

Credentials:
  ~/.trade-kit/tiger/.env
  ~/.trade-kit/moomoo/.env
```

---

## Troubleshooting

| Symptom | Likely cause | Fix |
|---|---|---|
| `Tiger credentials not found` | `.env` file missing or wrong path | Check `~/.trade-kit/tiger/.env` exists with correct keys |
| `connection refused` (moomoo) | Futu OpenD not running | `futu-opend &` |
| `Order ID 0` | Tiger API rejected the order | Check symbol is valid US stock, account has buying power |
| `SKIPPED: no position held` | Scheduler sell guard triggered | You already sold, or symbol spelling mismatch |
| `signal expired` | Signal older than expiry window | Re-run the scan to generate a fresh signal |
| `Gate blocked — max positions` | 6+ open positions | Close something before opening new trades |
| `drift >2%` | Price moved away from signal before execution | Signal is stale — rescan |
| `yahoo_kline: no data` | Yahoo Finance rate limit or bad symbol | Wait 60s, retry, or check ticker spelling |
| `build failed` | Go version < 1.21 | `go version`, upgrade if needed |

### Debug mode (tiger-cli)

Set `TIGER_LOG_LEVEL=debug` to see raw API request/response payloads:

```bash
TIGER_LOG_LEVEL=debug ./tiger-cli quote AAPL
```

---

## Versioning

Semantic versioning. See [CHANGELOG.md](CHANGELOG.md).

Current version: **v0.5.0**

---

## Support this project

trade-kit is free and MIT-licensed. If it's useful, here's how to support it — at no cost to you:

### Open a broker account through a referral link

Both brokers trade-kit talks to run referral programs with welcome rewards for you and a credit for the project.

- **Tiger Brokers (SG)** — [Sign up with code F3MUAJ](https://www.tigerbrokers.com.sg/?invite=F3MUAJ) · up to S$1,000 in welcome rewards for new accounts
- **Moomoo (SG)** — [Sign up](https://j.moomoo.com/0CXYTY) · free stocks + cash coupons for new accounts

### Charts & analysis

- **TradingView** — [Get started](https://www.tradingview.com/?aff_id=168034&source=trade-kit) · the charting platform most trade-kit users already rely on. New users get a $15 coupon toward any plan.

### Sponsor on GitHub

If trade-kit saves you time, consider [sponsoring on GitHub](https://github.com/sponsors/Orbital-Trade). Sponsorship funds new bots, broker integrations, and pre-built binaries.

### Use the hosted platform

trade-kit is the open-source core of [**OrbitalTrade**](https://trade.orbitalpay.ai) — a hosted trading-intelligence platform with AI-generated theses, a live multi-market scanner (US/SGX/HK), a copilot, and a browser extension.

- 🧩 [OrbitalTrade for Chrome](https://chromewebstore.google.com/detail/orbitaltrade/kfndmcgcalllbgjiebgjhmefhfoiimde) — free
- 🌐 [trade.orbitalpay.ai](https://trade.orbitalpay.ai) — free tools, blog, and paid tiers for higher limits

---

## Roadmap

- [ ] `notifier` — Telegram/Discord signal delivery
- [ ] `alert` — price alert daemon (monitors watchlist for threshold triggers)
- [ ] `journal` — trade journal + P&L stats (SQLite, auto-populated from brokers)
- [ ] `backtest` — historical strategy validation via Yahoo Finance OHLCV
- [ ] `options` — options chain viewer (calls/puts, IV, OI)
- [ ] GoReleaser CI — pre-built binaries for macOS, Linux, Windows

---

## License

MIT — see [LICENSE](LICENSE).

Not financial advice — see [DISCLAIMER.md](DISCLAIMER.md).
