# CLAUDE.md — trade-kit

## CRITICAL RULES — read before doing anything

1. **NEVER leak PII in commits, code, or output.** No real account IDs, no home directory paths, no email addresses, no API keys, no private key blobs. Use placeholders: `YOUR_TIGER_ID`, `YOUR_ACCOUNT`, `~/.trade-kit/`, etc.
2. **NEVER modify files outside this repo.** If integration needs changes in OrbitalTrade/desktop or any other repo, document what needs to change in Plane/Outline and let the owning agent implement it.
3. **NEVER add Co-Authored-By or Claude attribution to commits.**
4. **Paper mode is always default.** `--live` flag required for real orders. Never commit code that defaults to live.
5. **Never commit** `.env`, `order-queue.json`, `nav-history.json`, `*.properties`, compiled binaries, `signals.json`, `*-trades.json`.
6. **Sanitize recordings.** Any asciinema/demo output must be run through `sanitize-cast.py` before publishing. Verify with `--verify` flag.

---

## What this is

trade-kit is an **open-source multi-broker CLI toolkit** for retail traders. 15 Go tools — scan, backtest, trade, manage risk — across Tiger Brokers, Moomoo, and eToro.

**Repo:** https://github.com/Orbital-Trade/trade-kit
**License:** MIT
**Current version:** check `VERSION` file (not this doc)

---

## Build

```bash
make all                    # build all 15 tools
make tiger                  # build one tool
make test                   # run tiger + etoro test suites
```

Standalone tools (scheduler, daytrader, etc.) use `GOWORK=off` because they're not in the Go workspace. The Makefile handles this.

**Go workspace:** `go.work` links `sidecar/`, `tiger/`, `moomoo/`, `etoro/`, `daytrader/`, `bounce/`, `earnings/`, `index/` for cross-module imports.

---

## Tools (15 total)

| Dir | Binary | Purpose |
|-----|--------|---------|
| `tiger/` | `tiger-cli` | Tiger Brokers CLI — quotes, orders, positions, analyze, markov |
| `moomoo/` | `moomoo-cli` | Moomoo/Futu CLI — pure Go TCP client via OpenD |
| `etoro/` | `etoro-cli` | eToro CLI — REST API, demo/live, watchlists, alerts |
| `scheduler/` | `scheduler` | Order queue daemon — market-window scheduling |
| `daytrader/` | `daytrader-bot` | Gap-up scanner (Game 3) |
| `earnings/` | `earnings-bot` | Earnings play scanner (Game 1) |
| `bounce/` | `bounce-bot` | RSI oversold bounce scanner (Game 2) |
| `index/` | `index-trader` | QQQ/VIX momentum — TQQQ/SQQQ (Game 5) |
| `controller/` | `controller` | Risk manager — circuit breaker, NAV, gate, e-stop |
| `backtest/` | `backtest` | Historical strategy replay via Yahoo Finance |
| `options/` | `options` | Options chain viewer — calls/puts, IV, OI |
| `notifier/` | `notifier` | Signal delivery — Telegram + Discord |
| `alert/` | `alert` | Price threshold daemon |
| `journal/` | `journal` | SQLite trade journal with FIFO P&L |
| `sidecar/` | `trade-kit` | HTTP server — bridges Electron desktop app to brokers |

Each tool: own `go.mod`, own binary, own JSON config. No shared runtime deps.

---

## Architecture

```
tiger/client/     → TigerClient (RSA auth, REST)
moomoo/client/    → Client (TCP to OpenD)
etoro/client/     → EtoroClient (token auth, REST)

*/strategy/       → exported strategy packages (FetchSetup, Evaluate)
*/internal/       → broker, signal bus, store (NOT importable by sidecar)

sidecar/
  broker/         → unified BrokerAdapter (Positions, Account, Orders, Buy, Sell)
  recipe/         → goroutine-based strategy runner (membus, SSE events)
  handler/        → HTTP handlers for REST API
  server/         → HTTP server, auth middleware, SSE broadcaster
```

**Key pattern:** Each broker CLI has a `NewFromCreds()` constructor (used by sidecar) and a `New()` constructor (used by CLI, reads .env files).

**Strategy packages** are exported at `*/strategy/` (not `internal/`) so the sidecar can import them via Go workspace.

---

## Sidecar API (for desktop app agents)

The sidecar (`sidecar/trade-kit`) is spawned by the Electron desktop app as `trade-kit serve --port 19090`. Auth via `ORBITAL_AUTH_TOKEN` env var → Bearer token.

Key endpoints:
- `GET /v1/status` — version, uptime, paper mode, broker states
- `GET/POST /v1/brokers/{id}/connect|test|disconnect|positions|account|orders|buy|sell`
- `GET/POST /v1/recipes/{id}/start|stop|signals`
- `POST /v1/settings/paper-mode`
- `GET /v1/events` — SSE stream

Full API reference: see Outline doc "Sidecar — Go HTTP Server for Desktop App"

---

## Commit rules

- Check `VERSION` for current version before referencing it
- Bump `VERSION` + update `CHANGELOG.md` for feature releases
- Tag format: `git tag v0.X.0 -m "description"` then `git push origin main --tags`
- No Co-Authored-By lines
- No personal paths in commit messages
- Commit messages: conventional commits style (`feat:`, `fix:`, `chore:`)

---

## Demo / recording rules

- `export TIGER_LOG_LEVEL=off` — suppresses account IDs in stderr
- `export ETORO_LOG_LEVEL=off` — same for eToro
- Always use `sanitize-cast.py` on `.cast` files before converting to GIF/MP4
- Run `python3 sanitize-cast.py input.cast output.cast --verify` — must show PASS
- Add any new PII patterns to `sanitize-cast.py` REPLACEMENTS dict before recording

---

## External systems

- **Plane (ticketing):** OTK project for trade-kit issues, OTR project for OrbitalTrade issues
- **Outline (docs):** Orbital Trade collection — feature docs, NOT ticket logs
- **GitHub:** Orbital-Trade/trade-kit — push to main (branch protection bypassed)

---

## What NOT to do

- Don't modify OrbitalTrade/desktop/ — that's a different repo with a different agent
- Don't hardcode account IDs, Tiger IDs, or home directory paths
- Don't add demo recordings (.cast, .gif, .mp4) to git unless explicitly asked
- Don't create Outline pages titled with ticket numbers (OTK-N) — use feature names
- Don't run `--live` mode in any demo or test
- Don't add external Go dependencies — stdlib only (except journal which uses modernc.org/sqlite)
