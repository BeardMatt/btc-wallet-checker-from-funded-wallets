# Plan: Extract shared search package

| Field | Value |
|-------|-------|
| **ID** | `net-high-01` |
| **Priority** | High |
| **Status** | `pending` |
| **Depends on** | — |
| **Estimated gain** | Foundation — enables coordinator/worker without duplicating hot loop |

## Problem

The search hot loop (`newWallet`, `runBoundedSearch`, `runForeverSearch`, funded lookup glue) lives in `cryptogen.go` inside `package main`. Coordinator and worker binaries cannot import it without copy-paste.

## Goal

Extract reusable search logic into `search/` so `btcfind`, `btcfind-coordinator`, and `btcfind-worker` share one implementation. Standalone `btcfind` behavior and throughput must remain unchanged.

## Scope (this plan only)

- Move hot-loop types and functions into `search/` package
- Thin `cryptogen.go` wrapper calling `search.Run(...)`
- No network code, no new binaries yet

**Out of scope:** HTTP, TLS, coordinator lobby, worker client (later `net-*` plans).

## Implementation steps

### Step 1 — Define `search` package API

```go
// search/search.go
type Config struct {
    Threads       int
    Formats       bitcoin.FormatSet
    NumKeys       int           // 0 = unbounded (forever)
    SessionBase   uint64
    SimulateHit   bool
    SimulateHitAt int
    // inject paths for testing
}

type FundedIndex interface {
    // wraps FundedSets + bloom for lookup
}

func Run(ctx context.Context, cfg Config, funded FundedIndex, hooks Hooks) (Stats, error)
```

`Hooks` carries progress callbacks, hit handler, and optional UI — standalone passes existing `appUI` adapters; worker passes network hit reporter later.

### Step 2 — Move worker pool + batch loop

From `cryptogen.go`:

- `newWallet` → `search.startWorkers`
- `runBoundedSearch` / `runForeverSearch` → `search.Run`
- Keep `keyBatchSize` and lazy-derivation logic intact

### Step 3 — Funded index adapter

Wrap existing `FundedSets` + `matchFunded` in `search/funded.go` so workers can mmap-load the same structure without `package main` types.

### Step 4 — Wire standalone `btcfind`

```go
stats, err := search.Run(ctx, searchCfg, fundedAdapter, defaultHooks())
```

Verify `./btcfind 50000 8` matches pre-extraction benchmark (within noise).

### Step 5 — Export test hook

Keep `--simulate-hit` and `--inject-index-hit` working through `search.Config`.

## Files to touch

| File | Action |
|------|--------|
| `search/search.go` | New — main run loop |
| `search/workers.go` | New — goroutine pool |
| `search/funded.go` | New — lookup adapter |
| `search/hooks.go` | New — progress + hit callbacks |
| `cryptogen.go` | Thin wrapper only |
| `search/search_test.go` | New — simulate-hit smoke test |

## Verification

```bash
go build -o btcfind .
./scripts/run-benchmark.sh
./btcfind 100 --simulate-hit
```

- [ ] Benchmark delta vs previous ≤ ±1%
- [ ] All existing flags behave identically
- [ ] `go test ./search/...` passes

## Rollback

Revert extraction; restore monolithic `cryptogen.go`. No network code to remove.