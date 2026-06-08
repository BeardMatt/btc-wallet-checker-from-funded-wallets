# Distributed search — system design (draft)

| Field | Value |
|-------|-------|
| **Status** | Draft (approved 2026-06-08) |
| **Supersedes** | `low-03` sketch — split into `net-*` plans |
| **Goal** | Linear scale-out: N workers ≈ N × single-machine keys/sec |

## Summary

A **coordinator** (`btcfind-coordinator`) loads the funded index, accepts **worker** connections (`btcfind-worker`), waits until the operator presses **Enter**, then starts a cluster-wide search. Each worker runs the **full hot loop locally** (keygen + lookup). The coordinator aggregates stats and owns the canonical hit log.

Random key search is probabilistic — workers do **not** partition keyspace. Overlap between machines is acceptable.

## Roles

| Role | Binary | Responsibility |
|------|--------|----------------|
| **Coordinator** | `btcfind-coordinator` | Funded data source, run config, start gate, TLS, stats, hits |
| **Worker** | `btcfind-worker` | Connect, sync cache, search with **local** `--threads`, report stats/hits |
| **Standalone** | `btcfind` | Unchanged — single-machine use |

## Operator workflow

```
1. Coordinator host:
   btcfind-coordinator --listen :8443 --tls-cert ... --tls-key ... --auth-token ...

2. Each worker machine (threads per machine):
   btcfind-worker --coordinator https://coordinator:8443 --threads 16 --auth-token ...

3. Coordinator TTY shows:
   Workers connected: 3 (host-a 16t, host-b 8t, host-c 32t)
   Press Enter to start search...

4. Operator presses Enter → all workers begin searching

5. Live display:
   Cluster: 412,000 keys/s | 1.2B keys tried | Hits: 0 | Workers: 3/3

6. On hit → coordinator banner + wallets.txt (workers do not keep canonical WIF log)

7. Ctrl+C on coordinator → graceful stop broadcast to workers
```

## Design decisions (approved)

| Question | Decision |
|----------|----------|
| Network scope | **LAN and internet-facing from day one** (TLS required on coordinator) |
| Start gate | **Enter on coordinator TTY** |
| Binaries | **Separate** `btcfind-coordinator` and `btcfind-worker` |
| Thread count | **Per-worker** — each client passes its own `--threads` (or default `NumCPU`) |

## Architecture

```
┌─────────────────────────┐
│   btcfind-coordinator   │
│  - funded.cache (mmap)  │
│  - TLS + auth token     │
│  - run config (formats, │
│    min-balance)         │
│  - Enter → START        │
│  - aggregate stats      │
│  - wallets.txt (hits)   │
└───────────┬─────────────┘
            │ HTTPS (control + cache + stats + hits only)
    ┌───────┼───────┐
    ▼       ▼       ▼
 Worker   Worker   Worker
 (16 thr) (8 thr)  (32 thr)
 local    local    local
 hot loop hot loop hot loop
```

**Critical rule:** zero per-key network I/O. Workers only call the coordinator for:

- Registration / heartbeat
- Cache download (once per run, or on mismatch)
- Stats batches (~every 1–5s)
- Hit reports (rare)

## Run configuration

Coordinator owns **search parameters** for the cluster:

| Setting | Source | Notes |
|---------|--------|-------|
| `--formats` | Coordinator | Broadcast at START |
| `--min-balance` | Coordinator | Determines which `funded*.cache` to serve |
| `--threads` | **Each worker** | Declared at register; not overridden |
| `--forever` / max keys | Coordinator (optional v2) | v1: run until coordinator stop |

Workers reject START if local cache does not match coordinator `cache_version` header.

## HTTP API (v1)

Base: `https://<coordinator>/api/v1`

### Worker → Coordinator

| Method | Path | Body | Purpose |
|--------|------|------|---------|
| `POST` | `/register` | `{hostname, threads, version}` | Join lobby; get `worker_id` |
| `POST` | `/heartbeat` | `{worker_id, state: waiting\|running}` | Keepalive every 10s |
| `GET` | `/run` | — | Long-poll or fetch current run state + config |
| `GET` | `/cache` | — | Download funded cache (`If-None-Match` supported) |
| `POST` | `/stats` | `{worker_id, keys_tried_delta, keys_per_second}` | Batched throughput |
| `POST` | `/hit` | `{worker_id, address, wif, format, balance_sats, key_index}` | Report match |

### Coordinator internal

- `GET /health` — unauthenticated liveness (no sensitive data)
- Auth: `Authorization: Bearer <token>` on all `/api/v1/*` except `/health`

### Run states

```
LOBBY → RUNNING → STOPPING → ENDED
```

- **LOBBY:** workers register; coordinator waits for Enter
- **RUNNING:** workers search; stats flow in
- **STOPPING:** coordinator broadcast stop; workers drain + final stats
- **ENDED:** summary printed; optional new lobby on restart

## Funded cache distribution

Coordinator serves the same v3 cache format already on disk:

```
GET /api/v1/cache
Headers:
  ETag: "<mtime-nano>-<min_balance_sats>"
  X-Cache-Min-Balance: 30000
Body: funded.cache bytes (or funded.N.cache)
```

Worker flow:

1. Compare local cache ETag with coordinator
2. Download if missing or mismatch (~900MB; show progress)
3. mmap load identically to standalone `btcfind`

## Security (internet-facing day one)

| Layer | v1 requirement |
|-------|----------------|
| Transport | TLS 1.2+ on coordinator (self-signed OK for LAN; documented cert setup for internet) |
| Auth | Shared bearer token (`--auth-token` / env `BTCFIND_AUTH_TOKEN`) |
| Hits | WIF crosses network only on `/hit` — document risk; recommend VPN for untrusted internet |
| Optional later | mTLS, per-worker keys (`net-optional-01`) |

Workers validate coordinator certificate (`--tls-skip-verify` **only** for dev/LAN testing).

## Stats model

- Each worker tracks `run_keys_tried` locally for the current run
- Every N seconds: `POST /stats` with delta since last report
- Coordinator sums into `cluster_total_keys` and computes aggregate keys/sec
- Per-worker keys/sec shown in dashboard for imbalance visibility

## Hit handling

1. Worker detects hit via existing `matchFunded` + `PrintHit` logic (local banner optional/suppressed)
2. Worker `POST /hit` with full payload
3. Coordinator writes `wallets.txt` (append-only, mode 0600) and prints framed banner on TTY
4. Coordinator increments `cluster_hits`

Duplicate reports (same address from two workers) — log second as warning; probabilistically negligible.

## Code layout (implementation target)

```
cmd/btcfind-coordinator/main.go
cmd/btcfind-worker/main.go
coordinator/          # HTTP server, lobby, run state
worker/               # client, cache sync, search glue
search/               # extracted hot loop (from cryptogen.go)
```

Standalone `btcfind` imports `search/` — behavior unchanged.

## Performance expectations

| Setup | Expected aggregate throughput |
|-------|-------------------------------|
| 1 worker, 8 threads | ~120k keys/s (baseline) |
| 2 workers, 8 threads each | ~240k keys/s (±10% protocol overhead) |
| N workers | ~N × per-worker keys/s |

Protocol overhead target: &lt;2% of hot-loop time on workers.

## Rollback

Remove `cmd/btcfind-coordinator`, `cmd/btcfind-worker`, `coordinator/`, `worker/`, `search/` if extracted. Standalone `btcfind` remains the default binary.

## Plan index

See `plans/manifest.json` → `network_recommended_order` and `plans/net/README.md`.