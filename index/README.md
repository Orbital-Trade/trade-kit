# index-trader

QQQ/VIX momentum bot for TQQQ/SQQQ day trades (Game 5 strategy). Polls QQQ and VIX every 30 seconds and enters leveraged ETF positions based on intraday momentum signals.

## Build

```bash
cd index && go build -o index-trader ./cmd/
```

## Configuration

`index/index.json` — all thresholds and position sizing:

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

| Field | Description |
|---|---|
| `qqq_long_threshold` | QQQ change% required to trigger TQQQ buy (e.g. 0.3 = +0.3%) |
| `qqq_short_threshold` | QQQ change% required to trigger SQQQ buy (e.g. -0.3 = -0.3%) |
| `vix_max` | Maximum VIX level for a long (TQQQ) signal |
| `vix_spike_min` | Minimum VIX level for a short (SQQQ) signal |
| `budget` | Max USD per trade |
| `tqqq_shares` | Number of TQQQ shares to buy on long signal |
| `sqqq_shares` | Number of SQQQ shares to buy on short signal |
| `stop_pct` | Stop-loss percentage below entry |
| `target_pct` | Take-profit percentage above entry |
| `grace_period_min` | Minutes after open to wait before taking entries (avoids open volatility) |
| `exit_by_min` | Minutes into session for forced EOD exit (750 = 12:30 PM ET) |
| `scan_interval_sec` | Poll interval in seconds |

**Signal logic:**
- QQQ > +0.3% and VIX < 25 → BUY TQQQ
- QQQ < -0.3% and VIX > 20 → BUY SQQQ

Executes via `tiger-cli --live` in the background.

## Commands

```
index-trader [--mode <mode>] [--config <path>]
```

**Flags:**

| Flag | Description |
|---|---|
| `--mode watch` | Print signals only, no orders (default) |
| `--mode semi` | Prompt for confirmation before each order (10-second timeout) |
| `--mode live` | Execute orders automatically via tiger-cli |
| `--config <path>` | Path to index.json (defaults to directory next to binary) |

The bot runs continuously until interrupted (Ctrl+C). It:
1. Waits for the grace period after open
2. Polls QQQ and VIX on each tick
3. Emits a signal when thresholds are met
4. Monitors the open position for stop/target/EOD exit

## Examples

```bash
# Watch mode — see signals without placing orders
index-trader --mode watch

# Semi-auto — confirm each trade before sending (recommended)
index-trader --mode semi

# Fully automatic — requires live tiger-cli credentials
index-trader --mode live

# Use a custom config file
index-trader --mode watch --config ~/my-index.json

# Run in the background with output logged
index-trader --mode semi >> ~/index-trader.log 2>&1 &
```
