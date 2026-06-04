# daytrader-bot — Game 3: Gap-Up Day Trade

Stocks that gap up 3–20% at open on real news pull back in the first 15 minutes as early
buyers take profit. Buy the pullback during the 9:35–10:00 AM ET window. Hard exit by 11 AM.

## ⚠️ PDT Warning

The US Pattern Day Trader rule requires $25,000 minimum equity to make 4+ day trades per week.
**Tiger Brokers Singapore accounts may be PDT-exempt** — verify with Tiger support before using
`--live`. Use `--paper` or `--semi` until confirmed.

## Strategy

| Parameter | Value |
|---|---|
| Gap threshold | 3% – 20% from previous close |
| Entry window | 9:35 AM – 10:00 AM ET |
| Stop | 2% below entry |
| Target | Prior high or 2:1 R/R (whichever is closer) |
| Hard exit | 11:00 AM ET (no afternoon holds) |
| Budget | $200 max per trade |
| Volume filter | Avg daily volume ≥ 1,000,000 |
| Price filter | ≥ $3.00 |

**Do not chase >20% gaps.** The stock has already run too far and is likely to reverse hard.

## Build

```bash
cd tools/daytrader
go build -o daytrader-bot ./cmd/
```

## Commands

```bash
# Scan for gap-ups (best run at 9:30–9:45 AM ET)
daytrader-bot scan

# Continuous loop with auto time-exit at 11 AM
daytrader-bot run
daytrader-bot --semi run     # confirm each trade (recommended)
daytrader-bot --live run     # full auto (PDT-exempt accounts only)

# Inspect and manage
daytrader-bot monitor        # show signal bus
daytrader-bot status         # show trade store
daytrader-bot close NVDA     # manual exit
```

## Config: `daytrader.json`

```json
{
  "gap_min_pct": 3.0,
  "gap_max_pct": 20.0,
  "entry_window_start_min": 575,
  "entry_window_end_min": 600,
  "exit_by_min": 660,
  "stop_pct": 2.0,
  "rr_min": 2.0,
  "budget": 200.0,
  "min_adv": 1000000.0,
  "min_price": 3.0,
  "scan_interval_sec": 60,
  "watchlist": ["SPY", "QQQ", "NVDA", "AMD", "AAPL", "TSLA", "META", "AMZN", "MSFT", "GOOGL"]
}
```

Time values are **minutes since midnight ET**:
- `575` = 9:35 AM ET
- `600` = 10:00 AM ET
- `660` = 11:00 AM ET

## Gap Detection

Uses Yahoo Finance v8 chart (`chartPreviousClose` vs `regularMarketPrice`). Run at market open
for most accurate gap readings. Pre-market prices are approximated by the current price at scan time.

## Signal Expiry

Day trade signals expire within 2 hours of creation — they're intraday only.
No overnight holds. The `run` command enforces a hard exit at `exit_by_min`.

## Daily Workflow

```bash
# 9:20 AM ET: pre-scan (identify gap candidates)
daytrader-bot scan

# 9:35 AM ET: start run loop (enters on pullback during 9:35-10:00 window)
daytrader-bot --semi run

# 11:00 AM ET: all positions auto-closed by bot
# (or close manually: daytrader-bot close <SYMBOL>)
```

## Example (from W18 2026)

```
RKLB +25.24% — gap too large (>20%), skip
NVDA +8.4%   — valid gap, wait for 9:35 entry window
AMD  +26.3%  — too extended, skip
TSLA +9.6%   — valid, wait for entry window
```

## Examples

```bash
# Pre-market scan at 9:20 AM ET — identify gap candidates
daytrader-bot scan

# Scan with earnings mode — tighter params, allows shorts on gap-downs
daytrader-bot --earnings scan

# Start the continuous loop in semi-auto (recommended for live trading)
daytrader-bot --semi run

# Start fully automatic — PDT-exempt accounts only
daytrader-bot --live run

# Check the signal bus for active signals
daytrader-bot monitor

# Check open and closed trades with P&L
daytrader-bot status

# Manually exit a position at market
daytrader-bot close NVDA

# Use a custom config file
daytrader-bot --config ~/my-daytrader.json scan
```

## Architecture

```
internal/
  strategy/   — types, scanner (gap detection), signal evaluation + entry window check
  broker/     — paper | semi | live
  store/      — daytrader-trades.json (includes gap_pct, exit_by time)
  signal/     — signal bus
cmd/main.go   — CLI with time-exit enforcement
```
