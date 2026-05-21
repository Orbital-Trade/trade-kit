# Changelog

All notable changes to trade-kit will be documented here.

Format: [Semantic Versioning](https://semver.org/) — `MAJOR.MINOR.PATCH`

---

## [0.2.0] — 2026-05-21

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
