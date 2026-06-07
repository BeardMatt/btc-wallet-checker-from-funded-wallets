# Plan: Partition funded set by address type

| Field | Value |
|-------|-------|
| **ID** | `medium-01` |
| **Priority** | Medium |
| **Status** | `completed` |
| **Depends on** | none |
| **Estimated gain** | Foundation for 2–5× lookup improvement |

## Problem

All 31M funded addresses live in one `[]string` sorted slice. Lookups compare full variable-length strings (`1...`, `3...`, `bc1q...`, `bc1p...`) even though each format has a fixed-size internal identifier.

## Goal

At load time, decode each address into its raw hash/program bytes and store in type-specific sorted slices.

## Scope (this plan only)

- Define `FundedSets` struct with four sorted byte slices
- Decode addresses from TSV into correct bucket
- Keep existing string-based hot loop temporarily (or pair with `high-01`)

**Out of scope:** cache file, bloom filter, faster parsing.

## Implementation steps

### Step 1 — Define funded set types

Create `funded_index.go`:

```go
type FundedSets struct {
    Legacy    [][20]byte // 1... P2PKH pubkey hashes
    P2SH      [][20]byte // 3... script hashes
    SegwitV0  [][20]byte // bc1q... 20-byte witness programs
    TaprootV1 [][32]byte // bc1p... 32-byte witness programs
}
```

### Step 2 — Address decode helpers

Create `bitcoin/decode.go`:

```go
func DecodeFundedAddress(addr string) (kind AddressKind, hash []byte, err error)
```

Use `btcutil.DecodeAddress` or manual prefix dispatch:

| Prefix | Kind | Extract |
|--------|------|---------|
| `1` | Legacy | 20-byte pubkey hash from base58check |
| `3` | P2SH | 20-byte script hash |
| `bc1q` | SegwitV0 | 20-byte witness program |
| `bc1p` | TaprootV1 | 32-byte witness program |

### Step 3 — Load into partitions

In `loadFunded()`, replace `[]string` append with bucket append + sort each slice at end.

### Step 4 — Logging

Print per-bucket counts:

```
Loaded funded: legacy=15M p2sh=10M segwit=5M taproot=700K
```

### Step 5 — Bridge to existing lookup (temporary)

Until `high-01` lands, optionally keep `[]string` in parallel for A/B testing, or implement byte lookup immediately.

## Files to touch

| File | Action |
|------|--------|
| `funded_index.go` | New — `FundedSets` type |
| `bitcoin/decode.go` | New — address decoding |
| `cryptogen.go` | Load into partitions, update `loadFunded` return type |

## Verification

```bash
go build -o btcfind .
./btcfind 1000 4
```

- [x] Sum of bucket counts = previous total loaded count (31,733,986)
- [x] Sample addresses decode to correct buckets (legacy/p2sh/segwit/taproot/other)
- [x] Sort invariant: each slice sorted via `FundedSets.Sort()`

## Rollback

Revert `loadFunded()` to return `[]string`. Partition code is additive and can be deleted independently.