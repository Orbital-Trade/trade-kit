# moomoo-cli

A standalone Go CLI for executing Moomoo/Futu trades from the command line. Pure Go TCP client — no Python bridge required.

## Build

```bash
cd moomoo && go build -o moomoo-cli ./cmd/
```

Requires Futu OpenD running locally before use:

```bash
futu-opend &
```

## Configuration

Credentials are read from `~/.trade-kit/moomoo/.env` or `../../brokers/Moomoo/.env`:

```
MOOMOO_HOST=127.0.0.1
MOOMOO_PORT=11111
TRADE_PASSWORD=<6-digit PIN>
ACC_ID=<account ID>
```

No JSON config file — all settings are in the `.env` file.

## Commands

```
moomoo-cli [--paper|--live] [--json] <command> [args]
```

**Global flags:**

| Flag | Description |
|---|---|
| `--paper` | Paper/simulation mode (default — no real orders sent) |
| `--live` | Live trading mode (requires Y confirmation for write ops) |
| `--json` | Output as JSON for scripting |

**Read commands (no confirmation needed):**

```bash
moomoo-cli positions                        # list open positions
moomoo-cli account                          # net assets, cash, buying power
moomoo-cli quote <SYMBOL>                   # real-time price via Yahoo Finance
moomoo-cli orders                           # list open/pending orders
```

**Write commands:**

```bash
moomoo-cli buy   <SYMBOL> <SHARES>          # market buy
moomoo-cli buy   <SYMBOL> <SHARES> --limit <price>    # limit buy (DAY)
moomoo-cli buy   <SYMBOL> <SHARES> --stop  <price>    # also place stop-loss after buy
moomoo-cli buy   <SYMBOL> <SHARES> --target <price>   # also place take-profit after buy

moomoo-cli sell  <SYMBOL> <SHARES>          # market sell
moomoo-cli sell  <SYMBOL> <SHARES> --limit <price>    # limit sell (GTC)

moomoo-cli stop  <SYMBOL> <SHARES> --price <price>    # set stop-loss (GTC, stop-market)
moomoo-cli target <SYMBOL> <SHARES> --price <price>   # set take-profit (GTC, limit)

moomoo-cli cancel <ORDER_ID>                # cancel an open order
moomoo-cli modify <ORDER_ID> --limit <price>          # change limit price
moomoo-cli modify <ORDER_ID> --stop  <price>          # change aux/stop price
moomoo-cli modify <ORDER_ID> --qty   <n>              # change quantity
```

**Symbol format (auto-detected):**

| Input | Resolved |
|---|---|
| `AAPL` | `US.AAPL` |
| `Z74`, `G13` | `SG.Z74` (SGX alpha+digit pattern) |
| `558`, `558.SI` | `SG.558` |
| `SG.558` | `SG.558` (pass-through) |
| `HK.00700` | `HK.00700` (pass-through) |

## Examples

```bash
# Check account and positions
moomoo-cli account
moomoo-cli positions

# Get a live price quote
moomoo-cli quote Z74
moomoo-cli quote AAPL

# Buy 100 shares of Z74 at limit $4.50 with a stop at $4.20 (live)
moomoo-cli --live buy Z74 100 --limit 4.50 --stop 4.20

# Sell 100 shares at market (live)
moomoo-cli --live sell Z74 100

# Place a GTC stop-loss on an existing position (live)
moomoo-cli --live stop Z74 100 --price 4.20

# Place a take-profit limit order (live)
moomoo-cli --live target Z74 100 --price 5.00

# Cancel an open order (live)
moomoo-cli --live cancel FS1C71DE6D05BA1000

# Modify an order's limit price (live)
moomoo-cli --live modify FS1C71DE6D05BA1000 --limit 4.60

# Output positions as JSON for scripting
moomoo-cli --json positions
```
