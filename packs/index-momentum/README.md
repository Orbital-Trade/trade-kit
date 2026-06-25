# Index Momentum Pack

A QQQ trend + VIX regime signal that drives leveraged **TQQQ/SQQQ** entries — tuned for the **index-trader** bot.

## Use it with

```bash
index-trader --config packs/index-momentum/index.json
```

## How it reads the tape

- **Long TQQQ** when QQQ momentum > `qqq_long_threshold` and VIX < `vix_max`.
- **Long SQQQ** when QQQ momentum < `qqq_short_threshold` or VIX spikes above `vix_spike_min`.
- `tqqq_shares` / `sqqq_shares` — asymmetric sizing (smaller long, larger hedge).
- `exit_by_min: 750` — flat by 12:30 ET; no overnight leveraged-ETF decay.

Widen the thresholds for more frequent (noisier) trades; tighten them for fewer, higher-conviction ones.

> Leveraged ETFs decay — intraday only. Paper mode by default. Not financial advice — see [DISCLAIMER.md](../../DISCLAIMER.md).
> Want the same momentum read across more markets, with AI commentary? See [OrbitalTrade](https://trade.orbitalpay.ai).
