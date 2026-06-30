# CLAUDE.md — trade-kit

## What this project is

trade-kit is an **open-source CLI trading toolkit** for retail traders using Tiger Brokers and Moomoo. It is published at https://github.com/jpramirez/trade-kit under MIT license.

**Owner:** Juan Pablo Ramirez (`jpramirez`), Epyphite Pte Ltd, Singapore  
**Email:** jramirez@epyphite.com  
**Current version:** v0.1.0 (tagged May 15, 2026)  
**Working directory:** `/home/jramirez/development/trade-kit/`

This is a standalone repo — separate from the main OrbitalTrade platform. The tools here are generic and broker-agnostic by design.

---

## Repository layout

```
trade-kit/
├── CLAUDE.md           ← this file
├── README.md           ← user-facing pitch + broker affiliate links
├── CHANGELOG.md        ← semantic version history
├── DISCLAIMER.md       ← not financial advice
├── LICENSE             ← MIT
├── VERSION             ← current version string
├── Makefile            ← build all tools (TODO: create)
├── tiger/              ← tiger-cli binary
├── moomoo/             ← moomoo-cli binary
├── scheduler/          ← scheduler binary
├── daytrader/          ← daytrader-bot binary
├── earnings/           ← earnings-bot binary
├── bounce/             ← bounce-bot binary
├── controller/         ← controller binary
├── index/              ← index-trader binary
└── notifier/           ← NEXT: notifier binary (Telegram/Discord)
```

Each tool is **fully independent**: own `go.mod`, own `go.sum`, own JSON config, own binary. No shared runtime dependencies between tools.

---

## Tools — current state

### tiger-cli (`tiger/`)
Tiger Brokers REST API wrapper in Go. Full test coverage.

**Commands:** `positions`, `account`, `quote <SYM>`, `orders`, `buy`, `sell`, `stop`, `target`, `cancel`, `modify`

**Flags:** `--live` (real orders), `--paper` (default), `--json`

**Auth:** RSA key signing. Reads credentials from:
1. `./brokers/Tiger/.env`
2. `../../brokers/Tiger/.env`
3. `~/.trade-kit/tiger/.env`

**Known limitation:** SGX stocks (Singapore Exchange) are rejected by the Tiger REST API for TBSG accounts — US stocks only via CLI.

**Build:** `cd tiger && go build -o tiger-cli ./cmd/`  
**Test:** `cd tiger && go test ./...` — all pass

---

### moomoo-cli (`moomoo/`)
Moomoo/Futu API wrapper — **pure Go TCP client** (no Python bridge).

**Commands:** Same interface as tiger-cli.

**Protocol:** Futu OpenD TCP on localhost:11111. 44-byte header + JSON body (body_type=1, avoids protobuf). MD5-hashed trade PIN for unlock.

**Key proto IDs:** InitConnect=1001, GetAccList=2001, UnlockTrade=2005, GetPositionList=2102, GetAccInfo=2101, GetOrderList=2111, PlaceOrder=2202, ModifyOrder=2205

**Auth:** Reads from `~/.trade-kit/moomoo/.env` or `../../brokers/Moomoo/.env`:
```
MOOMOO_HOST=127.0.0.1
MOOMOO_PORT=11111
TRADE_PASSWORD=<6-digit PIN>
ACC_ID=<account ID>
```

**Requires:** Futu OpenD running (`futu-opend &`)

**Build:** `cd moomoo && go build -o moomoo-cli ./cmd/`

---

### scheduler (`scheduler/`)
Order queue daemon. Fires orders at market windows.

**Windows:** `pre_open` (21:20 SGT), `next_open` (21:30 SGT), `morning` (07:00 SGT)

**Order types:** `TypeBuy`, `TypeSell`, `TypeModify`, `TypeExec` (shell commands)

**Commands:** `scheduler add buy AAPL 10 --window next_open`, `scheduler list`, `scheduler cancel <ID>`, `scheduler daemon`

**Key feature:** `--daily` flag on TypeExec re-queues the command automatically for next day.

**Position guard:** Sell orders check Tiger positions before executing to prevent accidental shorts.

**Config:** `scheduler/scheduler.json` (windows, timezone). Runtime state in `order-queue.json` (gitignored — contains personal paths).

**Build:** `cd scheduler && go build -o scheduler ./cmd/`

