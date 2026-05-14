# bounce-bot — Game 2: RSI Oversold Bounce

When a fundamentally sound stock hits extreme oversold (RSI ≤ 20), it almost always snaps back
toward the mean (RSI 50). Buy the panic, ride the recovery, exit fast.

## Strategy

| Parameter | Value |
|---|---|
| Entry trigger | RSI (14-period daily) ≤ 20 |
| Exit trigger | RSI recovers to 50, OR 5 trading days pass |
| Stop | 5% below entry |
| Budget | $150 max per trade |
| Volume filter | Avg daily volume ≥ 500,000 |
| Price filter | ≥ $3.00 |
| Max hold | 5 trading days (forced exit) |

**Critical rule:** Always check news before acting on an RSI signal. RSI 16 on a fraud story
is a short, not a bounce. The bot does not check news — you must verify manually.

## Build

```bash
cd tools/bounce
go build -o bounce-bot ./cmd/
```

## Commands

```bash
# Evaluate watchlist → write pending signals
bounce-bot scan

# Continuous loop
bounce-bot run
bounce-bot --semi run     # confirm each trade
bounce-bot --live run     # fully automatic

# Inspect and manage
bounce-bot monitor        # show signal bus
bounce-bot status         # show trade store
bounce-bot close MCK      # manual exit
```

## Config: `bounce.json`

```json
{
  "rsi_entry": 20.0,
  "rsi_exit": 50.0,
  "stop_pct": 5.0,
  "max_hold_days": 5,
  "budget": 150.0,
  "min_adv": 500000.0,
  "min_price": 3.0,
  "scan_interval_sec": 300,
  "watchlist": ["MCK", "SPY", "QQQ", "AAPL", "MSFT", "AMZN", "GOOGL",
                "META", "NVDA", "AMD", "JPM", "BAC", "XOM", "CVX"]
}
```

## RSI Calculation

14-period standard RSI from daily closing prices. Fetched from Yahoo Finance v8 chart
(no auth required). 20 days of daily bars fetched to ensure accurate RSI.

```
RSI = 100 - (100 / (1 + RS))
RS  = avg gain (14 days) / avg loss (14 days)
```

RSI < 20 = extreme oversold (historically ≥65% bounce rate on large caps with no fundamental damage)
RSI > 80 = overbought (not used by this bot, but tracked for context)

## Signal Bus

Writes `"strategy": "bounce"` signals. Only executes signals matching this tag.

```bash
bounce-bot --signals ~/signals.json scan
bounce-bot --signals ~/signals.json run
```

## Live Candidate (W18 2026)

| Symbol | RSI | Note |
|---|---|---|
| MCK | 14.4 | Extreme oversold. No fundamental damage. $736/sh — raise budget or check options. |

## Trade Persistence

Stored in `bounce-trades.json`. Includes entry RSI, target RSI, and expiry date.

## Architecture

```
internal/
  strategy/   — types, scanner (Yahoo Finance + RSI), signal evaluation
  broker/     — paper | semi | live
  store/      — bounce-trades.json (includes entry_rsi, target_rsi, expires_at)
  signal/     — signal bus
cmd/main.go   — CLI dispatcher
```
