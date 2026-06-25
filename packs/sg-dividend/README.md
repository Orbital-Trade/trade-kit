# SG Dividend Pack

A curated watchlist of SGX blue-chip dividend payers and large-cap S-REITs, for **income-focused, long-term holding** — not day trading.

| Group | Tickers |
|---|---|
| Banks | D05 (DBS), O39 (OCBC), U11 (UOB) |
| Blue chips | Z74 (Singtel), C6L (SIA), F34 (Wilmar) |
| S-REITs | C38U (CICT), A17U (CLAR), M44U (MLT), N2IU (MPACT), ME8U (MINT), K71U (Keppel REIT), J69U (FCT), BUOU (Frasers L&I) |
| Index | ES3 (STI ETF) |

## Use it with

```bash
# Price alerts on the whole pack
alert --watchlist packs/sg-dividend/watchlist.json

# Track your holdings
journal sync

# Portfolio risk / circuit breaker
controller --watchlist packs/sg-dividend/watchlist.json
```

> These are SGX ticker codes. Not financial advice — see [DISCLAIMER.md](../../DISCLAIMER.md).
> Want yield, gearing, P/NAV, and DPU overlays automatically? The [SGX REIT Toolkit](https://trade.orbitalpay.ai) browser extension does it on SGX.com, Yahoo Finance, and POEMS.
