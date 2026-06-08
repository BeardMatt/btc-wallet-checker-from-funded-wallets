# Plan: Lazy per-format derivation (taproot last)

| Field | Value |
|-------|-------|
| **ID** | `low-09` |
| **Priority** | **High** |
| **Status** | `pending` |
| **Depends on** | none (synergistic with `low-10`) |
| **Estimated gain** | **15–35%** hot-loop throughput (taproot EC skipped on most keys) |

## Problem

`DeriveLookupKeys` computes **all** address formats up front for every key. Profiling (`low-02`) showed taproot derivation (`ComputeTaprootKeyNoScript`) consumes ~55% of CPU, while bloom/lookup is &lt;1%.

A prior lazy-taproot spike **regressed ~2.6×** because taproot work moved to the single consumer goroutine. That was an architecture mistake, not proof that lazy derivation cannot work.

## Goal

Derive formats **incrementally inside worker goroutines**, cheapest first, running bloom checks after each step. Skip remaining derivations when impossible to match (all prior blooms definitively miss — or when `--formats` limits scope per `low-10`).

Recommended derivation order (expensive last):

1. Compressed pubkey → `CompressedHash` → legacy compressed + segwit bloom
2. Uncompressed pubkey → `UncompressedHash` → legacy uncompressed bloom
3. P2SH script hash → `P2SHHash` → p2sh bloom
4. Taproot key → `TaprootKey` → taproot bloom (**only if still needed**)

## Scope (this plan only)

- Refactor `GenKeypair` / `DeriveLookupKeys` into staged derivation API
- `matchFundedPartial(sets, keys, stages)` or early-exit checks after each stage
- Keep derivation **in worker batch loop** — never on main consumer
- Preserve correctness: a hit on any format must still be found

**Out of scope:** GPU, changing funded index layout.

## Implementation steps

### Step 1 — Staged lookup keys type

```go
type LookupKeys struct { /* existing fields */ }
type DeriveStage int
func DeriveThrough(keys *LookupKeys, priv *btcec.PrivateKey, stage DeriveStage) 
func MatchAfterStage(sets FundedSets, keys LookupKeys, stage DeriveStage) (MatchKind, bool)
```

### Step 2 — Worker integration

In `newWallet` batch loop, replace `GenKeypair()` with staged path:

```go
priv := newPrivKey()
var keys LookupKeys
for stage := StageCompressed; stage <= maxStage; stage++ {
    DeriveThrough(&keys, priv, stage)
    if kind, ok := MatchAfterStage(sets, keys, stage); ok {
        // full wallet for channel
    }
    if canSkipRemainingStages(sets, keys, stage) {
        break
    }
}
```

`canSkipRemainingStages`: if all blooms for remaining formats would reject (optional aggressive skip) OR formats flag excludes them.

### Step 3 — Benchmark and profile

```bash
go test -bench=BenchmarkHotLoopMatch -cpuprofile=cpu.prof
./scripts/run-benchmark.sh
```

Target: measurable drop in taproot CPU samples; keys/sec &gt; baseline without regression on hit correctness.

## Files to touch

| File | Action |
|------|--------|
| `bitcoin/lookup.go` | Staged derivation API |
| `bitcoin/wallets.go` | `GenKeypair` → staged path |
| `cryptogen.go` | Pass `FundedSets` into workers OR partial-match helper |
| `hotloop_test.go` | Benchmark before/after |

## Verification

- [ ] `./btcfind 50000 8` keys/sec improves vs ~56k baseline
- [ ] `--simulate-hit` and `--inject-index-hit` still pass
- [ ] No regression &gt;2% without profile justification
- [ ] Taproot share of CPU time drops in profile

## Rollback

Restore eager `DeriveLookupKeys` in `GenKeypair`.