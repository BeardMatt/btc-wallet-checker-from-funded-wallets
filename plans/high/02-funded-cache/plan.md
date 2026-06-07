# Plan: Preprocess funded.tsv into cache file

| Field | Value |
|-------|-------|
| **ID** | `high-02` |
| **Priority** | High |
| **Status** | `completed` |
| **Depends on** | `medium-01` |
| **Estimated gain** | ~27s cold startup → ~1s on cache hit |

## Problem

Every run parses a 2.6 GB TSV, filters 31M rows, and sorts strings. This takes ~27 seconds even when `funded.tsv` has not changed.

## Goal

After first parse, write a binary cache. On subsequent runs, load the cache if `funded.tsv` mtime matches.

## Scope (this plan only)

- Define binary cache format for partitioned funded sets
- Write cache after `loadFunded()` completes
- Load cache on startup when valid

**Out of scope:** changing parse logic (see `medium-04`), bloom filter (see `medium-02`).

## Implementation steps

### Step 1 — Define cache format

File: `funded.cache` (or `funded.cache.bin`)

```
Header:
  magic:    [4]byte  "BFND"
  version:  uint32   1
  mtime:    int64    funded.tsv ModTime unix nano
  counts:   uint32×4 legacy, p2sh, segwit_v0, taproot_v1

Body:
  legacy:     count × [20]byte (sorted)
  p2sh:       count × [20]byte (sorted)
  segwit_v0:  count × [20]byte (sorted)
  taproot_v1: count × [32]byte (sorted)
```

### Step 2 — Write cache function

```go
func writeFundedCache(path string, sets FundedSets, srcMtime time.Time) error
```

Call at end of `loadFunded()` after sorting.

### Step 3 — Read cache function

```go
func readFundedCache(path string, expectedMtime time.Time) (FundedSets, error)
```

Validate magic, version, and mtime before trusting data.

### Step 4 — Integrate in startup

In `loadFunded()`:

1. Stat `funded.tsv` for mtime
2. If `funded.cache` exists and mtime matches → `readFundedCache`
3. Else → parse TSV, then `writeFundedCache`

### Step 5 — Invalidate on download

In `funded_download.go`, delete `funded.cache` after successful download/replace of `funded.tsv`.

## Files to touch

| File | Action |
|------|--------|
| `funded_cache.go` | New — read/write cache |
| `cryptogen.go` | Load from cache when valid |
| `funded_download.go` | Delete stale cache on update |

## Verification

```bash
# First run (builds cache)
time ./btcfind 1000 4

# Second run (should load cache)
time ./btcfind 1000 4
```

- [x] Second run startup < 3 seconds (0.65s cache load)
- [x] Cache invalidated after `funded.tsv` download
- [x] Loaded counts match uncached parse (four byte buckets; Other not cached)

## Rollback

Delete `funded.cache` and fall back to TSV-only parsing. No code path should require the cache to exist.