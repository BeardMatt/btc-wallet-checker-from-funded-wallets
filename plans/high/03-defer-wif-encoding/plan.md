# Plan: Defer WIF encoding until hit

| Field | Value |
|-------|-------|
| **ID** | `high-03` |
| **Priority** | High |
| **Status** | `pending` |
| **Depends on** | none |
| **Estimated gain** | 10–20% hot-loop throughput |

## Problem

`btcutil.NewWIF(...).String()` runs on every generated key. WIF is a base58-check string only needed when printing a match — vanishingly rare.

## Goal

Keep the raw 32-byte private key in the hot path. Encode WIF only inside the hit branch.

## Scope (this plan only)

- Change `Wallet` to hold raw privkey bytes instead of WIF string
- Move WIF encoding to hit handler

**Out of scope:** hash-based lookup, address encoding changes.

## Implementation steps

### Step 1 — Change Wallet struct

```go
type Wallet struct {
    PrivKey   []byte      // 32-byte serialized private key
    Addresses []string    // unchanged for now (or Keys in high-01)
}
```

### Step 2 — Remove WIF from GenKeypair

In `bitcoin/wallets.go`, replace:

```go
wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
// ...
Privkey: wif.String(),
```

With:

```go
PrivKey: privKey.Serialize(),
```

### Step 3 — Encode WIF on hit only

In `cryptogen.go` hit branch:

```go
wif, _ := btcutil.NewWIFFromBytes(wallet.PrivKey, &chaincfg.MainNetParams, true)
fmt.Println(wif.String(), " : ", matchedAddr)
```

### Step 4 — Add helper (optional)

```go
func PrivKeyToWIF(key []byte) string { ... }
```

## Files to touch

| File | Action |
|------|--------|
| `bitcoin/wallets.go` | Store raw key bytes |
| `cryptogen.go` | WIF encode on hit |

## Verification

```bash
go build -o btcfind .
./btcfind 50000 8
```

- [ ] Throughput improves vs baseline
- [ ] Program still builds and runs
- [ ] Hit output (if testable) shows valid WIF

## Rollback

Restore `Privkey string` and `wif.String()` in `GenKeypair`. One-file revert.