# Plan: Configurable minimum funded balance

| Field | Value |
|-------|-------|
| **ID** | `low-05` |
| **Priority** | Low |
| **Status** | `maybe` |
| **Depends on** | none |
| **Estimated gain** | Correctness / transparency — no hot-loop throughput change |

## Problem

`loadFunded()` silently skips addresses with `balance < 30000` satoshis. That drops ~27M rows (~46% of the TSV) — wallets with real but tiny balances (1–29,999 sats).

The threshold has been hard-coded since the initial commit with no comment, CLI flag, or log line explaining the filter.

## Goal

Make the minimum balance cutoff explicit and user-configurable. Default stays `30000` sats to preserve current performance and index size.

## Scope (this plan only)

- Named constant or config for `minBalanceSats` (default `30000`)
- CLI flag or env var override (pick one; document in `AGENTS.md`)
- Log the active threshold and skipped-row count at load time
- Update `loadFunded()` filter to use the setting

**Out of scope:** changing the default, bloom filter, cache format changes (cache plan can embed threshold in header later).

## Implementation steps

### Step 1 — Define default and override

```go
const defaultMinBalanceSats = 30000
```

Override via one of:

| Mechanism | Example |
|-----------|---------|
| CLI flag | `./btcfind 50000 --min-sats 1000` |
| Env var | `BTCFIND_MIN_SATS=1 ./btcfind 50000` |

Use `0` or `1` to include all funded rows (no balance filter).

### Step 2 — Apply in `loadFunded()`

Replace magic number:

```go
if !ok || balance < minBalanceSats {
    continue
}
```

Track `skipped` count for logging.

### Step 3 — Logging

After load:

```
Loaded 31,733,986 wallets in 44.5s (min balance: 30,000 sats, skipped 27,344,649 below threshold)
```

When threshold is `0`/`1`, expect ~59M rows and longer startup until `high-02` cache exists.

### Step 4 — Document tradeoffs

In `AGENTS.md` or plan notes:

| `min_sats` | ~Rows kept | Tradeoff |
|------------|------------|----------|
| `30000` (default) | ~31.7M | Current behavior; fastest startup |
| `1000` | ~45M+ | Broader coverage; larger index |
| `1` | ~59M | All funded addresses; ~2× load/sort cost |

Balances are **satoshis** from Blockchair TSV (`funded.tsv`).

## Files to touch

| File | Action |
|------|--------|
| `cryptogen.go` | Configurable threshold, skip counter, logging |
| `AGENTS.md` | Document flag/env and default |

## Verification

```bash
go build -o btcfind .
./btcfind 1000 8                    # default: ~31.7M loaded
BTCFIND_MIN_SATS=1 ./btcfind 1000 8 # all funded: ~59M loaded
```

- [ ] Default `30000` matches current loaded count (31,733,986)
- [ ] `min_sats=1` loads all nonzero-balance rows
- [ ] Log line shows threshold and skipped count
- [ ] Standard benchmark (`./btcfind 50000 8`) unchanged at default

## Rollback

Restore hard-coded `balance < 30000` check; remove flag/env parsing.