# Plan: Distributed search across machines

| Field | Value |
|-------|-------|
| **ID** | `low-03` |
| **Priority** | Low |
| **Status** | `maybe` |
| **Depends on** | `high-01`, `high-02` |
| **Estimated gain** | Linear scale-out; not single-machine optimization |

## Problem

Single-machine throughput caps out at CPU core count and memory bandwidth. Searching larger keyspaces faster requires horizontal scaling.

## Goal

Design a coordinator/worker model that partitions the random key search across multiple machines without duplicate work or missed coverage.

## Scope (this plan only)

- Architecture design and minimal coordinator/worker prototype
- Shared funded index distribution (cache file sync)

**Out of scope:** single-machine optimizations.

## Implementation steps

### Step 1 — Define work unit

Options:

| Strategy | Description |
|----------|-------------|
| **Counter-based** | Coordinator assigns `[start, end)` key indices; workers seed PRNG deterministically |
| **Stream-based** | Workers pull batches of N keys from coordinator queue |
| **Range-based** | Partition private key space (impractical — space is 2^256) |

Recommended: **stream-based batches** with shared random search (collisions acceptable — search is probabilistic).

### Step 2 — Funded index distribution

- Coordinator hosts `funded.cache` (from `high-02`)
- Workers download cache on startup (HTTP or rsync)
- Workers run fully local lookups (no network per key)

### Step 3 — Worker binary

```bash
btcfind-worker --coordinator http://host:8080 --threads 8
```

Worker loop:

1. Request batch ID from coordinator
2. Generate + check keys locally
3. Report hits and throughput stats

### Step 4 — Coordinator API

```
POST /register        → worker ID
GET  /stats           → aggregate keys/sec
POST /hit             → worker reports match (WIF, address)
GET  /funded.cache    → serve cache file
```

### Step 5 — Prototype with two local processes

Run coordinator + 2 workers on same machine to validate protocol before real distribution.

## Files to touch

| File | Action |
|------|--------|
| `cmd/coordinator/main.go` | New — HTTP coordinator |
| `cmd/worker/main.go` | New — worker process |
| `cryptogen.go` | Extract search loop into reusable package |

## Verification

- [ ] 2 local workers achieve ~2× single-worker throughput
- [ ] Hits reported to coordinator exactly once
- [ ] Workers survive coordinator restart (re-register)

## Rollback

Delete `cmd/coordinator` and `cmd/worker`. Keep single-binary `btcfind` unchanged.