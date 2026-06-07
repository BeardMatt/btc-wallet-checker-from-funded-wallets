# Plan: Persist discovered wallets to wallets.txt

| Field | Value |
|-------|-------|
| **ID** | `low-08` |
| **Priority** | Low |
| **Status** | `completed` |
| **Depends on** | `low-07` |
| **Estimated gain** | Reliability — survives crashes; no hot-loop throughput change |

## Problem

When a funded wallet is discovered, the only record is terminal output (stderr banner from `low-07`). If the process crashes, the terminal scrollback is lost, or the session is interrupted before the user copies the WIF, the discovery is gone.

`low-07` explicitly left file persistence out of scope.

## Goal

On every **real** wallet hit, append a durable log entry to `wallets.txt` in the project root (same directory the binary is run from). Multiple hits in one run or across runs accumulate in the same file.

## Scope (this plan only)

### File location and mode

| Setting | Value |
|---------|-------|
| Path | `wallets.txt` (cwd at runtime, typically repo root) |
| Mode | Append (`O_APPEND\|O_CREATE\|O_WRONLY`) |
| Permissions | `0600` on create (owner read/write only) |

Create the file on first hit; do not truncate existing content.

### When to write

| Event | Write to `wallets.txt`? |
|-------|-------------------------|
| Real funded match (`matchFunded` hit) | Yes |
| `--simulate-hit` (no real match) | No |
| `--inject-index-hit` with real lookup match | Yes (real index entry matched) |

Write **immediately** when the hit is detected — before or in the same step as the stderr banner — so a crash right after discovery still leaves a file record.

### Record format

One discovery per append block. Plain text, easy to grep and hand-edit. Include enough context to recover funds without re-running.

Example:

```
--- wallet found 2026-06-07T19:47:55-04:00 ---
key_index: 42891
format: segwit_v0 (Native SegWit bc1q)
address: bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh
wif: L3bYT3MKumyB6hhCSAh1C51db3a1cERszJKcC2uGn3MwuE8sYPSV

```

- Timestamp: local time, RFC3339
- `key_index`: 1-based index from the hot loop (same as banner)
- `format`: machine id + human label (reuse `kindLabels` from `output.go`)
- `address` / `wif`: full values, never truncated

Separate blocks with a blank line.

### Error handling

- If append fails (disk full, permission denied): log warning to stderr, **continue searching** — do not panic
- Optionally mention `wallets.txt` path in the hit banner: `Saved to wallets.txt`

### Security note (document only)

`wallets.txt` contains live private keys. Document in `AGENTS.md`:

- Add to `.gitignore` if not already present
- Never commit or share the file
- Treat like a password vault

## Implementation steps

### Step 1 — `appendWalletHit` helper

New file `wallet_log.go` (or add to `output.go`):

```go
const walletsLogFile = "wallets.txt"

func appendWalletHit(keyIndex int, kind bitcoin.MatchKind, address, wif string) error
```

Use `os.OpenFile` with append + create, `chmod 0600` after create, `fmt.Fprintf` the block, `Sync()` before close (best-effort durability on crash).

### Step 2 — Call from hit path

In `cryptogen.go` hot loop, on real hit (`ok == true`):

```go
if err := appendWalletHit(keyIndex, kind, addr, wif); err != nil {
    appUI.Warnf("could not write %s: %v\n", walletsLogFile, err)
}
```

Reuse WIF/address already computed in `PrintHit` — extract shared `hitDetails(...)` or call append from inside `PrintHit` with a `simulated bool` guard.

### Step 3 — `.gitignore` and docs

- Add `wallets.txt` to `.gitignore`
- `AGENTS.md`: note auto-append behavior and security

### Step 4 — Tests

```go
func TestAppendWalletHit(t *testing.T) {
    dir := t.TempDir()
    // chdir or pass path override for test
    // append twice, assert file has two blocks, mode 0600
}
```

Use a test hook or `walletsLogFile` override via internal var for temp-dir tests.

`--simulate-hit` must not create or modify `wallets.txt`.

## Files to touch

| File | Action |
|------|--------|
| `wallet_log.go` (new) | Append helper, format block |
| `output.go` or `cryptogen.go` | Invoke on real hit |
| `.gitignore` | Ignore `wallets.txt` |
| `AGENTS.md` | Document persistence + security |
| `wallet_log_test.go` (new) | Append and simulate-hit exclusion |

## Verification

```bash
go build -o btcfind .
rm -f wallets.txt
./btcfind 1 --simulate-hit          # wallets.txt must not exist
./btcfind 1 --inject-index-hit legacy --simulate-hit-verify-lookup  # wallets.txt created with one entry
cat wallets.txt
./scripts/run-benchmark.sh           # no file created; throughput unchanged
```

- [x] Real hit appends to `wallets.txt` with full WIF and address
- [x] Second hit in same run appends second block
- [x] `--simulate-hit` does not write the file
- [x] Write failure warns but does not stop the run
- [x] `wallets.txt` is gitignored
- [x] Standard benchmark unchanged

## Rollback

Remove `appendWalletHit` and call sites; delete `wallet_log.go`; revert `.gitignore`.