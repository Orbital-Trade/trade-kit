# orbital-ctrl — Master Controller

Unified dashboard across all running strategy bots. Shows account, positions, open orders,
and the full signal bus in one view. Also handles emergency stop.

## Build

```bash
cd tools/controller
go build -o orbital-ctrl ./cmd/
```

## Commands

```bash
# Unified dashboard: account + positions + orders + signal summary
orbital-ctrl status

# Full signal bus detail across all strategies
orbital-ctrl monitor

# EMERGENCY: cancel all orders + market-sell all positions
orbital-ctrl estop
```

## Signal Bus Integration

Point all bots at the same `signals.json` for a unified view:

```bash
SIGNALS=/path/to/shared/signals.json

earnings-bot  --signals $SIGNALS run &
bounce-bot    --signals $SIGNALS run &
daytrader-bot --signals $SIGNALS run &
orbital-ctrl  --signals $SIGNALS status
```

Default path is `./signals.json` in the working directory.

## Status Output Example

```
═══ ORBITAL CTRL — 2026-05-11 09:30 SGT ═══

  Net Liq:      $772.98
  Cash:         $154.54
  Buying Power: $1511.23

  POSITIONS (4)
  SYMBOL    QTY   AVG COST    MKT PRC  UNREAL P&L    REAL P&L
  ──────────────────────────────────────────────────────────────
  EXTR        9      21.98      24.06  +    18.74       0.00
  QUBT       15       9.60       9.61  +     0.22       0.00
  SCHD        8      31.78      31.63       -1.23       0.00
  ──────────────────────────────────────────────────────────────
  TOTAL                                    +17.73

  OPEN ORDERS (2)
  43134450120921088   QUBT  SELL  STP  qty 15  stop $8.20
  43116645746347008   EXTR  SELL  STP  qty 9   stop $23.20

  SIGNAL BUS (1 total)
  earnings      1 pending  0 active  0 closed

  PENDING SIGNALS (1)
  QUBT    earnings    20 sh @ $9.60  stop $9.12  [expires in 34h]
         1 day to earnings May 11 2026
```

## Emergency Stop

The `estop` command:
1. Fetches all open orders → cancels each one
2. Fetches all positions → market-sells each one
3. Requires typing `CONFIRM` — cannot be triggered accidentally

```bash
orbital-ctrl --signals ~/signals.json estop
# ⚠️  EMERGENCY STOP — this will cancel ALL orders and close ALL positions.
# Type 'CONFIRM' to proceed: CONFIRM
```

## Architecture

```
internal/signal/  — signal bus (read-only for status/monitor)
cmd/main.go       — status | monitor | estop
```

Imports `tiger-cli/ops` directly for positions, orders, account, cancel, sell.
No broker abstraction needed — the controller always acts in live mode.
