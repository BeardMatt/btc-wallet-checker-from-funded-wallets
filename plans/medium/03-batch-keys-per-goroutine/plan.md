# Plan: Batch keys per goroutine

| Field | Value |
|-------|-------|
| **ID** | `medium-03` |
| **Priority** | Medium |
| **Status** | `completed` |
| **Depends on** | none |
| **Estimated gain** | 5–15% from reduced channel overhead |

## Problem

Each worker goroutine generates one key and sends it over a channel. The main goroutine receives one at a time. Channel send/receive has synchronization overhead that adds up at 30k+ keys/sec.

## Goal

Workers generate keys in local batches (e.g. 64–256) and send batches to the consumer.

## Scope (this plan only)

- Batch channel protocol
- Consumer iterates batch locally

**Out of scope:** lookup, encoding, loading changes.

## Implementation steps

### Step 1 — Define batch size constant

```go
const keyBatchSize = 128
```

Make configurable via env or flag later if needed.

### Step 2 — Change channel type

```go
ch := make(chan []bitcoin.Wallet, workers)
```

### Step 3 — Update worker loop

```go
go func() {
    batch := make([]bitcoin.Wallet, 0, keyBatchSize)
    for {
        batch = append(batch, bitcoin.GenKeypair())
        if len(batch) >= keyBatchSize {
            ch <- batch
            batch = make([]bitcoin.Wallet, 0, keyBatchSize)
        }
    }
}()
```

Consider reusing batch slice with `batch = batch[:0]` to reduce allocations.

### Step 4 — Update consumer loop

```go
for processed < numtests {
    batch := <-ch
    for _, wallet := range batch {
        if processed >= numtests { break }
        // lookup...
        processed++
    }
}
```

### Step 5 — Tune batch size

Benchmark with 32, 64, 128, 256:

```bash
./btcfind 200000 8
```

## Files to touch

| File | Action |
|------|--------|
| `cryptogen.go` | Batch channel send/receive |

## Verification

- [x] Throughput ≥ unbatched version (+11.4% vs medium-02)
- [x] Exact key count still honored (`numtests` not exceeded)
- [x] No goroutine leak at exit (workers exit with process)

## Rollback

Revert channel to `chan bitcoin.Wallet` single-key protocol.