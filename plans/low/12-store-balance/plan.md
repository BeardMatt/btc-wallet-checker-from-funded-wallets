# Plan: Store balance in funded index

| Field | Value |
|-------|-------|
| **ID** | `low-12` |
| **Priority** | **Medium** |
| **Status** | `completed` |
| **Depends on** | none |
| **Estimated gain** | UX / correctness — enables `low-11`, richer hits |

## Problem

`funded.tsv` includes balance per address, but `loadFunded()` discards it after the 30k-sat filter. On hit, the user sees address and WIF but not **how much** was found.

## Goal

Retain balance (satoshis) keyed by address hash so hits can display and log balance without external APIs.

## Scope (this plan only)

### Data structure

Parallel maps or packed arrays aligned with sorted hash slices:

```go
type FundedSets struct {
    // existing buckets...
    LegacyBalance    []uint64  // same length as Legacy, sorted with hashes
    // or map[[20]byte]uint64 for sparse — prefer parallel slices for cache locality
}
```

Lookup on hit: after binary search finds hash index, read `balance[idx]`.

### Cache format v3

- Bump `fundedCacheVersion` to 3
- Write balance arrays after each hash bucket
- Invalidate v2 caches automatically (version mismatch → cold parse)

### Output

- Hit banner: `Balance  1.23456789 BTC (123,456,789 sats)`
- `wallets.txt`: `balance_sats: 123456789`
- `low-11` tier filtering uses same balance at load

**Out of scope:** live balance updates from network.

## Implementation steps

### Step 1 — Parse and store balance

In `parseTSVLine` / `addAddress`, record balance alongside hash (handle duplicate addresses — keep max balance).

### Step 2 — `balanceForMatch(kind, hash)` lookup

### Step 3 — Cache v3 read/write

### Step 4 — Wire into `PrintHit` and `appendWalletHit`

## Files to touch

| File | Action |
|------|--------|
| `funded_index.go` | Balance slices + lookup |
| `funded_cache.go` | v3 format |
| `cryptogen.go` | Parse path |
| `output.go` | Banner balance line |
| `wallet_log.go` | Log balance_sats |

## Verification

- [x] Inject-index-hit shows balance for known funded entry
- [x] v2 cache rebuilds to v3 on first run
- [x] Benchmark hot loop regression &lt;2% (balance only read on hit)

## Rollback

Drop balance arrays; revert to cache v2.