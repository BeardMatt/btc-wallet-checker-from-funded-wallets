# Plan: Bloom filter pre-check

| Field | Value |
|-------|-------|
| **ID** | `medium-02` |
| **Priority** | Medium |
| **Status** | `completed` |
| **Depends on** | `medium-01`, `high-01` |
| **Estimated gain** | 1.5–3× lookup throughput |

## Problem

Every key runs up to five binary searches over millions of entries. Almost all keys miss, but each search still walks O(log n) comparisons.

## Goal

Add a bloom filter per funded bucket. On miss (definitely not funded), skip binary search entirely. On possible hit, confirm with exact sorted-slice lookup.

## Scope (this plan only)

- Build bloom filters at load time from partitioned sets
- Check bloom before binary search in hot loop

**Out of scope:** cache format changes (add bloom to cache in a follow-up after this plan works).

## Implementation steps

### Step 1 — Choose bloom filter library

Add dependency or implement minimal bloom filter:

```bash
go get github.com/bits-and-blooms/bloom/v3
```

Or a small in-tree implementation to avoid deps.

### Step 2 — Extend FundedSets

```go
type FundedSets struct {
    Legacy    [][20]byte
    P2SH      [][20]byte
    SegwitV0  [][20]byte
    TaprootV1 [][32]byte

    LegacyBloom    *bloom.BloomFilter
    P2SHBloom      *bloom.BloomFilter
    SegwitV0Bloom  *bloom.BloomFilter
    TaprootV1Bloom *bloom.BloomFilter
}
```

### Step 3 — Build filters at load

After sorting each slice:

```go
bf := bloom.NewWithEstimates(uint(n), 0.0001)
for _, h := range legacy {
    bf.Add(h[:])
}
```

Target false-positive rate: 0.01% (1 in 10,000). Tune m/k for 31M entries.

### Step 4 — Two-stage lookup

```go
func maybeFunded20(bf *bloom.BloomFilter, set [][20]byte, key [20]byte) bool {
    if bf != nil && !bf.Test(key[:]) {
        return false
    }
    return inFunded20(set, key)
}
```

### Step 5 — Benchmark false positive rate

Log bloom stats at startup: estimated FP rate, bit count, hash count.

## Files to touch

| File | Action |
|------|--------|
| `funded_index.go` | Bloom filter fields + build |
| `cryptogen.go` | Two-stage lookup in hot loop |
| `go.mod` | Optional bloom dependency |

## Verification

```bash
./btcfind 500000 8
```

- [x] Throughput improves vs `high-01` alone (+5% hot loop vs high-02 with cache)
- [x] No false negatives (bloom miss = definitely not in set)
- [x] Unit test confirms all entries in test sets return `bf.Test() == true`

## Rollback

Set bloom pointers to `nil` and fall through to direct binary search. Zero behavior change.