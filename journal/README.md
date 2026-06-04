# journal

Trade journal with SQLite storage. Records every fill manually or from broker exports, computes realized P&L using FIFO cost basis, and provides a filterable trade history. Duplicate order IDs are silently ignored on import.

## Build

```bash
cd journal && go build -o journal ./cmd/
```

## Configuration

No JSON config file. The database is stored at `~/.trade-kit/journal/trades.db` by default. Override with `--db <path>`.

## Commands

```
journal [--db <path>] <command> [args]
```

**Global flag:**

| Flag | Description |
|---|---|
| `--db <path>` | Override database path (default: `~/.trade-kit/journal/trades.db`) |

---

**`add` — record a trade:**

```bash
journal add <BUY|SELL> <SYMBOL> <QTY> <PRICE> [flags]
```

| Flag | Description |
|---|---|
| `--broker <name>` | Broker name: `tiger`, `moomoo`, or `manual` |
| `--order-id <id>` | Broker order ID — duplicate IDs are silently ignored |
| `--strategy <name>` | Strategy tag: `daytrader`, `bounce`, `earnings`, `manual` |
| `--note <text>` | Free-text annotation |
| `--date <YYYY-MM-DD>` | Fill date (default: today) |

---

**`list` — show trade history:**

```bash
journal list [--symbol <SYM>] [--days <N>]
```

| Flag | Description |
|---|---|
| `--symbol <SYM>` | Filter by ticker symbol |
| `--days <N>` | Show only trades from the last N days |

---

**`pnl` — realized P&L per symbol:**

```bash
journal pnl [--symbol <SYM>]
```

| Flag | Description |
|---|---|
| `--symbol <SYM>` | Show P&L for a specific symbol only |

P&L is computed on FIFO cost basis. Win/loss counts and total realized P&L are shown per symbol and as a grand total.

## Examples

```bash
# Record a buy fill
journal add BUY LUNR 10 35.50 --broker tiger --strategy daytrader

# Record the matching sell
journal add SELL LUNR 10 38.20 --broker tiger --note "target hit"

# Record a historical trade with a specific date
journal add BUY NVDA 2 900.00 --broker moomoo --date 2026-05-10

# List all trades
journal list

# List trades for a single symbol
journal list --symbol LUNR

# List trades from the last 30 days
journal list --days 30

# Show realized P&L summary for all symbols
journal pnl

# Show P&L for one symbol
journal pnl --symbol LUNR
```
