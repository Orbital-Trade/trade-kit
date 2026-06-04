# options

Options chain viewer for US stocks. Displays calls and puts with strike, bid/ask, IV, open interest, and volume. Data sourced from Yahoo Finance — no API key or broker connection required.

## Build

```bash
cd options && go build -o options ./cmd/
```

## Configuration

No configuration file. All options are passed as command-line flags.

## Commands

```
options <command> <SYMBOL> [flags]
```

**`chain` — display the options chain:**

```bash
options chain <SYMBOL> [flags]
```

| Flag | Description |
|---|---|
| `--expiry <YYYY-MM-DD>` | Specific expiry date. Defaults to nearest available expiry. |
| `--calls` | Show calls only |
| `--puts` | Show puts only |
| `--json` | Machine-readable JSON output |

**`expiries` — list available expiry dates:**

```bash
options expiries <SYMBOL>
```

Shows all available expiry dates and days-to-expiry (DTE) for the given symbol.

**Display markers:**

| Marker | Meaning |
|---|---|
| `>` | Near-the-money contract (within 2% of underlying price) |
| `*` | In-the-money contract |

**Columns:** STRIKE, BID, ASK, LAST, IV%, OI (open interest), VOLUME, ITM

## Examples

```bash
# Show nearest-expiry chain for AAPL (calls + puts)
options chain AAPL

# Show calls only for AAPL
options chain AAPL --calls

# Show puts only for a specific expiry
options chain AAPL --expiry 2026-06-20 --puts

# Full chain for a specific expiry in JSON format
options chain AAPL --expiry 2026-06-20 --json

# List all available expiry dates for NVDA
options expiries NVDA

# Check TSLA puts at a specific expiry
options chain TSLA --expiry 2026-07-18 --puts
```
