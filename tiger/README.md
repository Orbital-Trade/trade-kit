# tiger-cli

A standalone Go CLI for executing Tiger Brokers trades from the command line.

Each operation is an isolated function in `ops/` — the CLI is one consumer; an MCP server would be another. No refactoring needed when you add the MCP layer.

## Prerequisites

- Go 1.21+
- Tiger Brokers account with API access (TBSG license)
- Credentials at `brokers/Tiger/.env` and `brokers/Tiger/tiger_openapi_config.properties`

## Build

```bash
cd tools/tiger
go build -o tiger-cli ./cmd/
```

## Usage

```
tiger-cli [--paper|--live] [--json] <command> [args]
```

**Modes:**
- Default (no flag) — paper mode. Logs intent; never sends a real order.
- `--live` — live trading. Prints the order and asks `y/N` before sending.
- `--json` — output as JSON (clean stdout, no formatting). Used for scripting and MCP piping.

**Log level** (stderr only, stdout stays clean):
```bash
TIGER_LOG_LEVEL=debug tiger-cli positions   # debug|info|warn|error|off
```

---

## Commands

### Read (no confirmation, no side effects)

```bash
# All open stock positions with P&L
tiger-cli positions

# Account: cash, buying power, net liquidation
tiger-cli account

# Real-time price snapshot (today's OHLCV + change %)
tiger-cli quote NOK

# All open/pending orders
tiger-cli orders
```

### Stock orders

```bash
# Market buy
tiger-cli buy NOK 100

# Limit buy
tiger-cli buy NOK 100 --limit 4.50

# Limit buy + stop-loss + take-profit in one command
tiger-cli buy NOK 100 --limit 4.50 --stop 4.20 --target 5.00

# Market sell
tiger-cli sell NOK 50

# Limit sell (GTC — stays open until filled or cancelled)
tiger-cli sell NOK 50 --limit 4.80

# Protective stop-loss on an existing position (GTC, stop-market)
tiger-cli stop NOK 100 --price 4.20

# Take-profit limit on an existing position (GTC, limit sell)
tiger-cli target NOK 100 --price 5.00

# Cancel an open order
tiger-cli cancel 123456789

# Modify an open order in-place (no cancel + replace)
tiger-cli modify 123456789 --limit 4.60          # change limit price
tiger-cli modify 123456789 --stop 4.10           # move stop trigger
tiger-cli modify 123456789 --qty 50              # change quantity
tiger-cli modify 123456789 --qty 50 --tif GTC   # change multiple fields
```

### Futures

```bash
# Entry + protective stop pair (limit entry, stop-market stop)
tiger-cli futures entry MES long 1 --entry 5100 --stop 5090

# Market close
tiger-cli futures close MES long 1

# Move trailing stop (cancel old, place new)
tiger-cli futures update-stop MES long 1 --stop 5095 --old-id 789012345
```

### Live mode examples

```bash
# Asks "Execute live order? [y/N]" before sending
tiger-cli --live buy NOK 100 --limit 4.50 --stop 4.20

# JSON output for scripting
tiger-cli --live --json sell NOK 50 --limit 4.80
```

---

## Order type reference

| Command      | Tiger order_type | time_in_force | Notes |
|---|---|---|---|
| `buy` (default) | MKT | DAY | Market fill, no price control |
| `buy --limit`   | LMT | DAY | Preferred for entries |
| `sell` (default)| MKT | DAY | Urgent exits |
| `sell --limit`  | LMT | GTC | Take-profit exits |
| `stop`          | STP | GTC | Stop-market — guaranteed fill on trigger |
| `target`        | LMT | GTC | Alias for `sell --limit` |
| `futures entry` | LMT + STP | DAY + GTC | Entry + stop pair, atomic |
| `futures close` | MKT | DAY | Fastest exit |
| `futures update-stop` | cancel + STP | GTC | Trailing stop management |
| `modify`              | —            | —   | In-place modify; fetches order state, sends only changed fields |

**STP vs STP_LMT:** All stops use `STP` (stop-market), never `STP_LMT` (stop-limit). A stop-limit can miss its fill in a fast-moving market and leave the position unprotected. Stop-market guarantees the exit at the cost of price slippage.

---

## JSON output

Every command supports `--json` for clean, machine-readable output:

