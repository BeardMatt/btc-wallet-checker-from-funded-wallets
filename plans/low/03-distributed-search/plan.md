# Plan: Distributed search across machines

| Field | Value |
|-------|-------|
| **ID** | `low-03` |
| **Priority** | Low |
| **Status** | `skipped` |
| **Depends on** | `high-01`, `high-02` |
| **Estimated gain** | Superseded — see `plans/net/*` |
| **Superseded by** | `plans/net/` (`net-high-01` … `net-optional-03`) |

## Status

This sketch is **superseded** by the network plan tree. Do not implement from this file.

Use instead:

- **Design:** [`plans/net/design.md`](../../net/design.md)
- **Index:** [`plans/net/README.md`](../../net/README.md)
- **Order:** `plans/manifest.json` → `network_recommended_order`

## Approved design decisions (2026-06-08)

| Topic | Decision |
|-------|----------|
| Network scope | LAN + internet-facing (TLS day one) |
| Start gate | Enter on coordinator TTY |
| Binaries | Separate `btcfind-coordinator` + `btcfind-worker` |
| Thread count | Per-worker `--threads` (not coordinator-controlled) |
| Keyspace | No partitioning — probabilistic overlap OK |

## Original problem (historical)

Single-machine throughput caps at CPU core count. Horizontal scaling requires a coordinator/worker model with shared funded index distribution and zero per-key network I/O.

See `net-high-01` through `net-high-03` for implementation steps.