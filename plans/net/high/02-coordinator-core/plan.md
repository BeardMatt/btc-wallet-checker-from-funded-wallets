# Plan: Coordinator core + Enter-to-start

| Field | Value |
|-------|-------|
| **ID** | `net-high-02` |
| **Priority** | High |
| **Status** | `pending` |
| **Depends on** | `net-high-04` |
| **Estimated gain** | Operator-controlled cluster start gate |

## Problem

Workers need a central process that loads funded data, tracks connected clients, and signals when to begin searching. Random key search does not partition keyspace — the coordinator only orchestrates, not assigns work units.

## Goal

Ship `btcfind-coordinator` with lobby state, worker registration, **Enter-to-start** on coordinator TTY, and run state machine (`LOBBY → RUNNING → STOPPING → ENDED`).

## Scope (this plan only)

- `cmd/btcfind-coordinator/main.go`
- Register + heartbeat endpoints
- Long-poll or periodic `GET /api/v1/run` for state changes
- TTY: list connected workers (hostname, threads); block on Enter
- Ctrl+C → broadcast STOPPING

**Out of scope:** worker search loop (`net-high-03`), stats aggregation (`net-medium-01`), hit pipeline (`net-medium-02`).

## Implementation steps

### Step 1 — Run state machine

```go
type RunState int
const (
    Lobby RunState = iota
    Running
    Stopping
    Ended
)

type RunConfig struct {
    Formats      bitcoin.FormatSet
    MinBalance   uint64
    CacheETag    string
}
```

Coordinator holds `sync.RWMutex`-protected state + connected worker registry.

### Step 2 — Registration API

```
POST /api/v1/register
Body: {"hostname":"host-a","threads":16,"version":"..."}
Response: {"worker_id":"w-uuid","run_state":"lobby","config":{...}}
```

Store: `worker_id`, hostname, threads, last heartbeat, state (`waiting`|`running`).

### Step 3 — Heartbeat

```
POST /api/v1/heartbeat
Body: {"worker_id":"...","state":"waiting"}
```

Mark stale workers offline after 30s without heartbeat.

### Step 4 — Run status endpoint

```
GET /api/v1/run
Response: {"state":"lobby|running|stopping|ended","config":{...}}
```

Workers poll every 1s (or long-poll 30s timeout).

### Step 5 — Enter-to-start (TTY)

When stdin is a TTY:

```
Workers connected: 3
  host-a  16 threads  waiting
  host-b   8 threads  waiting
Press Enter to start search...
```

On Enter → set state `RUNNING`, stamp `started_at`.

Non-TTY mode: `--auto-start` flag for scripted runs (documented).

### Step 6 — Graceful stop

SIGINT on coordinator → `STOPPING` → workers observe via `/run` → coordinator waits for final heartbeats → `ENDED` + summary.

### Step 7 — Funded load at startup

Coordinator runs same `ensureFunded()` + mmap load as standalone (reuse shared helpers).

## Files to touch

| File | Action |
|------|--------|
| `cmd/btcfind-coordinator/main.go` | New binary entry |
| `coordinator/lobby.go` | Registry + run state |
| `coordinator/handlers.go` | register, heartbeat, run |
| `coordinator/tty.go` | Enter gate |
| `coordinator/lobby_test.go` | State transitions |

## Verification

```bash
go build -o btcfind-coordinator ./cmd/btcfind-coordinator
./btcfind-coordinator --listen :8443 --tls-cert cert.pem --tls-key key.pem --auth-token test

# second terminal: curl register (no search yet)
curl -k -H "Authorization: Bearer test" -d '{"hostname":"dev","threads":8}' \
  https://localhost:8443/api/v1/register
```

- [ ] Workers stay in `waiting` until Enter pressed
- [ ] `/run` flips to `running` after Enter
- [ ] Ctrl+C transitions to `stopping`
- [ ] Standalone `btcfind` benchmark unchanged

## Rollback

Remove `cmd/btcfind-coordinator` and coordinator lobby code. Cache HTTP from `net-high-05` can remain for tests.