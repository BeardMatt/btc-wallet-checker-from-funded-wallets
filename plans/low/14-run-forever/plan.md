# Plan: Run-forever mode and session checkpoints

| Field | Value |
|-------|-------|
| **ID** | `low-14` |
| **Priority** | **Low** |
| **Status** | `completed` |
| **Depends on** | `low-08` (`wallets.txt` persistence pattern) |
| **Estimated gain** | Ops / UX — long-running production use |

## Problem

Runs require an upfront `num_keys` cap. Long searches need manual restarts, mental tracking of total keys tried, and lose aggregate stats across sessions.

## Goal

Support continuous search with periodic checkpointing and resumable session stats (not keyspace replay — search remains probabilistic).

## Scope (this plan only)

### CLI

```bash
./btcfind --forever [threads] [flags]
./btcfind --forever --checkpoint-interval 60s
```

- No `num_keys` required in forever mode
- Ctrl+C graceful shutdown: flush checkpoint, preserve `wallets.txt`

### Checkpoint file (`session.json` or `btcfind.session`)

```json
{
  "started_at": "2026-06-07T20:00:00Z",
  "last_checkpoint": "2026-06-07T21:00:00Z",
  "total_keys_tried": 1234567890,
  "total_hits": 0,
  "best_keys_per_second": 56335.83
}
```

Append-only or atomic replace on interval. Display in live progress line.

### Optional

- Rotate `wallets.txt` → `wallets-YYYYMMDD.txt` at midnight
- `--max-keys N` as safety cap even in forever mode

**Out of scope:** deterministic keyspace partitioning (see `low-03`).

## Implementation steps

### Step 1 — CLI mode mutual exclusion

`--forever` vs positional `num_keys`.

### Step 2 — Hot loop until signal

Listen for `SIGINT`/`SIGTERM`; break loop, write checkpoint.

### Step 3 — Periodic checkpoint goroutine

### Step 4 — UI stats line

`Keys tried: 1,234,567,890 (56.3k/s avg) | Hits: 0`

## Files to touch

| File | Action |
|------|--------|
| `cli.go` | `--forever`, `--checkpoint-interval` |
| `cryptogen.go` | Unbounded loop + signal handling |
| `session.go` (new) | Checkpoint read/write |
| `output.go` | Extended progress stats |
| `AGENTS.md` | Forever mode docs |

## Verification

- [x] `--forever` runs until SIGINT
- [x] Checkpoint file updates on interval
- [x] `wallets.txt` still appends on hit
- [x] `./btcfind 50000 8` unchanged

## Rollback

Remove forever mode and session file logic.