# earnings-bot — Game 1: Earnings Momentum Run

Buy stocks N days before their earnings report. Sell day-of, before the report drops.
Captures the pre-earnings run-up. Avoids the binary event entirely.

## Strategy

| Parameter | Value |
|---|---|
| Entry window | 1–3 calendar days before earnings |
| Stop | 5% below entry |
| Exit | Market sell at 15:45 ET on earnings day |
| Budget | $200 max per trade |
| Volume filter | Avg daily volume ≥ 500,000 |
| Price filter | ≥ $3.00 |
| Chase filter | Skip if stock already up >20% in last 5 days |

## Build

```bash
cd tools/earnings
go build -o earnings-bot ./cmd/
```

## Commands

```bash
# Evaluate watchlist → write pending signals to signals.json
earnings-bot scan

# Continuous loop: scan every 5 min + execute fresh signals every 30s
earnings-bot run
earnings-bot --semi run    # confirm each trade
earnings-bot --live run    # fully automatic

# Inspect and manage
earnings-bot monitor       # show signal bus state
earnings-bot status        # show filled trade store
earnings-bot close NVDA    # manual exit at market
```

## Modes

```
--paper   Log only (default) — safe for testing
--semi    Confirm each order before sending
--live    Execute automatically
```

## Config: `earnings.json`

```json
{
  "days_before": 3,
  "stop_pct": 5.0,
  "target_pct": 0.0,
  "budget": 200.0,
  "min_adv": 500000.0,
  "min_price": 3.0,
  "max_run_pct": 20.0,
  "watchlist": ["NVDA", "AAPL", "MSFT", "AMD", "META", "AMZN", "QUBT", "EXTR"],
  "scan_interval_sec": 300,
  "earnings_dates": {
    "NVDA": "2026-05-20",
    "QUBT": "2026-05-11"
  }
}
```

## Adding a New Earnings Play

1. Add symbol to `watchlist`
2. Add date to `earnings_dates` (format: `"YYYY-MM-DD"`)
3. Run `./earnings-bot scan` — signal appears in `signals.json`

Earnings dates can also be left out of config. The bot attempts to fetch them from Yahoo Finance
using a crumb session. For reliability, always add dates manually.

## Signal Bus

`scan` writes signals with `"strategy": "earnings"`. `run` only executes signals matching this tag.
All bots share the same `signals.json` — use `--signals path` to point to a shared location.

```bash
# Shared bus example
earnings-bot --signals ~/signals.json scan
earnings-bot --signals ~/signals.json run
```

## Trade Persistence

Filled positions are stored in `earnings-trades.json` in the working directory.

## Upcoming Catalysts (2026)

| Symbol | Earnings Date | Entry Window |
|---|---|---|
| QUBT | May 11, 2026 | May 8–10 |
| NVDA | May 20, 2026 | May 17–19 |

## Architecture

```
internal/
  strategy/   — types, scanner (Yahoo Finance), signal evaluation
  broker/     — paper | semi | live (imports tiger-cli ops)
  store/      — JSON trade persistence (earnings-trades.json)
  signal/     — signal bus (signals.json)
cmd/main.go   — CLI dispatcher
```
