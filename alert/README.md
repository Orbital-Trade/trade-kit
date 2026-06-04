# alert

Price threshold alert daemon. Watches a list of symbols via Yahoo Finance and fires a notification when price crosses a configured threshold. Integrates with the `notifier` binary for Telegram/Discord delivery.

## Build

```bash
cd alert && go build -o alert ./cmd/
```

## Configuration

`alert.json` in the working directory (or specify with `--config`):

```json
{
  "poll_interval_sec": 60,
  "alerts": [
    { "symbol": "LUNR",  "above": 40.00, "note": "breakout level" },
    { "symbol": "AAPL",  "below": 170.00, "note": "support breach" },
    { "symbol": "QQQ",   "above": 480.00 },
    { "symbol": "SQQQ",  "below": 8.00, "repeat": true }
  ]
}
```

| Field | Description |
|---|---|
| `poll_interval_sec` | How often to check prices in seconds (default: 60) |
| `alerts[].symbol` | Ticker symbol to watch (Yahoo Finance format) |
| `alerts[].above` | Fire when price >= this value |
| `alerts[].below` | Fire when price <= this value |
| `alerts[].note` | Optional text appended to the alert message |
| `alerts[].repeat` | If true, re-fires every poll cycle while price remains triggered. Default: one-shot only. |

Both `above` and `below` can be set on the same symbol to create a range alert.

## Commands

```
alert [--config <path>] <command>
```

| Command | Description |
|---|---|
| `daemon` | Poll continuously, fire on threshold cross. Runs forever until interrupted. |
| `check` | One-shot: check all alerts against current prices and exit. |
| `list` | Show configured alerts and fetch current live prices. |

**Flags:**

| Flag | Description |
|---|---|
| `--config <path>` | Path to config file (default: `alert.json`) |

## Examples

```bash
# Start the daemon in the background
alert daemon &

# Start with a custom config file and log output
alert daemon --config ~/my-alerts.json >> ~/alert.log 2>&1 &

# One-shot check — useful from cron
alert check

# See all configured alerts with current prices
alert list

# Use a non-default config for a second alert profile
alert daemon --config ~/vix-alerts.json &
```
