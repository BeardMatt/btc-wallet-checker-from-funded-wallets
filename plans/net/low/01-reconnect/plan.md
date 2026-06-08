# Plan: Reconnect + coordinator restart

| Field | Value |
|-------|-------|
| **ID** | `net-low-01` |
| **Priority** | Low |
| **Status** | `pending` |
| **Depends on** | `net-medium-01`, `net-medium-02` |
| **Estimated gain** | Resilience — workers survive transient failures |

## Problem

Network blips or coordinator restarts currently drop workers permanently. Long-running clusters need automatic re-registration without manual intervention.

## Goal

Workers reconnect with exponential backoff; coordinator accepts re-registration with same or new `worker_id`. Optional: coordinator restart returns to `LOBBY` with workers waiting for Enter again.

## Scope (this plan only)

- Worker reconnect loop on register/heartbeat/run/stats/hit failures
- Coordinator: prune stale workers; accept re-register
- Preserve local cache (no re-download on 304)
- Document behavior: search run does not resume keyspace (probabilistic search)

**Out of scope:** coordinator HA / leader election, persistent run state across coordinator restart.

## Implementation steps

### Step 1 — Worker connection state machine

```
connected → degraded → reconnecting → connected
```

Backoff: 1s, 2s, 4s, … cap 60s. Reset on success.

### Step 2 — Re-register on coordinator restart

If `worker_id` unknown (coordinator cold start):

```
POST /register  → new worker_id
GET /run        → lobby
```

Worker returns to heartbeat `waiting`; operator must press Enter again.

### Step 3 — Mid-run disconnect

If coordinator dies during `RUNNING`:

- Worker stops search, enters reconnect
- On reconnect in `LOBBY`: wait for Enter
- On reconnect still `RUNNING` (coordinator survived): resume search + stats

### Step 4 — Stale worker cleanup

Coordinator removes workers with no heartbeat for 30s; stats aggregate excludes them.

### Step 5 — Tests

`httptest` coordinator killed mid-run; worker reconnects within 60s cap.

## Files to touch

| File | Action |
|------|--------|
| `worker/client.go` | Retry + backoff wrapper |
| `worker/run.go` | Reconnect integration |
| `coordinator/lobby.go` | Stale pruning |

## Verification

- [ ] Kill coordinator during run → workers backoff, re-register on restart
- [ ] Network drop 10s → worker recovers without manual restart
- [ ] Cache not re-downloaded when ETag unchanged
- [ ] No duplicate `wallets.txt` lines from reconnect retries (idempotent hit handling)

## Rollback

Single-shot connection only (revert to `net-high-03` behavior).