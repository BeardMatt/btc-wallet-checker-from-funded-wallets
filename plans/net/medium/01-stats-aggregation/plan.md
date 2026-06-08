# Plan: Cluster stats aggregation

| Field | Value |
|-------|-------|
| **ID** | `net-medium-01` |
| **Priority** | Medium |
| **Status** | `pending` |
| **Depends on** | `net-high-03` |
| **Estimated gain** | Visibility — aggregate keys/sec across workers |

## Problem

With multiple workers searching independently, the operator cannot see cluster throughput or total keys tried without per-machine logs.

## Goal

Workers batch stats every 1–5s; coordinator sums deltas into cluster totals and computes aggregate keys/sec. Per-worker rates exposed for imbalance debugging.

## Scope (this plan only)

- `POST /api/v1/stats` endpoint
- Worker-side periodic reporter goroutine
- Coordinator aggregation + in-memory totals
- Basic coordinator TTY line (full dashboard in `net-low-02`)

**Out of scope:** persistent stats history, Prometheus export.

## Implementation steps

### Step 1 — Stats payload

```json
{
  "worker_id": "w-uuid",
  "keys_tried_delta": 125000,
  "keys_per_second": 118400.5,
  "run_keys_tried": 5000000
}
```

Worker tracks `run_keys_tried` locally since last `RUNNING` transition.

### Step 2 — Coordinator aggregation

```go
type ClusterStats struct {
    TotalKeysTried uint64
    AggregateKPS   float64   // sum of worker keys_per_second
    PerWorker      map[string]WorkerStats
    UpdatedAt      time.Time
}
```

Reset totals on new run (`LOBBY` → `RUNNING`).

### Step 3 — Worker reporter

Every 2s (configurable `--stats-interval`):

```go
delta := atomic.SwapUint64(&sinceLastReport, 0)
client.PostStats(ctx, StatsPayload{...})
```

Only report while `running`.

### Step 4 — Coordinator display (minimal)

Update TTY once per second:

```
Cluster: 236,800 keys/s | 10.2M keys tried | Workers: 2/2 reporting
```

### Step 5 — Final stats on stop

On `STOPPING`, workers send one last batch; coordinator prints summary.

## Files to touch

| File | Action |
|------|--------|
| `coordinator/stats.go` | Aggregation |
| `coordinator/handlers.go` | POST /stats |
| `worker/stats.go` | Reporter goroutine |
| `worker/run.go` | Wire reporter lifecycle |

## Verification

- [ ] 2 workers × ~120k/s → cluster ~240k/s (±15%)
- [ ] Stale worker removed from aggregate after heartbeat timeout
- [ ] Stats reset on second Enter-start run
- [ ] Protocol overhead &lt;2% worker CPU (profile spot-check)

## Rollback

Remove stats endpoint and reporter; search still works without aggregation.