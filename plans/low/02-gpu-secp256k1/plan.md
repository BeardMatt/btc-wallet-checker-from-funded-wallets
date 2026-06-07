# Plan: GPU / assembly secp256k1

| Field | Value |
|-------|-------|
| **ID** | `low-02` |
| **Priority** | Low |
| **Status** | `pending` |
| **Depends on** | `high-01`, `medium-02` |
| **Estimated gain** | Uncertain; keygen likely not bottleneck after prior plans |

## Problem

Private key generation and pubkey derivation use CPU-bound secp256k1 math. After hash-based lookup and bloom filters, key generation may become a larger fraction of total time.

## Goal

Evaluate whether offloading batch key generation to GPU or SIMD-optimized assembly libraries improves end-to-end throughput.

## Scope (this plan only)

- Benchmark to determine if keygen is the new bottleneck
- Spike integration with one fast library (e.g. `libsecp256k1` via cgo, or batch EC mult)

**Out of scope:** distributed search, lookup changes.

## Implementation steps

### Step 1 — Profile after high/medium plans

```bash
go build -o btcfind .
go tool pprof -cpuprofile=cpu.prof ./btcfind 200000 8
go tool pprof -top cpu.prof
```

If `NewPrivateKey` / `PubKey` < 30% of CPU, mark plan `skipped`.

### Step 2 — Evaluate options

| Option | Pros | Cons |
|--------|------|------|
| `libsecp256k1` via cgo | Battle-tested | CGO, build complexity |
| Batch keygen on CPU | No GPU driver deps | Limited speedup |
| GPU (CUDA/OpenCL) | Massive parallelism | Hard to integrate, no SHA/RIPEMD on GPU |

### Step 3 — Spike batch CPU keygen

Generate N keys per call using a faster path before considering GPU:

```go
func GenKeypairsBatch(n int) []Wallet
```

### Step 4 — Measure

Compare `./btcfind 500000 8` before and after spike.

### Step 5 — Decision gate

Only proceed to GPU if batch CPU keygen shows keygen > 50% of profile and batching insufficient.

## Files to touch

| File | Action |
|------|--------|
| `bitcoin/wallets.go` | Batch generation API |
| `bitcoin/secp256k1_fast.go` | Optional cgo wrapper |

## Verification

- [ ] Profile proves keygen is bottleneck OR plan marked `skipped`
- [ ] If implemented: measurable throughput gain on benchmark command

## Rollback

Remove cgo / GPU code path. Keep standard `btcec.NewPrivateKey()`.