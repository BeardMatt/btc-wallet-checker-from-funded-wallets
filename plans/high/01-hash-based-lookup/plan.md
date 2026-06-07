# Plan: Hash-based lookup (no string encode)

| Field | Value |
|-------|-------|
| **ID** | `high-01` |
| **Priority** | High |
| **Status** | `completed` |
| **Depends on** | `medium-01` |
| **Estimated gain** | 2–5× hot-loop throughput |

## Problem

Every generated key encodes five full address strings (base58 + bech32) via btcd, then runs five binary searches over 31M variable-length strings. Encoding and string comparison dominate CPU time.

## Goal

Compare raw fixed-size hashes and witness programs against preprocessed funded sets. Only encode a human-readable address string when a hit occurs.

## Scope (this plan only)

- Change `GenKeypair` / hot loop to produce lookup keys as bytes, not strings
- Change `inFunded` to search byte slices instead of `[]string`
- Encode WIF + display address only on match

**Out of scope:** cache file, bloom filter, TSV parsing changes (separate plans).

## Implementation steps

### Step 1 — Add lookup key struct

Create `bitcoin/lookup.go`:

```go
type LookupKeys struct {
    CompressedHash   [20]byte  // P2PKH compressed, P2WPKH program
    UncompressedHash [20]byte  // P2PKH uncompressed
    P2SHHash         [20]byte  // hash160(0x0014 || compressedHash)
    TaprootKey       [32]byte  // P2TR output key
}
```

### Step 2 — Derive keys without encoding

Replace string `Addresses []string` in `Wallet` with `Keys LookupKeys` and `PrivKey []byte` (32-byte raw key). Compute:

- `CompressedHash` = `btcutil.Hash160(compressed_pubkey)`
- `UncompressedHash` = `btcutil.Hash160(uncompressed_pubkey)`
- `P2SHHash` = `btcutil.Hash160(append([]byte{0x00,0x14}, compressedHash...))`
- `TaprootKey` = `schnorr.SerializePubKey(ComputeTaprootKeyNoScript(pubKey))`

### Step 3 — Byte-slice search helpers

In `cryptogen.go` (or new `funded_index.go`), add:

```go
func inFunded20(set [][20]byte, key [20]byte) bool
func inFunded32(set [][32]byte, key [32]byte) bool
```

Use `sort.Search` with a comparator on fixed-size arrays.

### Step 4 — Update hot loop

```go
wallet := <-ch
if inFunded20(funded.Legacy, wallet.Keys.CompressedHash) ||
   inFunded20(funded.Legacy, wallet.Keys.UncompressedHash) ||
   inFunded20(funded.SegwitV0, wallet.Keys.CompressedHash) ||
   inFunded20(funded.P2SH, wallet.Keys.P2SHHash) ||
   inFunded32(funded.TaprootV1, wallet.Keys.TaprootKey) {
    // encode WIF + matching address string here only
}
```

### Step 5 — Add hit-path encoder

Create `bitcoin/encode.go` with functions that take raw key + which format matched and return display strings. Call only on hit.

## Files to touch

| File | Action |
|------|--------|
| `bitcoin/wallets.go` | Return `LookupKeys` instead of address strings |
| `bitcoin/lookup.go` | New — key derivation |
| `bitcoin/encode.go` | New — encode on hit only |
| `cryptogen.go` | Use partitioned funded sets + byte search |
| `funded_index.go` | New — funded set types (from medium-01) |

## Verification

```bash
go build -o btcfind .
./btcfind 50000 8
```

- [x] Throughput increases vs baseline (+15.7% hot loop vs medium-01)
- [x] No regressions in address format coverage (still checks 1/3/bc1q/bc1p)
- [x] Hit output still prints valid WIF and address (encode on hit only)

## Rollback

Revert to string-based `Addresses []string` and `sort.SearchStrings` if byte lookup introduces bugs. Keep `medium-01` partitioned loader — it is still useful.