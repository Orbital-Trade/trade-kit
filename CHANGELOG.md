# Changelog

All notable changes to trade-kit will be documented here.

Format: [Semantic Versioning](https://semver.org/) — `MAJOR.MINOR.PATCH`

---

## [0.4.0] — 2026-05-27

### Added

**notifier — Telegram and Discord signal delivery**
- New `notifier/` tool: delivers trade signals to Telegram and/or Discord
- `notifier send "<text>"` — free-text message to all configured channels
- `notifier send --symbol SYM --signal BUY --price P --stop S --target T --qty N --strategy name --note text` — structured signal with formatted output (icon, R:R ratio, %, SGT timestamp)
- `notifier test` — sends a test message to verify all channels are reachable
- `notifier status` — shows which channels are configured
- Config: `notifier/notifier.json` — `telegram_bot_token`, `telegram_chat_id`, `discord_webhook_url`, `enabled`
- Free tier: no channels configured → signals go to stdout only (no crash, no config required)
- Paid tier: configure private Telegram channel for subscriber delivery
- Falls back to `~/.trade-kit/notifier/notifier.json` if local config not found
- Build: `cd notifier && go build -o notifier ./cmd/`

**Bot integration — all scanner bots now call notifier on signals**
- `daytrader-bot`: notifies on every new signal written to the bus (scan + run modes) and on trade execution
- `bounce-bot`: notifies on RSI oversold signal write and on trade execution
- `earnings-bot`: notifies on ENTER/EXIT signal write (runScan) and on trade execution
- `index-trader`: notifies when QQQ/VIX signal fires (before semi/live execution)
- Integration is non-blocking (goroutine) — trading loop never waits on delivery
- Silent fallback: if `notifier` binary is not found in PATH, bots continue normally with stdout only

---

## [0.3.1] — 2026-05-25

### Fixed

**scheduler: timezone correctness**
- Replaced hardcoded UTC-4 (EDT) offset with `time.LoadLocation("America/New_York")` for proper EST/EDT transitions
- Applies to both `computeExecuteAt` (queue scheduling) and `printTimeHeader` (clock display)
- Orders scheduled in Dec–Mar were firing 1 hour early due to the fixed EDT offset; now uses DST-aware lookup
- `nextSessionTime` rewritten to accept `*time.Location` instead of a fixed duration offset
- Falls back to `UTC-5` (EST) if system timezone database is unavailable

**scheduler: input validation**
- `buy`/`sell` commands now reject empty symbols, non-positive quantities, and non-positive limit prices
- `stop` command validates symbol, quantity > 0, and `--price` > 0
- `target` command validates symbol, quantity > 0, and `--price` > 0
- Errors print to stderr and exit with code 1 before anything is queued

**tiger/ops: error propagation in `analyze`**
- `yahooKline`: `http.NewRequest` error was silently discarded; a nil request would panic in `http.Client.Do` — now returns the error
- `yahooKline`: `io.ReadAll` error was silently discarded; a failed read produced an opaque JSON parse error — now returns the error with context

**tiger/client: `json.Marshal` error in `Call`**
- `json.Marshal(params)` error was discarded with `_`; now propagated so callers see the failure

**tiger/ops: emergency close error logged in `FuturesEntry`**
- When stop-order response parsing fails after a live entry, the emergency `FuturesClose` error was silently dropped
- Both the stop-parse failure and any close failure are now logged to stderr before returning

---

## [0.3.0] — 2026-05-21

### Added

**tiger-cli: `analyze` command**
- Multi-timeframe technical analysis: 1D / 1H / 15m
- Indicators: RSI(14), MACD(12/26/9), Bollinger Bands(20,2), EMA(20/50/200)
- Bias scoring per timeframe (BULLISH / NEUTRAL / BEARISH) + overall alignment
- Tiger `kline` API primary; automatic fallback to Yahoo Finance when quota full
- Futures support via `--futures` flag (uses `future_kline` + contract auto-resolution)
- `--json` flag for scripting and MCP piping
- Corrected RSI overbought (>70) and %B > 1.0 scoring — treated as overextension, not strength

---

## [0.2.0] — 2026-05-21

### Added

**tiger-cli: `markov` command**
- Markov regime model: labels every historical trading day as BULL / SIDE / BEAR (20-day return ±5% thresholds)
- Builds a 3×3 transition matrix from full price history (2 years of daily bars via Yahoo Finance)
- Computes tomorrow's state distribution from current state row
- N-day forecasts via matrix exponentiation (2d / 5d / 10d)
- Signal = P(bull) − P(bear) with direction (LONG / SHORT / NEUTRAL) and confidence (HIGH / MEDIUM / LOW)
- Stickiness detection: highlights states where self-transition probability > 50%
- Works for US stocks and SGX (ES3.SI etc.), JSON output via --json

---

## [0.1.0] — 2026-05-15

### Initial release

**Tools included:**
- `tiger-cli` — Tiger Brokers API: positions, quotes, buy/sell/stop/modify/cancel, orders
- `moomoo-cli` — Moomoo/Futu API: pure Go TCP client (no Python bridge), same interface as tiger-cli
- `scheduler` — Order queue daemon with market-window scheduling and TypeExec support
- `daytrader-bot` — Gap-up scanner with RVOL filter and earnings mode (`--earnings`)
- `earnings-bot` — Earnings play scanner: RVOL, gap direction, long/short on miss/beat
- `bounce-bot` — RSI oversold bounce scanner with volume confirmation
- `controller` — Portfolio risk manager: circuit breaker, T1/T2 NAV tracking, P&L logs
- `index-trader` — QQQ/VIX momentum: auto-signals TQQQ/SQQQ with watch/semi/live modes

**Architecture:**
- All tools are standalone Go binaries with zero runtime dependencies
- Each tool is independently configurable via a single JSON file
- Paper mode by default — `--live` required for real order execution
- Yahoo Finance for market data (no API key required)
- Tiger REST API + RSA signing for broker connectivity
- Futu OpenD TCP protocol with JSON encoding for Moomoo connectivity
