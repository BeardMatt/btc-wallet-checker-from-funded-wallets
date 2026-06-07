# Plan: Simulate-hit flag for testing

| Field | Value |
|-------|-------|
| **ID** | `low-06` |
| **Priority** | Low |
| **Status** | `maybe` |
| **Depends on** | none |
| **Estimated gain** | Correctness — exercises hit path without waiting for a real match |

## Problem

A real hit is effectively impossible to observe in normal runs. The hit branch (WIF encoding, `EncodeMatchAddress`, stdout output) is rarely executed, so regressions there are easy to miss.

## Goal

Add a CLI flag that forces one simulated hit during a run so the full hit path can be tested on demand.

## Scope (this plan only)

- `--simulate-hit` flag (or `-simulate-hit`)
- Optional `--simulate-hit-at N` (default `1`) — trigger on the Nth key processed
- Print the same output as a real hit: `WIF : address`
- Log clearly that the hit was simulated

**Out of scope:** finding a real funded private key, modifying the funded index, benchmark changes.

## Implementation steps

### Step 1 — Parse flags

Extend CLI beyond positional args. Options:

```
./btcfind 1000 --simulate-hit
./btcfind 1000 --simulate-hit --simulate-hit-at 500
```

Keep positional `<num_keys> [threads]` working for existing usage and benchmarks.

### Step 2 — Force hit branch

In the hot loop, when `processed+1 == simulateHitAt`:

```go
if simulateHit {
    kind := pickMatchKind(wallet.Keys) // first matching format for display, or fixed e.g. MatchLegacyCompressed
    // same WIF + EncodeMatchAddress path as real hit
    fmt.Println("(simulated hit)", wif, " : ", addr)
}
```

Use the **current wallet's** keys for encoding — tests WIF + address output with real derived hashes, without requiring the key to be in `funded.tsv`.

### Step 3 — Optional lookup self-test

Add `--simulate-hit-verify-lookup` (optional stretch): on simulate, also check whether any of the wallet's hashes appear in the funded sets and log `lookup would match: true/false`. Separates **output encoding** from **index membership**.

### Step 4 — Document

- `AGENTS.md` — flag usage
- Note: simulated hit does not prove a random key matched a funded address; it proves the hit **code path** runs

## Files to touch

| File | Action |
|------|--------|
| `cryptogen.go` | Flag parsing, simulated hit in hot loop |
| `AGENTS.md` | Document flags |

## Verification

```bash
go build -o btcfind .
./btcfind 100 --simulate-hit
```

- [ ] Output includes `(simulated hit)` prefix and valid-looking WIF + address
- [ ] No simulate output when flag omitted
- [ ] Standard benchmark (`./btcfind 50000 8`) unchanged when flags not used

## Rollback

Remove flag parsing and simulated-hit branch; restore positional-only CLI.