# Changelog

All notable changes to trade-kit will be documented here.

Format: [Semantic Versioning](https://semver.org/) — `MAJOR.MINOR.PATCH`

---

## [0.5.0] — 2026-06-06

### Added

**alert — price threshold alert daemon**
- New `alert/` tool: watches a symbol list and fires notifications when price crosses configured thresholds
- `alert daemon` — polls continuously (configurable interval), fires once per threshold cross (or every poll if `"repeat": true`)
- `alert check` — one-shot check; print triggered alerts and exit
- `alert list` — show configured thresholds alongside live prices
- Config: `alert/alert.json` — symbols with `above`/`below` thresholds, `poll_interval_sec`, `repeat` flag
- Integrates with `notifier` for Telegram/Discord delivery; falls back to stdout if notifier not in PATH
- Build: `cd alert && go build -o alert ./cmd/`

**journal — SQLite trade journal**
- New `journal/` tool: records fills into a local SQLite database with FIFO P&L reporting
- `journal add BUY|SELL <SYMBOL> <QTY> <PRICE>` — record a trade manually
- `journal list [--symbol SYM] [--days N]` — filterable trade history
- `journal pnl [--symbol SYM]` — realized P&L per symbol (FIFO cost basis) with win/loss counts
- Duplicate broker order IDs silently ignored (partial unique index — only enforced when `order_id` is set)
- Pure-Go SQLite via `modernc.org/sqlite` — no CGO, no system dependencies
- Database: `~/.trade-kit/journal/trades.db`
- Build: `cd journal && go build -o journal ./cmd/`

**options — options chain viewer**
- New `options/` tool: displays calls and puts for any US optionable stock using Yahoo Finance
- `options chain <SYMBOL>` — nearest expiry chain (calls + puts)
- `options chain <SYMBOL> --expiry YYYY-MM-DD --calls|--puts --json` — filtered output
- `options expiries <SYMBOL>` — list all available expiry dates with DTE
- Handles Yahoo Finance session auth (fc.yahoo.com cookie + crumb) internally — no API key required
- Near-the-money rows highlighted with `>`, ITM contracts flagged with `*`
- Build: `cd options && go build -o options ./cmd/`

**backtest — historical strategy validation**
- New `backtest/` tool: replays strategies against historical OHLCV data and produces a performance report
- `backtest run --strategy daytrader|bounce --symbol SYM --from YYYY-MM-DD [--to YYYY-MM-DD] [--json]`
- Report: total trades, win rate, avg win/loss, max drawdown, max consecutive losses, full trade log
- Strategies: `daytrader` (gap-up momentum with stop/target), `bounce` (RSI oversold entry, max-hold exit)
- Data sources: Yahoo Finance (default, no key), Alpha Vantage (free tier), Polygon.io (free tier)
- Source selected via `"data_source"` in `backtest.json`; strategy parameters fully configurable
- Build: `cd backtest && go build -o backtest ./cmd/`

**watchlist — central symbol list**
- `watchlist.json` at repo root and `~/.trade-kit/watchlist.json` as shared symbol source
- `daytrader-bot`, `bounce-bot`, and `earnings-bot` all check for the central file at startup
- Falls back silently to tool-local config when central file is absent — no breaking change

**Makefile**
- `make all` — builds all 12 tools in one command
- `make <tool>` — build individual tool (tiger, moomoo, scheduler, etc.)
- `make test` — runs the tiger test suite
- `make clean` — removes all compiled binaries

**Per-tool README.md files**
- Every tool directory now has a README covering build, config reference, commands/flags, and examples
- 7 new READMEs: moomoo, scheduler, index, notifier, alert, journal, options
- 4 updated READMEs: tiger, daytrader, earnings, bounce (added examples sections)

### Fixed

**scheduler: TypeExec shell injection mitigation**
- `scheduler daemon` now blocks TypeExec orders by default
- Pass `--allow-exec` to opt in; without it, exec orders fail with a clear error message
- Prevents arbitrary shell commands from running if the queue file is written by an untrusted source

**moomoo: connection close error no longer silently dropped**
- `Close()` now returns the `net.Conn` error
- `defer c.Close()` in main logs any close failure to stderr

**DST bug: hardcoded UTC-4 offset replaced in daytrader, bounce, index**
- `daytrader/internal/strategy/scanner.go`, `bounce/cmd/main.go`, `index/cmd/main.go` all used `UTC().Add(-4 * time.Hour)` (EDT only)
- Replaced with `time.LoadLocation("America/New_York")` — now handles EST/EDT transitions automatically
- Matches the fix applied to `scheduler` in v0.3.1 (OTK-1)

**tiger/ops/quote.go: errors silently swallowed in yahooQuote()**
- `http.NewRequest` error was ignored (nil request → panic in `http.Client.Do`)
- `io.ReadAll` error was ignored (empty body → opaque JSON parse failure)
- Both errors now returned to caller with context

**index-trader: P&L calculated as zero on quote fetch failure**
- `FetchQuote(pos.Symbol)` errors were discarded with `_`
- A failed quote returned a zero-value `Quote{}`, making stop/target checks never trigger
- Now logs the error and skips the position check for that tick

**index-trader: logf() was writing to stdout instead of stderr**
- Daemon error logs now go to stderr, consistent with all other tools

**journal: timestamp parse errors produce silent zero-value dates**
- `time.Parse` errors on `filled_at` and `created_at` are now returned to the caller
- Previously a malformed date stored in the DB would silently produce a year-0001 timestamp

**earnings/strategy: http.NewRequest and io.ReadAll errors dropped in session setup**
- `ensureSession()` now checks all errors and returns early; crumb degrades gracefully to empty

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
