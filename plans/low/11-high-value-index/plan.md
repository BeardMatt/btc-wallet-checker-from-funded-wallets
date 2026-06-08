# Plan: High-value-only funded index

| Field | Value |
|-------|-------|
| **ID** | `low-11` |
| **Priority** | **Medium** (expected value, not raw keys/sec) |
| **Status** | `completed` |
| **Depends on** | `low-12` (balance metadata for tiering and hit display) |
| **Estimated gain** | Smaller/faster index; **higher expected sats per hit** |

## Problem

The funded set includes ~31M addresses ≥30k sats. Most have small balances. Searching the full set maximizes hit *count* but not expected *value* per key tried.

A user hunting worthwhile targets may prefer a smaller “whale” tier (e.g. ≥1 BTC, ≥0.1 BTC).

## Goal

Support loading or building a **tiered funded index** so searches can target high-balance addresses only.

## Scope (this plan only)

### Options (pick one in implementation)

| Approach | Description |
|----------|-------------|
| **A. CLI threshold** | `--min-balance 100000000` (1 BTC in sats) at load — overlaps `low-05` conceptually but as whale-focused feature |
| **B. Separate cache** | `funded-whale.cache` built from TSV with high threshold; `--index whale` |
| **C. Tier presets** | `--tier whale\|standard` mapping to thresholds |

Recommend **A + cache header field** embedding threshold so cache invalidates correctly.

### Effects

| Threshold | ~Rows | Tradeoff |
|-----------|-------|----------|
| 30k sats (default) | ~31M | Current behavior |
| 1 BTC | ~thousands–low millions | Faster load, smaller blooms, rare hits |
| 10 BTC | smaller still | Puzzle-adjacent targeting |

Log active tier and row count at startup.

**Out of scope:** on-chain balance refresh APIs.

## Implementation steps

### Step 1 — Threshold at load

Use balance from TSV parse (requires `low-12` balance storage or parallel parse-time filter).

### Step 2 — Cache invalidation

Embed `min_balance_sats` in `funded.cache` v3 header; mismatch → rebuild.

### Step 3 — CLI and docs

```bash
./btcfind 50000 8 --min-balance 100000000
```

## Files to touch

| File | Action |
|------|--------|
| `cryptogen.go` | Threshold filter in `parseFundedTSV` |
| `funded_cache.go` | Header field for threshold |
| `cli.go` | `--min-balance` or `--tier` |
| `AGENTS.md` | Tier tradeoffs |

## Verification

- [x] Default 30k matches current wallet count
- [x] `--min-balance 100000000` loads fewer rows, faster startup
- [x] Hit banner / `wallets.txt` shows balance (via `low-12`)

## Rollback

Remove tier flag; restore fixed 30k filter.