# Plan: Verbosity for funded.tsv check at startup

| Field | Value |
|-------|-------|
| **ID** | `low-16` |
| **Priority** | **Low** |
| **Status** | `completed` |
| **Depends on** | `low-07` |
| **Estimated gain** | UX — visible funded-data freshness check; no throughput change |

## Problem

`ensureFunded()` in `funded_download.go` is **silent on the happy path**. When `funded.tsv` exists and is up to date, the user sees nothing — the next output is `▸ Loading funded wallets` from `loadFunded()`.

There is no feedback that:

- The remote update check ran (HEAD request)
- Local file size and modification time
- Remote `Last-Modified` and whether local is current

Output only appears when the file is missing, the remote check fails, or an update is available.

## Goal

On every run (unless `--quiet`), show a clear **Funded data check** section before load/download, including the common “file OK, no update needed” case.

## Scope (this plan only)

### Default output (not `--quiet`)

```
▸ Funded data check
  Checking for updates…
  funded.tsv  2.7 GB  modified 2026-06-01T12:00:00Z
  remote      2026-06-01T12:00:00Z
  Up to date — using local file
```

### While checking

- `ProgressIndeterminate("Checking for updates…")` during HEAD request (reuse `low-07` UI)
- Clear spinner line when check completes

### Per-outcome messages

| Outcome | Message |
|---------|---------|
| File missing | Existing download section (unchanged) |
| Remote HEAD fails | Existing warning + `using local file` + local mtime/size |
| Remote newer | Existing out-of-date prompt (unchanged) |
| **Up to date** | **New:** `Up to date — using local file` |
| User declines update | `Using existing funded.tsv` (unchanged) |
| Download completes | Existing progress + saved size (unchanged) |

### `--verbose`

Additional lines:

- Remote URL
- HEAD latency (ms)
- Local path (absolute or cwd-relative)
- Cache invalidation note if download will remove `funded.cache`

### `--quiet`

Skip entire funded.tsv check section (current behavior for warnings/hits only). Still download if file missing (or warn? document: missing file always downloads with minimal one-line log).

## Implementation steps

### Step 1 — Refactor `ensureFunded()`

```go
func ensureFunded() {
    u := ui()
    if !u.quiet {
        u.Section("Funded data check")
    }
    // stat local → log size/mtime
    // ProgressIndeterminate during remoteLastModified()
    // log comparison result
}
```

Helper: `logLocalFundedInfo(info os.FileInfo)`

### Step 2 — Wire verbosity flags

Reuse existing `UI.verbose` and `UI.quiet` from `low-07` — no new flags required unless `--skip-update-check` is desired later (out of scope).

### Step 3 — Document in `AGENTS.md`

Note startup section order: Banner → Workers → **Funded data check** → Loading funded wallets → Search.

## Files to touch

| File | Action |
|------|--------|
| `funded_download.go` | Verbose check logging, spinner |
| `AGENTS.md` | Startup output description |

## Verification

```bash
go build -o btcfind .
./btcfind 1000 8                    # shows "Up to date" when current
./btcfind 1000 8 --verbose        # extra URL/latency lines
./btcfind 1000 8 --quiet          # no check section
./btcfind 1000 8 --no-color       # same text, no spinner ANSI
```

- [x] Happy path prints local + remote dates and "Up to date"
- [x] HEAD failure still warns and continues
- [x] `--quiet` suppresses check section
- [x] `--verbose` adds URL and timing
- [x] Standard benchmark unchanged

## Rollback

Restore silent happy path in `ensureFunded()`; remove check section.