# Plan: Faster TSV parsing at startup

| Field | Value |
|-------|-------|
| **ID** | `medium-04` |
| **Priority** | Medium |
| **Status** | `completed` |
| **Depends on** | none |
| **Estimated gain** | 2–4× faster cold startup parse |

## Problem

`loadFunded()` uses `bufio.Scanner` with default 64 KB max token size and `strings.Split` per line. Parsing 2.6 GB this way is slow and allocates heavily.

## Goal

Parse the TSV with larger buffers, manual byte scanning, and optional parallelism.

## Scope (this plan only)

- Faster single-threaded parser
- Optional: parallel chunk parsing

**Out of scope:** cache file (pairs well with `high-02` afterward), partition logic (works with `medium-01`).

## Implementation steps

### Step 1 — Increase scanner buffer

```go
scanner := bufio.NewScanner(file)
buf := make([]byte, 0, 1024*1024)
scanner.Buffer(buf, 10*1024*1024)
```

Handles long lines safely.

### Step 2 — Replace strings.Split with byte scan

```go
func parseTSVLine(line []byte) (addr []byte, balance int, ok bool) {
    tab := bytes.IndexByte(line, '\t')
    if tab < 0 { return }
    addr = line[:tab]
    balance, ok = parseIntBytes(line[tab+1:])
    return
}
```

Avoid per-line string allocations; convert to string only when appending to funded set (or decode directly to bytes in `medium-01`).

### Step 3 — Pre-allocate slice capacity

Count lines first (optional, costs extra pass) or use better estimate:

```go
funded := make([]string, 0, 32_000_000)
```

### Step 4 — Optional parallel parse

1. `mmap` or seek to chunk boundaries at newline offsets
2. Parse chunks in `GOMAXPROCS` goroutines
3. Merge results (requires partition-aware merge if using sorted output)

Start with Steps 1–3 only; add parallelism only if still too slow after cache miss.

### Step 5 — Benchmark cold parse

```bash
rm -f funded.cache
time ./btcfind 100 4
```

Measure only load+sort phase (add timing log around `loadFunded`).

## Files to touch

| File | Action |
|------|--------|
| `cryptogen.go` | Faster `loadFunded` parser |
| `parse.go` | New — byte-level TSV helpers (optional) |

## Verification

- [ ] Loaded count identical before/after
- [ ] Cold parse time reduced vs baseline
- [ ] No scanner token-too-long errors

## Rollback

Restore `scanner.Scan()` + `strings.Split` loop.