---

### daytrader-bot (`daytrader/`)
Gap-up scanner. Runs at pre-market open (21:20 SGT).

**Logic:** For each symbol in watchlist → fetch price + prev close + ADV via Yahoo Finance → calculate gap% + RVOL → if gap in [3%, 20%] and RVOL > 1.5 → BUY signal.

**Modes:** `scan` (print signals), `semi` (prompt), `run` (auto-execute via tiger-cli)

**Flags:** `--earnings` — loads today's earners from earnings.json, enables short side on gap-down

**Config:** `daytrader/daytrader.json`
```json
{
  "gap_min_pct": 3.0, "gap_max_pct": 20.0,
  "stop_pct": 2.0, "rr_min": 3.0,
  "budget": 200.0, "min_adv": 500000,
  "scan_interval_sec": 60,
  "watchlist": ["LUNR", "RKLB", "ASTS", "KTOS", "IONQ", "RDW", "QQQ"]
}
```

**Build:** `cd daytrader && go build -o daytrader-bot ./cmd/`

---

### earnings-bot (`earnings/`)
Earnings play scanner. Game 1 strategy — buy before earnings, sell day-of.

**Logic:** Loads `earnings_dates` from earnings.json. For stocks reporting today/soon: check RVOL, gap direction. Gap-up + RVOL > 2 → long. Gap-down + RVOL > 2 → short (earnings miss).

**Config:** `earnings/earnings.json`
```json
{
  "days_before": 3, "stop_pct": 5.0, "budget": 200.0,
  "min_adv": 500000, "min_price": 3.0, "max_run_pct": 20.0,
  "watchlist": ["NVDA", "LUNR", "RKLB", "OKLO"],
  "earnings_dates": {
    "LUNR": "2026-05-14",
    "NVDA": "2026-05-20"
  }
}
```

**Build:** `cd earnings && go build -o earnings-bot ./cmd/`

---

### bounce-bot (`bounce/`)
RSI oversold bounce scanner. Game 2 strategy.

**Logic:** Fetches 14-day OHLCV from Yahoo Finance → computes RSI(14). If RSI < 30 and volume > ADV → BUY signal.

**Config:** `bounce/bounce.json` — watchlist, rsi_threshold, min_adv, budget

**Build:** `cd bounce && go build -o bounce-bot ./cmd/`

---

### controller (`controller/`)
Portfolio risk manager. Circuit breaker + NAV tracking.

**Features:**
- Circuit breaker: alerts at -10% NAV drawdown, e-stop at -15%
- T1/T2 split tracking (target 50/50)
- R/R filter: min 3.0x for swing, 2.0x for scalp
- Daily P&L logs in `controller/logs/`

**Config:** `controller/controller.json`
```json
{
  "t1_symbols": ["Z74", "ES3"],
  "position_max_pct": 30.0,
  "cash_reserve_pct": 15.0,
  "max_positions": 6,
  "circuit_breaker_alert_pct": 10.0,
  "circuit_breaker_estop_pct": 15.0,
  "rr_min_swing": 3.0,
  "rr_min_scalp": 2.0
}
```

**Build:** `cd controller && go build -o controller ./cmd/`

---

### index-trader (`index/`)
QQQ/VIX momentum bot. Game 5 strategy — TQQQ/SQQQ day trades.

**Logic:** Polls QQQ + VIX every 30s via Yahoo Finance. After grace period:
- QQQ > +0.3% and VIX < 25 → BUY TQQQ
- QQQ < -0.3% and VIX > 20 → BUY SQQQ

Monitors open position for stop (-5%) and target (+6%). EOD exit.

**Modes:** `--mode=watch` (signals only), `--mode=semi` (confirm), `--mode=live` (auto-execute via tiger-cli)

**Config:** `index/index.json`
```json
{
  "qqq_long_threshold": 0.3,
  "qqq_short_threshold": -0.3,
  "vix_max": 25.0, "vix_spike_min": 20.0,
  "budget": 200.0, "tqqq_shares": 2, "sqqq_shares": 4,
  "stop_pct": 5.0, "target_pct": 6.0,
  "grace_period_min": 10, "exit_by_min": 750,
  "scan_interval_sec": 30
}
```

**Build:** `cd index && go build -o index-trader ./cmd/`

---

