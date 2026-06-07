# btcfind — Project Instructions

Bitcoin key search tool. Generates random keypairs, derives addresses across formats (1/3/bc1q/bc1p), and checks against a funded address list.

## Resuming work

1. Read `plans/manifest.json` for plan status, dependencies, and `recommended_order`
2. Read `plans/benchmarks.json` for performance history
3. Open the next `pending` plan's `plan.md` from `plans/<tier>/<slug>/`
4. Implement **one plan at a time** — do not batch multiple plans

## Optimization workflow

- Follow `recommended_order` in `plans/manifest.json` unless the user specifies otherwise
- Respect `depends_on` — do not start a plan until its dependencies are `completed`
- Mark plan `in_progress` before starting, `completed` only after benchmark is recorded
- Keep changes scoped to what the plan describes — no drive-by refactors

### Completed plans

Check `plans/manifest.json` for current status. As of last update: `high-04` (fix worker count) is completed.

## Benchmark after every change

Performance history lives in `plans/benchmarks.json`. **After every code change that could affect performance, run a benchmark and append a new entry.**

### When to benchmark

- After completing any plan in `plans/manifest.json`
- After any edit to `cryptogen.go`, `bitcoin/`, `funded_download.go`, or funded loading logic
- Before marking a plan `completed` in `plans/manifest.json`
- When the user asks for performance results

### Standard benchmark

Always use the same command so results are comparable:

```bash
./scripts/run-benchmark.sh
```

Or manually:

```bash
go build -o btcfind .
time ./btcfind 50000 8
```

Record these metrics:

| Metric | Source |
|--------|--------|
| `keys_per_second` | `Average X keys per second` line |
| `hot_loop_seconds` | `Took Xs...` line |
| `wall_seconds` | `time` real elapsed |
| `startup_seconds` | `wall_seconds - hot_loop_seconds` |

### Record results

1. Read `plans/benchmarks.json`
2. Get current commit: `git rev-parse --short HEAD`
3. Identify the plan being implemented (e.g. `high-03`)
4. Compute `delta_vs_previous` vs the most recent entry using `standard_command`:

```
delta = ((new_kps - prev_kps) / prev_kps) * 100
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

6. Write `plans/benchmarks.json` (2-space indent)
7. Update `plans/manifest.json` status only after benchmark is recorded
8. Report to the user: new keys/sec, delta vs previous, startup time if relevant

### Benchmark rules

- **Never skip** the benchmark when completing an optimization plan
- **Never compare** runs with different `num_keys` or `threads` without noting it in `notes`
- If `funded.tsv` is missing, `ensureFunded()` downloads it — note long wall time in `notes`
- If benchmark regresses >5%, investigate before marking plan completed
- Rebuild before benchmarking: `go build -o btcfind .`

### Show performance history

Read `plans/benchmarks.json` and present:

| Date | Plan | Keys/sec | Delta | Commit |

Sort by date descending. Flag entries where `command` differs from `standard_command`.

## Build and run

```bash
go build -o btcfind .
./btcfind <num_keys> [threads]
```

`threads` is optional — defaults to `runtime.NumCPU()`. Pass `0` or omit for auto; pass an explicit number to override (e.g. `./btcfind 50000 8` for benchmarks).

`funded.tsv` is downloaded automatically on first run if missing (see `funded_download.go`).

## Code conventions

- Match existing Go style in the repo
- Only modify files required by the current plan
- Run `go build -o btcfind .` to verify compilation after changes