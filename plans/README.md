# btcfind Optimization Plans

Incremental rollout plans for performance improvements. Each change is isolated in its own directory with a `plan.md` you can execute one at a time.

## Quick start

Open a plan directly:

```bash
cat plans/high/04-fix-worker-count/plan.md
```

Or browse the index in `plans/manifest.json` for status, dependencies, and recommended order.

## Directory layout

```
plans/
├── README.md           # this file
├── manifest.json       # plan index (status, deps, order)
├── high/               # high-impact plans (4)
├── medium/             # medium-impact plans (4)
└── low/                # low-priority plans (5)
```

Each plan lives at `<tier>/<slug>/plan.md`.

## Recommended rollout order

Work through plans in this order to respect dependencies:

| Step | Plan ID | Title |
|------|---------|-------|
| 1 | `high-04` | Fix worker count |
| 2 | `high-03` | Defer WIF encoding |
| 3 | `medium-01` | Partition funded set by address type |
| 4 | `high-01` | Hash-based lookup (no string encode) |
| 5 | `medium-04` | Faster TSV parsing |
| 6 | `high-02` | Funded cache file |
| 7 | `medium-02` | Bloom filter pre-check |
| 8 | `medium-03` | Batch keys per goroutine |
| 9 | `low-01` | Hand-rolled encoding |
| 10 | `low-02` | GPU / assembly secp256k1 |
| 11 | `low-03` | Distributed search |
| 12 | `low-04` | Default thread count to CPU count |
| 13 | `low-05` | Configurable minimum funded balance |

## Status workflow

Each plan has a `status` field in `manifest.json`:

- `pending` — not started
- `in_progress` — currently being worked on
- `completed` — done and verified
- `skipped` — evaluated and intentionally not pursued
- `maybe` — under consideration for future work; not scheduled

Update `status` in `manifest.json` as you complete each plan.

## Benchmark baseline

Record results before starting so each plan can be measured in isolation:

```bash
./btcfind 50000 8
```

Note the "Average X keys per second" line and startup time (load + sort).