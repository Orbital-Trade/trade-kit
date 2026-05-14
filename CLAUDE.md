# CLAUDE.md — trade-kit

trade-kit is an open-source CLI trading toolkit for Tiger Brokers and Moomoo. Eight standalone Go binaries, each independently configurable via JSON.

## Repository layout

```
trade-kit/
├── tiger/          tiger-cli  — Tiger Brokers REST API wrapper
├── moomoo/         moomoo-cli — Moomoo/Futu OpenD TCP client (pure Go)
├── scheduler/      scheduler  — Order queue daemon + TypeExec
├── daytrader/      daytrader-bot — Gap-up scanner (Game 3)
├── earnings/       earnings-bot  — Earnings play scanner (Game 1)
├── bounce/         bounce-bot    — RSI bounce scanner (Game 2)
├── controller/     controller    — Portfolio risk manager
├── index/          index-trader  — QQQ/VIX momentum (TQQQ/SQQQ)
└── notifier/       notifier      — Telegram/Discord signal delivery [PLANNED]
```

## Building

Each tool has its own `go.mod`. Build individually:

```bash
cd tiger && go build -o tiger-cli ./cmd/
cd moomoo && go build -o moomoo-cli ./cmd/
cd scheduler && go build -o scheduler ./cmd/
cd daytrader && go build -o daytrader-bot ./cmd/
cd earnings && go build -o earnings-bot ./cmd/
cd bounce && go build -o bounce-bot ./cmd/
cd controller && go build -o controller ./cmd/
cd index && go build -o index-trader ./cmd/
```

Or build all at once:

```bash
make build   # uses Makefile
```

## Testing

```bash
cd tiger && go test ./...
cd moomoo && go test ./...   # pure Go now, no Python needed
```

## Versioning

Semantic versioning. Single version for the whole toolkit via git tags (`v0.1.0`, `v0.2.0`).

Version is embedded at build time via ldflags:
```bash
go build -ldflags "-X main.Version=v0.1.0" -o tiger-cli ./cmd/
```

## Key architecture rules

- **Paper mode is the default** — never execute real orders without `--live`
- **Never hardcode credentials** — all secrets via `.env` files or env vars
- **Yahoo Finance for quotes** — free, no API key, works for all markets
- **Tiger REST API** — RSA-signed requests, HMAC not needed
- **Moomoo OpenD TCP** — JSON encoding (body_type=1), no protobuf dependency
- **Each tool is self-contained** — no shared state, no inter-process dependencies at runtime

## Sensitive files (never commit)

- `.env` files
- `tiger_openapi_config.properties`
- `OpenD.xml`
- `scheduler/order-queue.json` (contains live order state and personal paths)
- `controller/logs/`
- `controller/nav-history.json`

## Planned tools

- `notifier` — Telegram/Discord integration (all bots call this to push signals)
- `alert` — price alert daemon
- `journal` — trade journal + P&L stats with SQLite
- `backtest` — historical strategy validation via Yahoo Finance
- `options` — options chain viewer
- `watchlist` — central watchlist shared across all tools

## Module paths

All tools use `github.com/jpramirez/trade-kit/<toolname>` as module path.

## GitHub

Repo: https://github.com/jpramirez/trade-kit
Account: jpramirez (Juan Pablo Ramirez, Epyphite Pte Ltd, Singapore)