## Next tool to build: notifier (`notifier/`)

**Purpose:** Send trade signals to Telegram and/or Discord. Every bot calls `notifier send` to push signals to the user's phone. This enables the paid subscription model.

**Interface:**
```bash
notifier send "LUNR gap +8.3% — BUY signal"
notifier send --symbol LUNR --signal BUY --price 35.50 --stop 32.00
notifier test   # sends a test message to verify config
```

**Config:** `notifier/notifier.json`
```json
{
  "telegram_bot_token": "",
  "telegram_chat_id": "",
  "discord_webhook_url": "",
  "enabled": true
}
```

**How other tools integrate:** Each bot shells out: `exec.Command("notifier", "send", message)`. Or the notifier binary is optional — if not found, bots log to stdout only.

**Build target:** `cd notifier && go build -o notifier ./cmd/`

**Monetization:** 
- Free: signals go to stdout only
- Paid tier: notifier configured with private Telegram channel bot token
- Owner runs signals on VPS, subscribers pay $15/month for channel access

---

## Versioning

Semantic versioning. Single version for the whole toolkit.

```
v0.1.0  ← current (8 tools, initial release, May 2026)
v0.2.0  ← notifier + alert added
v0.3.0  ← journal + backtest
v0.4.0  ← options chain + GoReleaser pre-built binaries
v1.0.0  ← all planned tools complete
```

Tag and push: `git tag v0.2.0 && git push origin v0.2.0`

Version embedded at build time: `-ldflags "-X main.Version=v0.2.0"`

---

## Planned tools (roadmap)

| Priority | Tool | Description |
|---|---|---|
| 1 | `notifier` | Telegram/Discord signal delivery — enables monetization |
| 2 | `alert` | Price alert daemon — monitors watchlist, fires when threshold hit |
| 3 | `journal` | Trade journal + P&L stats (SQLite, auto-populated from brokers) |
| 4 | `backtest` | Historical strategy validation via Yahoo Finance OHLCV |
| 5 | `options` | Options chain viewer (calls/puts, IV, OI) via Yahoo Finance |
| 6 | `watchlist` | Central watchlist.json shared across all tools |
| 7 | `pnl` | Unified P&L dashboard — Tiger + Moomoo + currency conversion |
| 8 | `market-hours` | Exchange clock — open/close times in SGT for all markets |
| 9 | `dividend` | T1 dividend tracker — ex-div dates, yield, payout history |

---

## Monetization strategy

1. **Broker referrals** (passive) — Tiger Commission Factory CPA + Moomoo Influencer Program. Links in README.
2. **Telegram signal channel** ($15/month) — run bots on VPS, push signals to private channel. 100 subscribers = $1,500/month.
3. **Strategy packs** ($19-49 one-time) — curated configs + watchlists: "SG Dividend Pack", "US Earnings Pack".
4. **SaaS hosting** ($9-19/month) — "we run the bots for you". Target non-technical traders.
5. **GitHub Sponsors** — structured donations.

---

## GitHub

- **Repo:** https://github.com/jpramirez/trade-kit
- **Account:** jpramirez (full repo access via `gh` CLI)
- **Auth:** `gh auth status` — logged in, `repo` scope
- **Push:** `git push origin main && git push origin --tags`

---

## Development rules

- **Paper mode always default** — `--live` flag required for real orders
- **Never commit** `.env`, `order-queue.json`, `*.properties`, compiled binaries
- **No personal data in code** — no account IDs, no personal paths
- **Yahoo Finance for all market data** — free, no API key
- **Each tool self-contained** — no shared state between tools at runtime
- **Rebuild binary after Go source changes** before testing CLI behavior
- **Tests before claiming a fix works** — `go test ./...`

---

## Key commands

```bash
# Build all tools
for d in tiger moomoo scheduler daytrader earnings bounce controller index; do
  cd $d && go build -o ${d}-cli ./cmd/ 2>/dev/null || go build -o ${d}-bot ./cmd/ 2>/dev/null || go build -o ${d} ./cmd/
  cd ..
done

# Test tiger (has full test suite)
cd tiger && go test ./...

# Check GitHub status
gh repo view jpramirez/trade-kit
gh release list

# Tag new version
git tag v0.2.0 -m "feat: add notifier and alert tools"
git push origin main --tags
```