```bash
$ tiger-cli --json positions
[
  {
    "symbol": "NOK",
    "shares": 100,
    "avg_cost": 4.5,
    "market_price": 4.8,
    "market_value": 480,
    "unrealized_pnl": 30,
    "realized_pnl": 0
  }
]

$ tiger-cli --json quote NOK
{
  "symbol": "NOK",
  "price": 4.80,
  "open": 4.50,
  "high": 4.90,
  "low": 4.48,
  "volume": 2000000,
  "prev_close": 4.50,
  "change_pct": 6.667
}
```

---

## Project structure

```
tools/tiger/
├── go.mod                        # standalone module: tiger-cli
├── client/
│   └── tiger.go                  # Tiger REST transport: config loading, RSA signing, apiCall
├── ops/                          # one file = one operation = one future MCP tool
│   ├── types.go                  # Caller interface + shared OrderResult type
│   ├── positions.go              # GetPositions(c) → []Position
│   ├── account.go                # GetAccount(c) → Account
│   ├── quote.go                  # GetQuote(c, symbol) → Quote
│   ├── orders.go                 # GetOrders(c) → []Order
│   ├── cancel.go                 # CancelOrder(c, id) → CancelResult
│   ├── modify.go                 # ModifyOrder(c, id, params) → ModifyResult
│   ├── buy.go                    # BuyMarket(c, sym, n), BuyLimit(c, sym, n, price) → OrderResult
│   ├── sell.go                   # SellMarket(c, sym, n), SellLimit(c, sym, n, price) → OrderResult
│   ├── stop.go                   # SetStopLoss(c, sym, n, price) → OrderResult
│   ├── take_profit.go            # SetTakeProfit(c, sym, n, price) → OrderResult
│   └── futures.go                # FuturesEntry, FuturesClose, FuturesUpdateStop
├── internal/
│   └── tlog/
│       └── tlog.go               # leveled logger to stderr (debug/info/warn/error)
└── cmd/
    └── main.go                   # thin CLI dispatcher (~200 lines)
```

---

## Running tests

```bash
cd tools/tiger
go test ./...
```

Tests use a mock `Caller` implementation — no Tiger credentials needed. The mock queues per-method responses and errors, allowing full coverage of edge cases (API errors, stop-failure emergency close, zero-ID rejection, bar sorting).

```
ok  tiger-cli/client   (fallbackContract, parseKey, loadConfig)
ok  tiger-cli/ops      (all 68 tests: positions, account, quote, orders,
                        cancel, modify, buy, sell, stop, futures)
```

---

## Adding a new operation

1. Create `ops/my_op.go` with a function `MyOp(c Caller, ...) (MyResult, error)`.
2. Add a test in `ops/my_op_test.go` using `newMock(paper)`.
3. Add a case in `cmd/main.go` `switch cmd`.
4. When adding as an MCP tool: create a handler that calls `ops.MyOp(c, ...)` and returns the result as JSON.

The `Caller` interface is the only contract between the ops layer and any transport (CLI, MCP, HTTP handler). `*client.TigerClient` satisfies it in production; `*mockCaller` satisfies it in tests.

---

## Configuration

Credentials are loaded automatically from the project's existing Tiger config files:

```
brokers/Tiger/.env
    TIGER_ID=<your tiger ID>
    PRIVATE_KEY=<base64-encoded PKCS8 RSA key>
    TRADE_PASSWORD=<6-digit PIN>

brokers/Tiger/tiger_openapi_config.properties
    tiger_id=<your tiger ID>
    account=<your account number>
    private_key_pk8=<base64-encoded PKCS8 RSA key>
    license=TBSG
```

The client searches for credentials in `./brokers/Tiger/`, `../../brokers/Tiger/`, or `$HOME/.trade-kit/tiger/`.

## Examples

```bash
# Check account and open positions
tiger-cli account
tiger-cli positions

# Get a real-time price quote
tiger-cli quote AAPL

# Limit buy with stop-loss and take-profit in one command (paper mode)
tiger-cli buy NOK 100 --limit 4.50 --stop 4.20 --target 5.00

# Live limit buy — prints order details and prompts y/N before sending
tiger-cli --live buy NOK 100 --limit 4.50 --stop 4.20

# Live market sell
tiger-cli --live sell NOK 100

# Cancel an open order
tiger-cli --live cancel 123456789

# Modify a stop price in-place (no cancel + replace)
tiger-cli --live modify 123456789 --stop 4.10

# Futures: enter a long MES position with protective stop
tiger-cli --live futures entry MES long 1 --entry 5100 --stop 5090

# Multi-timeframe technical analysis
tiger-cli analyze AAPL
tiger-cli --json analyze AAPL

# Markov regime model — state probabilities and directional signal
tiger-cli markov NVDA
tiger-cli --json markov NVDA
```
