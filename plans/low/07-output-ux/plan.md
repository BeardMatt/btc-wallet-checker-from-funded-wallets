# Plan: Output UX — pretty terminal, progress, and wallet recovery

| Field | Value |
|-------|-------|
| **ID** | `low-07` |
| **Priority** | Low |
| **Status** | `completed` |
| **Depends on** | `low-06` |
| **Estimated gain** | UX only — no hot-loop throughput change expected |

## Problem

All user-facing output is plain `fmt.Print` lines scattered across `cryptogen.go`, `funded_download.go`, `funded_index.go`, and `simulate.go`. There is no visual hierarchy, no live progress during long operations, and no guidance when a wallet is found.

Current hit output is a single line:

```
WIF : address
```

(or `(simulated hit) WIF : address`). That is hard to spot in a long run and gives no next steps for recovering funds.

## Goal

Make `btcfind` feel polished in the terminal:

1. **Startup / load** — structured sections, progress where work is slow (download, TSV parse, sort, bloom build)
2. **Hot loop** — clear “searching” state with optional live throughput on TTYs
3. **Summary** — readable completion stats (preserve benchmark-parseable lines)
4. **Wallet discovery** — unmistakable banner with WIF, address, format, and step-by-step recovery instructions

## Scope (this plan only)

### Centralized output layer

Introduce a small `ui` package or `output.go` in `main` with:

| Concern | Behavior |
|---------|----------|
| TTY detection | `golang.org/x/term` or `os.Stdout` stat — colors/progress only when interactive |
| `--no-color` | Disable ANSI even on TTY (CI, logs) |
| `--quiet` | Errors + hit banner only; skip decorative startup (optional stretch) |
| Sections | Consistent labels: `▸ Loading`, `▸ Search`, `▸ Result` |
| Numbers | Keep `golang.org/x/text/message` thousand separators |

No new heavy dependencies unless justified (e.g. `charmbracelet/lipgloss` is optional; ANSI helpers or a tiny internal palette is fine).

### Startup and load aesthetics

Replace ad-hoc prints with sectioned output:

```
btcfind — Bitcoin key search
────────────────────────────
▸ Workers     8
▸ Funded data cache hit (1.2s)
  legacy 16,234,567  p2sh 10,123,456  segwit 4,567,890  taproot 808,073
```

When cache miss or cold parse:

- **TSV parse** — progress bar or spinner from bytes read / file size (`os.Stat` on `funded.tsv`)
- **Sort** — spinner or elapsed timer
- **Bloom build** — compact bucket summary (collapse verbose per-bucket debug into one table unless `--verbose`)

Reuse/improve `copyWithProgress` in `funded_download.go` (bar instead of `... N MB written` every 5s).

### Hot-loop runtime UX

While keys are tested:

- On TTY: single updating line — `Testing 50,000 keys… 12,345 (24.7k/s)` using `\r` and no flood of newlines
- On non-TTY / pipe: keep current one-line prefix `Testing N keys...` then block until done (benchmark-friendly)
- **Must preserve** these exact substrings for `scripts/run-benchmark.sh`:

  ```
  Took %fs... Average %.2f keys per second
  ```

  Either emit them on the final line after clearing the progress line, or document an update to the benchmark script.

### Wallet discovery banner

Replace `printHit` one-liner with a framed block on **stderr** (so stdout stays clean for piping) or clearly separated stdout section:

```
╔══════════════════════════════════════════════════════════════╗
║  WALLET FOUND                                                ║
╠══════════════════════════════════════════════════════════════╣
║  Address   bc1q…                                             ║
║  Format    segwit_v0 (P2WPKH)                                ║
║  WIF       L…                                                ║
║  Key #     42,891                                            ║
╚══════════════════════════════════════════════════════════════╝

Recover funds (do this on an offline or trusted machine):

  1. Install a wallet: Electrum, Sparrow, or Bitcoin Core.
  2. Choose “Import private key” / “Sweep private key” (not “watch-only”).
  3. Paste the WIF above. Use mainnet; do not change the key.
  4. Wait for sync, then send funds to a new address you control.

  Electrum: Wallet → Private keys → Import
  Sparrow:  File → Import wallet → Import private key
  Core:     bitcoin-cli importprivkey "<WIF>" "" false

Security:
  • Anyone with this WIF controls the funds — store offline, never share.
  • Prefer sweeping to a new wallet; the discovered key was public in this output.
```

