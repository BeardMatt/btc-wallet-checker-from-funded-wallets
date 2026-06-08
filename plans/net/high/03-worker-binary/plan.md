# Plan: Worker binary

| Field | Value |
|-------|-------|
| **ID** | `net-high-03` |
| **Priority** | High |
| **Status** | `pending` |
| **Depends on** | `net-high-02` |
| **Estimated gain** | Linear scale-out — N workers ≈ N × local keys/sec |

## Problem

No client exists to connect to the coordinator, sync cache, and run the hot loop locally with per-machine thread counts.

## Goal

Ship `btcfind-worker` as a **separate binary** that registers with the coordinator, downloads cache if needed, waits for `RUNNING`, then searches using **its own** `--threads` (default `runtime.NumCPU()`).

## Scope (this plan only)

- `cmd/btcfind-worker/main.go`
- Register → sync cache → poll `/run` → `search.Run` when `running`
- Heartbeat every 10s
- Stop when coordinator signals `stopping`
- Zero per-key network I/O

**Out of scope:** stats batches (`net-medium-01`), hit POST (`net-medium-02`), run config fields beyond cache ETag (`net-medium-03`).

## Implementation steps

### Step 1 — CLI

```bash
btcfind-worker \
  --coordinator https://coordinator:8443 \
  --threads 16 \
  --auth-token "$BTCFIND_AUTH_TOKEN" \
  [--tls-skip-verify]  # dev/LAN only
```

`--threads 0` → `runtime.NumCPU()`. Thread count is **never** overridden by coordinator.

### Step 2 — Registration flow

1. `POST /register` with hostname + threads + build version
2. Receive `worker_id` + initial run state + config snapshot

### Step 3 — Cache sync

1. Read local cache path + compute ETag (or missing)
2. `GET /api/v1/cache` with `If-None-Match`
3. On `200`: stream to temp file, atomic rename, show progress on stderr
4. On `304`: use local cache
5. mmap via shared funded loader (same as standalone)

### Step 4 — Lobby wait

Poll `GET /api/v1/run` until `state == running`. Heartbeat `state: waiting` during lobby.

### Step 5 — Search

```go
search.Run(ctx, search.Config{Threads: localThreads, ...}, funded, hooks)
```

Use placeholder hooks (local stderr progress only) until `net-medium-01`/`net-medium-02`.

### Step 6 — Stop handling

On `stopping`: cancel context, drain workers, send final heartbeat `state: waiting`, exit 0.

### Step 7 — Two-local-process smoke test

```bash
# terminal 1
./btcfind-coordinator ... 

# terminal 2 & 3
./btcfind-worker --coordinator https://127.0.0.1:8443 --threads 4 --tls-skip-verify ...
./btcfind-worker --coordinator https://127.0.0.1:8443 --threads 4 --tls-skip-verify ...
```

Press Enter → both workers search. Expect ~2× single-worker keys/sec (±10%).

## Files to touch

| File | Action |
|------|--------|
| `cmd/btcfind-worker/main.go` | New binary entry |
| `worker/client.go` | HTTP client (register, run, cache) |
| `worker/cache_sync.go` | Download + ETag |
| `worker/run.go` | Lobby + search glue |
| `worker/client_test.go` | Mock coordinator tests |

## Verification

- [ ] Worker respects local `--threads` (16 vs 8 visible in coordinator lobby)
- [ ] Cache download once; second worker reuses or 304s
- [ ] 2 local workers ≈ 2× throughput after Enter
- [ ] `./btcfind 50000 8` benchmark unchanged

## Rollback

Remove `cmd/btcfind-worker` and `worker/` package. Coordinator can still run in lobby-only mode.