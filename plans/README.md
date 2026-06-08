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
└── low/                # low-priority plans (16)
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
| 14 | `low-06` | Simulate-hit flag for testing |
| 15 | `low-07` | Output UX — pretty terminal, progress, wallet recovery |
| 16 | `low-08` | Persist discovered wallets to wallets.txt |
| 17 | `low-16` | Verbosity for funded.tsv check at startup |
| 18 | `low-10` | `--formats` flag to skip address types |
| 19 | `low-09` | Lazy per-format derivation (taproot last) |
| 20 | `low-15` | SHA-NI accelerated Hash160 |
| 21 | `low-12` | Store balance in funded index |
| 22 | `low-11` | High-value-only funded index |
| 23 | `low-13` | mmap funded cache |
| 24 | `low-14` | Run-forever mode and session checkpoints |

### New plans (low-09–low-15) — priority and dependencies

| Plan | Priority | Depends on | Why this order |
|------|----------|------------|----------------|
| `low-16` | Low | `low-07` | Quick UX fix — funded.tsv check currently silent when up to date |
| `low-10` | Medium | — | Simple flag; quick win before deeper derivation refactor |
| `low-09` | **High** | — | Largest single-machine perf lever; synergizes with `low-10` |
| `low-15` | Low | — | Small hash win; best done while derivation code is in flux |
| `low-12` | Medium | — | Cache v3 + balance on hit; foundation for whale tier |
| `low-11` | Medium | `low-12` | Tiered index needs balance at load and on hit |
| `low-13` | Low–Medium | `high-02`, `low-12` | mmap after cache format stabilizes |
| `low-14` | Low | `low-08` | Long-run ops; independent of perf work |

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