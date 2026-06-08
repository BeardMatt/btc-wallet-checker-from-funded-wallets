# Network / distributed search plans (`net-*`)

Horizontal scale-out: one **coordinator** + N **workers**, linear aggregate throughput.

**Design doc:** [`design.md`](design.md)

## Quick start

```bash
cat plans/net/design.md
cat plans/manifest.json   # network_recommended_order
```

## Recommended order

| Step | Plan ID | Title |
|------|---------|-------|
| 1 | `net-high-01` | Extract shared search package |
| 2 | `net-high-05` | Funded cache HTTP distribution |
| 3 | `net-high-04` | TLS + bearer auth |
| 4 | `net-high-02` | Coordinator core + Enter-to-start |
| 5 | `net-high-03` | Worker binary |
| 6 | `net-medium-03` | Run config sync (formats, min-balance) |
| 7 | `net-medium-01` | Cluster stats aggregation |
| 8 | `net-medium-02` | Hit pipeline + coordinator wallets.txt |
| 9 | `net-low-01` | Reconnect + coordinator restart |
| 10 | `net-low-02` | Coordinator live dashboard |
| 11 | `net-optional-01` | mTLS + cert tooling (optional) |
| 12 | `net-optional-02` | Reverse tunnel / NAT workers (optional) |
| 13 | `net-optional-03` | Public coordinator abuse limits (optional) |

## Status workflow

Same as main plans: `pending` → `in_progress` → `completed` (or `skipped`).

## Binaries (target)

| Binary | Purpose |
|--------|---------|
| `btcfind` | Single machine (unchanged) |
| `btcfind-coordinator` | Server / lobby / start gate |
| `btcfind-worker` | Connects to coordinator; local `--threads` |