For `--simulate-hit`, use the same layout with a visible `SIMULATED` badge (yellow/dim) so test output matches production hit UX.

Include `matchKindName` → human label map (`segwit_v0` → `Native SegWit (bc1q)`).

**Out of scope for v1:** writing WIF to a file, QR codes, automatic balance lookup from APIs.

### Flags and AGENTS.md

Document:

| Flag | Purpose |
|------|---------|
| `--no-color` | Plain text for logs and CI |
| `--quiet` | Minimal startup noise (optional) |

## Implementation steps

### Step 1 — Output helpers

Create `output.go` (or `ui/output.go`):

```go
type UI struct {
    color bool
    quiet bool
    out   io.Writer
    err   io.Writer
}

func NewUI(cfg uiConfig) *UI
func (u *UI) Section(title string)
func (u *UI) Infof(format string, args ...any)
func (u *UI) Progress(label string, current, total int64)
func (u *UI) ProgressIndeterminate(label string)
func (u *UI) ClearProgress()
```

Wire `--no-color` / `--quiet` in `cli.go`.

### Step 2 — Migrate startup messages

Replace prints in:

- `cryptogen.go` — workers, simulate/inject notices, load, sort, summary
- `funded_download.go` — download/decompress progress
- `funded_index.go` — bloom build (verbose bucket lines behind `--verbose` or collapsed table)
- `funded_cache.go` — cache warnings

### Step 3 — Hot-loop progress

In `main` hot loop:

- Track `processed` and elapsed; every ~100ms on TTY refresh one line
- On exit, `ClearProgress()` then print final `Took … Average … keys per second` unchanged for benchmarks

### Step 4 — Hit banner and recovery text

Refactor `printHit` in `simulate.go` (or move to `output.go`):

- Accept key index, `MatchKind`, simulated bool
- Render banner + recovery instructions via `UI`
- Update `--simulate-hit` tests in `cli_test.go` to expect new format (or golden substring checks)

### Step 5 — Benchmark compatibility

```bash
./scripts/run-benchmark.sh
```

Confirm `keys_per_second` and `hot_loop_seconds` still parse. If progress uses stderr only, stdout parsing may simplify.

### Step 6 — Document

Update `AGENTS.md` with UX flags and example hit output screenshot/description.

## Files to touch

| File | Action |
|------|--------|
| `output.go` (new) | UI helpers, colors, progress, hit banner |
| `cli.go` | `--no-color`, optional `--quiet`, pass UI config |
| `cryptogen.go` | Use UI for all user messages; hot-loop progress |
| `simulate.go` | Delegate `printHit` to UI |
| `funded_download.go` | Progress bar for download/decompress |
| `funded_index.go` | Compact bloom logging |
| `funded_cache.go` | UI warnings |
| `cli_test.go` | Adjust simulate-hit expectations |
| `AGENTS.md` | Document flags and hit UX |
| `scripts/run-benchmark.sh` | Only if stdout format changes |

## Verification

```bash
go build -o btcfind .
./btcfind 50000 8                    # TTY: progress + summary
./btcfind 50000 8 --no-color | cat   # plain, no escape codes
./btcfind 100 --simulate-hit         # full hit banner + SIMULATED badge
./scripts/run-benchmark.sh           # metrics still parse
```

- [x] Interactive run shows sectioned startup and live key progress
- [x] Non-TTY / `--no-color` has no ANSI artifacts
- [x] Hit banner shows WIF, address, format, key index, recovery steps
- [x] Simulated hit uses same banner with clear SIMULATED marker
- [x] `Average X keys per second` line unchanged for benchmark script
- [x] Standard benchmark throughput unchanged (±1% — UX must not add hot-loop work)

## Rollback

Remove `output.go` and flags; restore direct `fmt.Print` calls and one-line `printHit`.