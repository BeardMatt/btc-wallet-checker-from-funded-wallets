# Plan: Fix worker count (use all threads)

| Field | Value |
|-------|-------|
| **ID** | `high-04` |
| **Priority** | High |
| **Status** | `completed` |
| **Depends on** | none |
| **Estimated gain** | 10–30% on multi-core machines |

## Problem

The program accepts a `threads` argument but spawns only `threads - 1` worker goroutines:

```go
ch := newWallet(threads - 1)
```

With `--threads 4`, only 3 keys are generated in parallel.

## Goal

Spawn exactly `threads` workers (or `runtime.NumCPU()` when threads is 0).

## Scope (this plan only)

- Fix goroutine count
- Optionally default to `runtime.NumCPU()` when not specified

**Out of scope:** batching, lookup, encoding changes.

## Implementation steps

### Step 1 — Fix worker spawn count

In `cryptogen.go`:

```go
workers := threads
if workers < 1 {
    workers = 1
}
ch := newWallet(workers)
```

Remove the `threads - 1` logic entirely. Also remove the forced bump from 1→2 unless you want to keep a minimum of 2 for the main+worker model — the main goroutine only consumes, so `threads` workers is correct.

### Step 2 — Size channel buffer

```go
ch := make(chan bitcoin.Wallet, workers)
```

Buffer depth = worker count to reduce blocking.

### Step 3 — Optional: default to NumCPU

```go
if threads == 0 {
    threads = runtime.NumCPU()
}
```

Update usage string if added.

### Step 4 — Benchmark at several thread counts

```bash
./btcfind 50000 1
./btcfind 50000 4
./btcfind 50000 8
./btcfind 50000 16
```

Find the knee where more threads stop helping (likely near physical core count).

## Files to touch

| File | Action |
|------|--------|
| `cryptogen.go` | Fix `newWallet` call and channel buffer |

## Verification

- [ ] `./btcfind 50000 4` uses 4 generator goroutines (verify with `pprof` or logging)
- [ ] Throughput at N threads ≥ throughput at N-1 threads from old code
- [ ] No deadlock or panic at threads=1

## Rollback

Revert single line to `newWallet(threads - 1)`.