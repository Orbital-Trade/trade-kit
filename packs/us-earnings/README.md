# US Earnings Pack

Liquid US large-caps with regular, market-moving earnings — tuned for the **earnings-bot** pre-announcement run-up strategy (buys into the drift ahead of the report, exits before the print).

## Use it with

```bash
earnings-bot --config packs/us-earnings/earnings.json
```

## Config notes

- `days_before: 3` — enter ~3 trading days before the announcement.
- `stop_pct: 5.0` — hard stop.
- `max_run_pct: 20.0` — skip names that have already run >20% (chasing risk).
- `min_adv` / `min_price` — liquidity filters (avg daily volume, min share price).
- **`earnings_dates` must be refreshed each quarter** — the bot only acts on names with a known upcoming date. Free source: [nasdaq.com earnings calendar](https://www.nasdaq.com/market-activity/earnings).

> Paper mode by default. Not financial advice — see [DISCLAIMER.md](../../DISCLAIMER.md).
> For AI-generated earnings theses and a hosted scanner across US/SGX/HK, see [OrbitalTrade](https://trade.orbitalpay.ai).
