# Plan: Default thread count to CPU count

| Field | Value |
|-------|-------|
| **ID** | `low-04` |
| **Priority** | Low |
| **Status** | `completed` |
| **Depends on** | `high-04` |
| **Estimated gain** | Convenience only — sensible default, no tuning |

## Problem

Users must always pass `<threads>` manually. A reasonable default is the machine's CPU count, but today omitting threads or passing `0` falls back to `1` worker.

## Goal

Make `threads` optional. When omitted or `0`, use `runtime.NumCPU()`. Explicit values still override.

No calibration, no benchmarking at startup, no cache file.

## Scope (this plan only)

- Optional `[threads]` CLI argument
- Default to `runtime.NumCPU()` when omitted or `0`
- Log chosen worker count
- Update usage text in `AGENTS.md`

**Out of scope:** calibration, optimal thread detection, file cache.

## Implementation steps

### Step 1 — Make threads argument optional

Change CLI from:

```
btcfind <num_keys> <threads>
```

To:

```
btcfind <num_keys> [threads]
```

Behavior:

| Input | Workers used |
|-------|--------------|
| `./btcfind 50000` | `runtime.NumCPU()` |
| `./btcfind 50000 0` | `runtime.NumCPU()` |
| `./btcfind 50000 8` | `8` |

### Step 2 — Default to NumCPU

In `cryptogen.go`:

```go
import "runtime"

threads := 0
if len(os.Args) >= 3 {
    threads, err = strconv.Atoi(os.Args[2])
    // handle error
}

workers := threads
if workers < 1 {
    workers = runtime.NumCPU()
}

fmt.Printf("Using %d workers\n", workers)
```

### Step 3 — Update docs

- `AGENTS.md` — document optional `[threads]` and default behavior
- `scripts/run-benchmark.sh` — keep explicit `8` for comparable benchmark history (unchanged)

## Files to touch

| File | Action |
|------|--------|
| `cryptogen.go` | Optional threads arg, NumCPU default |
| `AGENTS.md` | Document new CLI |

## Verification

```bash
go build -o btcfind .
./btcfind 10000        # should use NumCPU workers
./btcfind 10000 0      # should use NumCPU workers
./btcfind 10000 4      # should use 4 workers
```

- [ ] Omitting threads uses `runtime.NumCPU()`
- [ ] Explicit override still works
- [ ] Standard benchmark (`./btcfind 50000 8`) unchanged

## Rollback

Restore required `<threads>` argument and `if workers < 1 { workers = 1 }`.