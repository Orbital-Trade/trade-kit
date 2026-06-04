# scheduler

Order execution queue daemon for trade-kit. Accepts orders at any time (even when the market is closed), queues them by market window, and fires them automatically when the session opens.

## Build

```bash
cd scheduler && go build -o scheduler ./cmd/
```

## Configuration

`scheduler/scheduler.json` controls the market windows and timezone. Runtime state is stored in `order-queue.json` (gitignored).

```json
{
  "windows": {
    "next_open":   "21:30",
    "pre_market":  "16:00",
    "now":         "immediate"
  },
  "timezone": "Asia/Singapore"
}
```

| Field | Description |
|---|---|
| `windows.next_open` | US regular session open: 9:30 AM ET / 21:30 SGT (default window) |
| `windows.pre_market` | US pre-market open: 4:00 AM ET / 16:00 SGT |
| `windows.now` | Execute on next daemon tick — market must be open |
| `timezone` | Display timezone for the time header (SGT) |

## Commands

```
scheduler <command> [args]
```

**Queue management:**

```bash
scheduler add [--at <window>] <type> <args>   # queue an order
scheduler list                                 # show all queued and executed orders
scheduler cancel <id>                          # remove a pending order from queue
scheduler clear                                # cancel all pending orders
```

**Daemon:**

```bash
scheduler daemon                               # start execution daemon (blocks)
scheduler daemon --log <path>                  # append execution log to file
scheduler daemon --allow-exec                  # enable TypeExec shell orders
```

**Order types for `add`:**

```bash
scheduler add buy    <SYMBOL> <QTY> [--limit <price>] [--note <text>]
scheduler add sell   <SYMBOL> <QTY> [--limit <price>] [--note <text>]
scheduler add stop   <SYMBOL> <QTY> --price <price>
scheduler add target <SYMBOL> <QTY> --price <price>
scheduler add modify <ORDER_ID> [--stop <p>] [--limit <p>] [--qty <n>] [--note <text>]
scheduler add cancel <ORDER_ID>
```

**Windows (`--at` flag):**

| Window | Time |
|---|---|
| `next_open` | 9:30 AM ET / 21:30 SGT (default) |
| `pre_market` | 4:00 AM ET / 16:00 SGT |
| `now` | Next daemon tick |

**Daemon flags:**

| Flag | Description |
|---|---|
| `--log <path>` | Append execution log to file |
| `--allow-exec` | Enable shell command orders (off by default for security) |

**Position guard:** Sell orders check Tiger positions before executing to prevent accidental shorts. If no position is found the sell is skipped.

**Daily exec orders:** `TypeExec` orders with `--daily` flag are automatically re-queued for the next occurrence after execution.

## Examples

```bash
# Queue a market sell at next open (tonight's order, placed now)
scheduler add sell QUBT 15

# Queue a limit buy at next open
scheduler add buy SCHD 4 --limit 31.65

# Queue a stop-loss update at next open
scheduler add modify 43116645746347008 --stop 23.50

# Queue a limit sell at pre-market open
scheduler add --at pre_market sell QUBT 15

# Review the queue
scheduler list

# Remove a pending order
scheduler cancel a3f2b1

# Start the daemon in the background with a log file
scheduler daemon --log ~/scheduler.log &

# Start the daemon with shell command support
scheduler daemon --allow-exec --log ~/scheduler.log &
```
