# Plan: Coordinator live dashboard

| Field | Value |
|-------|-------|
| **ID** | `net-low-02` |
| **Priority** | Low |
| **Status** | `pending` |
| **Depends on** | `net-medium-01` |
| **Estimated gain** | UX — at-a-glance cluster health |

## Problem

Minimal stats line from `net-medium-01` lacks per-worker detail, connection health, and run timing needed for multi-machine ops.

## Goal

Full-screen TTY dashboard on coordinator (when not `--quiet`): worker table, aggregate + per-worker keys/sec, keys tried, hits, run duration, lobby hint before start.

## Scope (this plan only)

- Refresh dashboard every 1s on stderr (reuse standalone progress patterns)
- Columns: hostname, threads, state, keys/s, run keys, last heartbeat
- Footer: cluster totals + `Press Ctrl+C to stop`
- `--no-color` / `--quiet` respected

**Out of scope:** web UI, Prometheus, log shipping.

## Implementation steps

### Step 1 — Dashboard model

```go
type Dashboard struct {
    RunState    string
    Workers     []WorkerRow
    ClusterKPS  float64
    TotalKeys   uint64
    Hits        uint64
    StartedAt   time.Time
}
```

### Step 2 — Render loop

Goroutine ticks every 1s; uses ANSI clear-line / cursor-up like standalone `BeginSearch` progress.

Example:

```
┌─ btcfind cluster ─────────────────────────────────────┐
│ State: RUNNING   Duration: 12m34s   Hits: 0           │
│ Cluster: 412,000 keys/s   Total keys: 1.24B           │
├──────────┬────────┬────────┬──────────┬─────────────┤
│ Host     │ Threads│ State  │ Keys/s   │ Run keys    │
│ host-a   │ 16     │ running│ 138,200  │ 412,000,000 │
│ host-b   │ 8      │ running│  68,100  │ 198,000,000 │
└──────────┴────────┴────────┴──────────┴─────────────┘
```

### Step 3 — Lobby view

Before Enter:

```
Workers connected: 3 (26 threads total)
Press Enter to start search...
```

### Step 4 — Hit overlay

On hit, pause dashboard refresh briefly; print banner; resume.

### Step 5 — Non-TTY fallback

Plain log lines every 10s (no cursor control).

## Files to touch

| File | Action |
|------|--------|
| `coordinator/dashboard.go` | Model + render |
| `coordinator/tty.go` | Integrate with Enter gate |
| `output.go` | Share ANSI helpers if useful |

## Verification

- [ ] Dashboard updates live during 2-worker run
- [ ] Stale worker shown as `offline` after heartbeat timeout
- [ ] `--quiet` suppresses dashboard
- [ ] Ctrl+C clears screen and prints final summary

## Rollback

Revert to single-line stats from `net-medium-01`.