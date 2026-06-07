# Plan: Hand-rolled address encoding

| Field | Value |
|-------|-------|
| **ID** | `low-01` |
| **Priority** | Low |
| **Status** | `completed` |
| **Depends on** | `high-01` |
| **Estimated gain** | Marginal after hash-based lookup |

## Problem

After `high-01`, address encoding only runs on hits. btcd's `EncodeAddress` on the hit path is no longer a bottleneck for normal operation.

## Goal

Replace btcd encoding on the hit path with minimal in-tree base58check and bech32/bech32m encoders — only if profiling shows hit-path or test-path encoding still matters.

## Scope (this plan only)

- Inline base58check for P2PKH and P2SH display
- Inline bech32 (v0) and bech32m (v1) for display on hit

**Out of scope:** hot-loop encoding (already removed in `high-01`).

## Implementation steps

### Step 1 — Profile hit path first

Confirm this plan is worth doing:

```bash
go test -bench=. ./bitcoin/...
```

If hit-path encoding is <1% of runtime, mark plan `skipped`.

### Step 2 — Port base58check

Reuse or restore pre-btcd `base58Encode` / `base58CheckEncode` from git history. Add version bytes:

- `0x00` P2PKH
- `0x05` P2SH

### Step 3 — Add bech32 encoder

Implement BIP-173 for `bc1q` display and BIP-350 bech32m for `bc1p` display. Reference: btcd's `btcutil/bech32` (read, don't necessarily import).

### Step 4 — Hit-path only

```go
func EncodeMatchedAddress(kind AddressKind, hash []byte) string
```

### Step 5 — Cross-validate

Generate 10,000 random keys; compare hand-rolled vs btcd output for all five formats. Must match exactly.

## Files to touch

| File | Action |
|------|--------|
| `bitcoin/encode.go` | Hand-rolled encoders |
| `bitcoin/encode_test.go` | Cross-validation tests |

## Verification

- [x] 100% match vs btcd on 10,000 random keys (all five formats)
- [x] Encode path uses base58/bech32 primitives only (no Address types)

## Rollback

Keep btcd `EncodeAddress` on hit path. Delete hand-rolled encoders.