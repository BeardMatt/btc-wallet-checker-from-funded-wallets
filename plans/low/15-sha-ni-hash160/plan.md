# Plan: SHA-NI accelerated Hash160

| Field | Value |
|-------|-------|
| **ID** | `low-15` |
| **Priority** | **Low** |
| **Status** | `pending` |
| **Depends on** | none (best after `low-09`/`low-10` while hash path is touched) |
| **Estimated gain** | **2–8%** if SHA-NI available; **0%** fallback on older CPUs |

## Problem

`btcutil.Hash160` (SHA256 → RIPEMD160) runs for every key on compressed and uncompressed pubkeys (~3% of CPU in `low-02` profile). On CPUs with **SHA-NI** extensions, hardware SHA256 can beat generic software implementations.

## Goal

Use SHA-NI for the SHA256 step of Hash160 when `cpu.X86.HasSHA` is true; fall back to existing path otherwise.

## Scope (this plan only)

- Build tag or runtime CPU feature detect via `golang.org/x/sys/cpu`
- Assembly or `github.com/minio/sha256-simd` / hand-rolled SHA-NI block — prefer minimal dep
- `hash160Fast(pubkey []byte) [20]byte` in `bitcoin/` used by `DeriveLookupKeys`
- RIPEMD160 remains software (no common HW accel)

**Out of scope:** GPU hashing, changing Hash160 algorithm.

## Implementation steps

### Step 1 — Feature detect

```go
var useSHANI = cpu.X86.HasSHA && cpu.X86.HasSSE41
```

### Step 2 — Fast path for pubkey hashing

Replace or wrap `btcutil.Hash160` calls in `lookup.go`.

### Step 3 — Cross-validate

Test vectors: fast path output == `btcutil.Hash160` for 10k random pubkeys.

### Step 4 — Benchmark

```bash
go test -bench=BenchmarkDeriveLookupKeys ./bitcoin/
./scripts/run-benchmark.sh
```

## Files to touch

| File | Action |
|------|--------|
| `bitcoin/hash160.go` (new) | SHA-NI + fallback |
| `bitcoin/lookup.go` | Call fast Hash160 |
| `bitcoin/hash160_test.go` (new) | Vector parity |

## Verification

- [ ] Output bit-identical to btcd Hash160 on test vectors
- [ ] Measurable bench improvement on SHA-NI host
- [ ] No regression when SHA-NI absent

## Rollback

Remove fast path; use `btcutil.Hash160` only.