---
name: benchmark
description: >
  Run and record btcfind performance benchmarks after every code change.
  Use when implementing optimization plans, modifying cryptogen.go, wallets.go,
  funded loading, or any performance-sensitive code. Also use when the user asks
  to benchmark, track performance history, measure throughput, or runs /benchmark.
metadata:
  short-description: "Benchmark after changes and log performance history"
  author: btcfind project
compatibility: Requires go, python3, built btcfind binary, funded.tsv present
---

# Benchmark After Every Change

This project tracks performance history in `plans/benchmarks.json`. **After every code change that could affect performance, run a benchmark and append a new entry.**

## When to run

- After completing any plan in `plans/manifest.json`
- After any edit to `cryptogen.go`, `bitcoin/`, `funded_download.go`, or funded loading logic
- Before marking a plan `completed` in `plans/manifest.json`
- When the user asks for performance results

## Standard benchmark

Always use the same command so results are comparable:

```bash
./.grok/skills/benchmark/scripts/run-benchmark.sh
```

Or manually:

```bash
go build -o btcfind .
time ./btcfind 50000 8
```

Record these metrics from the output:

| Metric | Source |
|--------|--------|
| `keys_per_second` | `Average X keys per second` line |
| `hot_loop_seconds` | `Took Xs...` line |
| `wall_seconds` | `time` real elapsed |
| `startup_seconds` | `wall_seconds - hot_loop_seconds` |

## Record results

1. Read `plans/benchmarks.json`
2. Get current commit: `git rev-parse --short HEAD`
3. Identify the plan being implemented (e.g. `high-03`) from context or `plans/manifest.json`
4. Compute `delta_vs_previous` vs the most recent entry that used the standard command (`./btcfind 50000 8`):

```python
delta = ((new_kps - prev_kps) / prev_kps) * 100  # percent change
```

5. Append a new entry to the `entries` array:

```json
{
  "date": "YYYY-MM-DD",
  "commit": "<short hash>",
  "plan_id": "<plan-id or 'adhoc'>",
  "title": "<human-readable change description>",
  "command": "./btcfind 50000 8",
  "num_keys": 50000,
  "threads": 8,
  "keys_per_second": 0.0,
  "hot_loop_seconds": 0.0,
  "wall_seconds": 0.0,
  "startup_seconds": 0.0,
  "delta_vs_previous": "+X.X%",
  "notes": "<what changed, any caveats>"
}
```

6. Write `plans/benchmarks.json` back (preserve formatting with 2-space indent)
7. If implementing a plan, update `plans/manifest.json` status only after benchmark is recorded
8. Report to the user:
   - New `keys_per_second`
   - Delta vs previous standard benchmark
   - Startup time if startup-related plan (high-02, medium-04)

## Rules

- **Never skip** the benchmark when completing an optimization plan
- **Never compare** runs with different `num_keys` or `threads` without noting the mismatch in `notes`
- If `funded.tsv` is missing, `ensureFunded()` will download it — note unusually long wall time in `notes`
- If benchmark fails or regresses >5%, investigate before marking plan completed
- Rebuild before benchmarking: `go build -o btcfind .`

## Show history

When the user asks for performance history, read `plans/benchmarks.json` and present a table:

| Date | Plan | Keys/sec | Delta | Commit |
|------|------|----------|-------|--------|

Sort by date descending. Flag entries where `command` differs from `standard_command`.

## Current baseline

See the latest entry in `plans/benchmarks.json` with `command: "./btcfind 50000 8"`.