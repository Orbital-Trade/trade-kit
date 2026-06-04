# notifier

Trade signal delivery for trade-kit. Sends signals to Telegram and/or Discord. All scanner bots call `notifier send` to push signals to the user's phone. Falls back to stdout if no channels are configured — no crash, no required setup.

## Build

```bash
cd notifier && go build -o notifier ./cmd/
```

## Configuration

`notifier.json` in the working directory, or `~/.trade-kit/notifier/notifier.json`:

```json
{
  "telegram_bot_token": "",
  "telegram_chat_id": "",
  "discord_webhook_url": "",
  "enabled": true
}
```

| Field | Description |
|---|---|
| `telegram_bot_token` | Bot token from BotFather. Leave blank for stdout-only mode. |
| `telegram_chat_id` | Channel or chat ID (`@channelusername` or numeric). |
| `discord_webhook_url` | Discord channel webhook URL. Leave blank to disable Discord. |
| `enabled` | Set to `false` to silence all delivery (stdout still prints). |

**Free tier:** leave `telegram_bot_token` and `discord_webhook_url` blank. Signals go to stdout only.

**Paid tier:** configure a private Telegram channel token. Run bots on a VPS, push signals to subscribers.

## Commands

```
notifier <command> [args]
```

**Send a free-text message:**

```bash
notifier send "<text>"
```

**Send a structured signal:**

```bash
notifier send --symbol <SYM> --signal <BUY|SELL|STOP|ALERT> \
  [--price <p>] [--stop <p>] [--target <p>] \
  [--qty <n>] [--strategy <name>] [--note <text>]
```

**Other commands:**

```bash
notifier test       # send a test message to all configured channels
notifier status     # show which channels are configured
```

**`send` flags:**

| Flag | Description |
|---|---|
| `--symbol` | Ticker symbol (uppercased automatically) |
| `--signal` | Signal direction: `BUY`, `SELL`, `STOP`, `ALERT`, `LONG`, `SHORT`, `EXIT` |
| `--price` | Entry price |
| `--stop` | Stop-loss price (risk % computed automatically) |
| `--target` | Take-profit price (R:R computed automatically) |
| `--qty` | Share quantity |
| `--strategy` | Strategy name for context (e.g. `daytrader`, `bounce`) |
| `--note` | Free-text annotation appended to the message |

## Examples

```bash
# Send a plain text alert
notifier send "LUNR gap +8.3% — BUY signal"

# Send a structured BUY signal with stop and target
notifier send --symbol LUNR --signal BUY --price 35.50 --stop 32.00 --target 42.00 --qty 5

# Send a SELL signal from the earnings strategy
notifier send --symbol NVDA --signal SELL --price 920.00 --strategy earnings --note "earnings day exit"

# Verify all channels are working
notifier test

# Check which channels are configured
notifier status
```
