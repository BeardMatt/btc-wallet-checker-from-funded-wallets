# Plan: mmap funded cache

| Field | Value |
|-------|-------|
| **ID** | `low-13` |
| **Priority** | **Low–Medium** |
| **Status** | `pending` |
| **Depends on** | `high-02`, `low-12` (cache v3 format must be stable) |
| **Estimated gain** | **Faster startup**, lower RSS spike; marginal hot-loop locality |

## Problem

`readFundedCache` copies hundreds of MB into Go heap slices on every run. That costs allocation time, increases RSS, and scatters data across the heap instead of one contiguous mapping.

## Goal

Memory-map `funded.cache` and use mapped byte ranges for hash buckets and bloom filters (read-only), falling back to heap load if mmap unavailable.

## Scope (this plan only)

- `syscall.Mmap` / `golang.org/x/sys/unix` for cache file
- Parse header → slice views into mapped region (`unsafe` or binary.Read from sub-slices)
- Bloom filters: `UnmarshalBinary` from mapped sub-range OR mmap-friendly bloom layout
- `Close`/`munmap` on shutdown (or runtime finalizer)

**Out of scope:** writable mmap, cache rebuild path (still writes via `writeFundedCache` then remaps).

## Implementation steps

### Step 1 — Mmap reader for v3 cache

```go
type MappedFundedCache struct {
    data []byte
    sets FundedSets // slices alias into data
}
```

### Step 2 — Integrate in `loadFunded`

Try mmap first; on failure or version mismatch, existing parse path.

### Step 3 — Benchmark startup

Measure `startup_seconds` before/after on cache hit.

## Files to touch

| File | Action |
|------|--------|
| `funded_cache.go` | Mmap load path |
| `funded_mmap.go` (new) | Platform mmap helpers |

## Verification

- [ ] Cache hit startup &lt; current ~0.8–0.9s (target ~30–50% reduction)
- [ ] Hot-loop keys/sec unchanged (±1%)
- [ ] Works on Linux; graceful fallback elsewhere

## Rollback

Remove mmap path; heap-only `readFundedCache`.