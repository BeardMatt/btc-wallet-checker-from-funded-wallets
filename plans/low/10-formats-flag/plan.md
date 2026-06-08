# Plan: `--formats` flag to skip address types

| Field | Value |
|-------|-------|
| **ID** | `low-10` |
| **Priority** | **Medium** |
| **Status** | `completed` |
| **Depends on** | none |
| **Estimated gain** | **0–50%** depending on formats omitted (skip taproot ≈ largest win) |

## Problem

Every key derives and checks all five match paths (legacy compressed/uncompressed, segwit, p2sh, taproot) even when the user only cares about certain address eras or wants maximum speed over full coverage.

## Goal

Add `--formats` CLI flag to limit which address types are derived and checked.

## Scope (this plan only)

### Flag syntax

```bash
./btcfind 50000 8 --formats legacy,segwit
./btcfind 50000 --formats taproot          # taproot-only search
./btcfind 50000                            # default: all formats (current behavior)
```

Allowed tokens: `legacy`, `legacy-compressed`, `legacy-uncompressed`, `segwit`, `p2sh`, `taproot`, or alias `all`.

| Token | Skips when omitted |
|-------|---------------------|
| `legacy` / compressed+uncompressed | Both legacy hash checks |
| `segwit` | Segwit v0 check (shares compressed hash) |
| `p2sh` | P2SH derivation + check |
| `taproot` | Taproot derivation + check |

### Behavior

- Invalid token → error at startup
- Empty or `all` → current behavior
- Document tradeoff: **omitted formats can never produce hits**

**Out of scope:** separate funded indexes per format (see `low-11`).

## Implementation steps

### Step 1 — Parse flag in `cli.go`

```go
type formatMask struct {
    legacyCompressed, legacyUncompressed, segwit, p2sh, taproot bool
}
```

### Step 2 — Thread mask through hot loop

- `DeriveLookupKeys` / staged derivation (`low-09`) respects mask
- `matchFunded` skips disabled kinds

### Step 3 — Document in `AGENTS.md`

## Files to touch

| File | Action |
|------|--------|
| `cli.go` | Parse `--formats` |
| `bitcoin/lookup.go` | Conditional derivation |
| `funded_index.go` | `matchFunded` mask |
| `cryptogen.go` | Pass mask to workers |
| `AGENTS.md` | Flag docs |

## Verification

- [x] Default (no flag) matches current benchmark
- [x] `--formats taproot` reduces CPU time for taproot in profile
- [x] `--formats legacy` still finds inject-index-hit legacy

## Rollback

Remove flag; restore unconditional derivation.