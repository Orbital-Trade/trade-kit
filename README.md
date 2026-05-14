# trade-kit

**The open-source CLI trading toolkit for Tiger Brokers and Moomoo.**

Zero dependencies. Paper mode by default. Go binaries — download and run.

Built for retail traders in Singapore, Hong Kong, Australia, and the US who use Tiger or Moomoo and want to automate their workflow without writing code.

> ⚠️ See [DISCLAIMER.md](DISCLAIMER.md) — this is not financial advice.

---

## Tools

| Tool | Binary | What it does |
|---|---|---|
| [tiger](tiger/) | `tiger-cli` | Tiger Brokers CLI — positions, quotes, buy/sell/stop/modify |
| [moomoo](moomoo/) | `moomoo-cli` | Moomoo CLI — same interface, pure Go (no Python) |
| [scheduler](scheduler/) | `scheduler` | Order queue daemon — schedule orders at market windows |
| [daytrader](daytrader/) | `daytrader-bot` | Gap-up scanner — finds gap plays at pre-market open |
| [earnings](earnings/) | `earnings-bot` | Earnings play scanner — RVOL, gap direction, long/short |
| [bounce](bounce/) | `bounce-bot` | RSI bounce scanner — oversold + volume confirmation |
| [index](index/) | `index-trader` | Index momentum — QQQ/VIX signals for TQQQ/SQQQ |
| [controller](controller/) | `controller` | Portfolio risk manager — circuit breaker, NAV tracking |

---

## Quick start

```bash
# Download latest release binaries (macOS/Linux/Windows)
# https://github.com/jpramirez/trade-kit/releases

# Or build from source (requires Go 1.21+)
cd tiger && go build -o tiger-cli ./cmd/
cd moomoo && go build -o moomoo-cli ./cmd/
cd scheduler && go build -o scheduler ./cmd/
# ... repeat for each tool
```

### Tiger setup

1. Apply for Tiger Open API access at [openapi.tigersecurities.com](https://openapi.tigersecurities.com)
2. Create `~/.trade-kit/tiger/.env`:

```env
TIGER_ID=<your tiger ID>
PRIVATE_KEY=<base64-encoded PKCS8 RSA private key>
TRADE_PASSWORD=<your 6-digit trade PIN>
```

3. Test connection:

```bash
./tiger-cli positions          # paper mode (safe)
./tiger-cli --live positions   # live account
```

### Moomoo setup

1. Download and start [Futu OpenD](https://www.futunn.com/download/OpenD)
2. Create `~/.trade-kit/moomoo/.env`:

```env
MOOMOO_HOST=127.0.0.1
MOOMOO_PORT=11111
TRADE_PASSWORD=<your 6-digit PIN>
ACC_ID=<your account ID>
```

3. Test:

```bash
./moomoo-cli positions
```

---

## Paper mode vs live mode

Every write command (buy, sell, stop) runs in **paper mode by default** — nothing is sent to the broker. Add `--live` to execute real orders (with confirmation prompt).

```bash
./tiger-cli buy AAPL 10             # paper — prints what would happen
./tiger-cli --live buy AAPL 10      # live — prompts "Execute? [y/N]"
```

---

## Strategies

Each scanner bot has three modes:

```bash
./daytrader-bot scan    # print signals, no action
./daytrader-bot semi    # print signals, prompt before each trade
./daytrader-bot run     # live — auto-executes via tiger-cli
```

Configure via each tool's JSON file (no code changes needed):

```json
// daytrader/daytrader.json
{
  "gap_min_pct": 3.0,
  "gap_max_pct": 20.0,
  "stop_pct": 2.0,
  "rr_min": 3.0,
  "budget": 200.0,
  "watchlist": ["LUNR", "RKLB", "ASTS", "IONQ", "QQQ"]
}
```

---

## Scheduler

Automate orders and bot runs at market windows:

```bash
./scheduler add buy AAPL 10 --window next_open       # buy at next market open
./scheduler add exec "./daytrader-bot scan" --window pre_open --daily
./scheduler list
./scheduler daemon &   # runs in background, fires orders at the right time
```

---

## Versioning

This project uses [Semantic Versioning](https://semver.org/). See [CHANGELOG.md](CHANGELOG.md) for release notes.

Current version: **v0.1.0**

---

## Broker sign-up

If trade-kit is useful, consider signing up through these referral links — helps keep the project maintained:

- **Tiger Brokers** → [Sign up](https://www.itiger.com/sg/invite) (up to S$1,000 welcome rewards)
- **Moomoo** → [Sign up](https://www.moomoo.com/sg) (free stocks for new accounts)

---

## Roadmap

- [ ] `notifier` — Telegram/Discord signal delivery
- [ ] `alert` — price alert daemon
- [ ] `journal` — trade journal + P&L stats
- [ ] `backtest` — historical strategy validation
- [ ] `options` — options chain viewer
- [ ] GoReleaser CI for pre-built binaries

---

## License

MIT — see [LICENSE](LICENSE).

Not financial advice — see [DISCLAIMER.md](DISCLAIMER.md).
