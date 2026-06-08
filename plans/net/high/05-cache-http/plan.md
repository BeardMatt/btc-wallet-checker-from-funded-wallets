# Plan: Funded cache HTTP distribution

| Field | Value |
|-------|-------|
| **ID** | `net-high-05` |
| **Priority** | High |
| **Status** | `pending` |
| **Depends on** | `net-high-01` |
| **Estimated gain** | Workers load funded index without manual rsync |

## Problem

Workers need the same v3 funded cache (~900MB mmap) as the coordinator. Copying files by hand does not scale across LAN or internet-facing deployments.

## Goal

Serve the on-disk cache file over HTTP with ETag support so workers download once per run (or skip when unchanged).

## Scope (this plan only)

- `GET /api/v1/cache` handler (plain HTTP first — TLS added in `net-high-04`)
- ETag from `mtime` + `min_balance_sats`
- `If-None-Match` → `304 Not Modified`
- Progress-friendly `Content-Length` streaming

**Out of scope:** TLS, auth (next plans), worker download client (in `net-high-03`).

## Implementation steps

### Step 1 — Cache metadata helper

```go
// coordinator/cache_serve.go
type CacheMeta struct {
    Path            string
    ETag            string
    MinBalanceSats  uint64
    Size            int64
}
func CacheMetaFrom(cfg CoordinatorConfig) (CacheMeta, error)
```

Reuse `fundedCachePath(minBalance)` logic from standalone.

### Step 2 — HTTP handler

```
GET /api/v1/cache
Response headers:
  ETag: "<mtime-nano>-<min_balance>"
  X-Cache-Min-Balance: 30000
  Content-Type: application/octet-stream
Body: raw cache bytes
```

Support range requests optional (v2); v1 full-file GET is sufficient.

### Step 3 — Standalone test server

Temporary `btcfind-coordinator --serve-cache-only` or unit test with `httptest.Server` to validate ETag round-trip.

### Step 4 — Document size expectations

Note ~900MB download; workers should show progress bar during sync (implemented in `net-high-03`).

## Files to touch

| File | Action |
|------|--------|
| `coordinator/cache_serve.go` | New — meta + handler |
| `coordinator/cache_serve_test.go` | New — ETag / 304 tests |
| `funded_cache.go` | Export path helper if needed |

## Verification

```bash
go test ./coordinator/... -run Cache
curl -I http://localhost:8080/api/v1/cache
curl -H 'If-None-Match: "..."' http://localhost:8080/api/v1/cache  # → 304
```

- [ ] Full download byte-identical to local `funded.cache`
- [ ] ETag changes when cache rebuilt or `--min-balance` changes
- [ ] No regression to standalone `btcfind` benchmark

## Rollback

Remove `coordinator/cache_serve.go`. Standalone cache loading unchanged.