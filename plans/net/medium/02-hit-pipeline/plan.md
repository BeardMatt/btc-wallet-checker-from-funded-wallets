# Plan: Hit pipeline + coordinator wallets.txt

| Field | Value |
|-------|-------|
| **ID** | `net-medium-02` |
| **Priority** | Medium |
| **Status** | `pending` |
| **Depends on** | `net-high-03` |
| **Estimated gain** | Canonical hit log on coordinator |

## Problem

Workers may detect funded matches locally but the operator needs one authoritative hit log (`wallets.txt`) and recovery banner on the coordinator TTY.

## Goal

Workers `POST /hit` with full payload; coordinator appends `wallets.txt` (mode 0600) and prints the framed recovery banner. Workers suppress or minimize local hit output.

## Scope (this plan only)

- `POST /api/v1/hit` endpoint
- Reuse existing `PrintHit` / `appendWallet` logic on coordinator
- Worker hit hook → HTTP POST (retry with backoff)
- Duplicate address warning (probabilistically rare)

**Out of scope:** encrypting WIF in transit beyond TLS (document VPN recommendation).

## Implementation steps

### Step 1 — Hit payload

```json
{
  "worker_id": "w-uuid",
  "address": "bc1q...",
  "wif": "L...",
  "format": "segwit",
  "balance_sats": 50000000,
  "key_index": 0
}
```

`key_index` is worker-local diagnostic (non-deterministic search).

### Step 2 — Coordinator handler

1. Validate worker_id registered
2. `appendWallet` — same format as standalone `wallets.txt`
3. `PrintHit` on coordinator TTY
4. Increment `cluster_hits` atomically

### Step 3 — Worker hit hook

```go
hooks.OnHit = func(hit Hit) error {
    return client.PostHit(ctx, hit, retryPolicy)
}
```

Suppress worker stderr banner (`--quiet-hits` default true on worker).

### Step 4 — Retry policy

3 retries, exponential backoff 1s/2s/4s. On persistent failure: log error locally with WIF (last resort — document risk).

### Step 5 — Simulate-hit cluster test

```bash
./btcfind-worker ... --simulate-hit --simulate-hit-at 1
```

Verify coordinator receives hit and writes `wallets.txt`.

## Files to touch

| File | Action |
|------|--------|
| `coordinator/hits.go` | Handler + dedup warning |
| `worker/hits.go` | POST client |
| `search/hooks.go` | OnHit callback |
| `output.go` | Share PrintHit if needed |

## Verification

- [ ] Real hit on worker → coordinator banner + `wallets.txt` line
- [ ] `--simulate-hit` works through worker
- [ ] Duplicate POST logs warning, does not corrupt file
- [ ] Worker does not append local `wallets.txt` by default

## Rollback

Workers print local hit banner only; remove `/hit` endpoint.