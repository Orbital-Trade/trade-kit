# Changelog

All notable changes to trade-kit will be documented here.

Format: [Semantic Versioning](https://semver.org/) — `MAJOR.MINOR.PATCH`

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
