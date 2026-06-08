# Plan: Run config sync (formats, min-balance)

| Field | Value |
|-------|-------|
| **ID** | `net-medium-03` |
| **Priority** | Medium |
| **Status** | `pending` |
| **Depends on** | `net-high-03` |
| **Estimated gain** | Consistent cluster search parameters |

## Problem

Coordinator and workers may disagree on `--formats` or `--min-balance`, causing workers to search against the wrong cache or skip formats the coordinator expects.

## Goal

Coordinator owns cluster search parameters; workers apply them at `RUNNING` and refuse start if cache ETag does not match.

## Scope (this plan only)

- Coordinator CLI: `--formats`, `--min-balance` (same tokens as standalone)
- Include full `RunConfig` in `/register` and `/run` responses
- Worker validates cache ETag + applies formats before `search.Run`
- Reject START if worker local cache mismatches coordinator `cache_version`

**Out of scope:** per-worker thread override (explicitly not supported), forever/max-keys cluster caps (v2).

## Implementation steps

### Step 1 — Coordinator config

```bash
btcfind-coordinator \
  --formats all \
  --min-balance 30000 \
  ...
```

Rebuild or select correct `funded.N.cache` at startup; embed ETag in `RunConfig`.

### Step 2 — Extend API responses

```json
{
  "state": "running",
  "config": {
    "formats": ["legacy-compressed","segwit","taproot"],
    "min_balance_sats": 30000,
    "cache_etag": "1717785600123456789-30000"
  }
}
```

### Step 3 — Worker validation

Before search:

```go
if localETag != cfg.CacheETag {
    return fmt.Errorf("cache mismatch: run btcfind-worker sync or delete local cache")
}
```

Auto re-download if coordinator ETag changed between lobby and start.

### Step 4 — Pass formats to search

```go
search.Run(ctx, search.Config{
    Threads:  localThreads,  // still local only
    Formats:  runConfig.Formats,
    ...
}, ...)
```

### Step 5 — CLI help text

Document that `--threads` is worker-local; `--formats` / `--min-balance` are coordinator-only.

## Files to touch

| File | Action |
|------|--------|
| `cmd/btcfind-coordinator/main.go` | formats + min-balance flags |
| `coordinator/lobby.go` | RunConfig fields |
| `worker/run.go` | validate + apply config |
| `cli.go` | Share format parsing helper if needed |

## Verification

- [ ] Coordinator `--formats segwit` → workers only derive segwit
- [ ] `--min-balance 100000000` serves correct cache ETag
- [ ] Worker with stale cache re-downloads before search
- [ ] Standalone `btcfind` unchanged

## Rollback

Workers ignore remote config; use local defaults (revert to `net-high-03` behavior).