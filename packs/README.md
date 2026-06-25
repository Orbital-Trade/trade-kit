# Strategy Packs

Ready-to-use config + watchlist bundles for the trade-kit bots. Drop-in starting points — copy a pack's config, point a bot at it, and tune from there. **Paper mode by default; not financial advice** (see [DISCLAIMER.md](../DISCLAIMER.md)).

| Pack | For | Strategy |
|---|---|---|
| [SG Dividend](sg-dividend/) | `alert`, `journal`, `controller` | SGX blue-chip dividend payers + large-cap S-REITs — income, long-term hold |
| [US Earnings](us-earnings/) | `earnings-bot` | Liquid US large-caps — pre-announcement run-up |
| [Index Momentum](index-momentum/) | `index-trader` | QQQ trend + VIX regime → TQQQ/SQQQ |

## Using a pack

```bash
# Most bots take a --config flag pointing at the pack's JSON:
earnings-bot --config packs/us-earnings/earnings.json
index-trader --config packs/index-momentum/index.json

# Watchlist-only packs work with any tool that reads a watchlist:
alert --watchlist packs/sg-dividend/watchlist.json
```

Each config carries `_pack` / `_description` metadata keys (ignored by the bots) so a pack is self-documenting.

## Want more?

These packs are deliberately simple. [**OrbitalTrade**](https://trade.orbitalpay.ai) is the hosted platform built on this toolkit — AI-generated theses, a live multi-market scanner (US/SGX/HK), and a browser extension. The [Chrome extension](https://chromewebstore.google.com/detail/orbitaltrade/kfndmcgcalllbgjiebgjhmefhfoiimde) is free.
