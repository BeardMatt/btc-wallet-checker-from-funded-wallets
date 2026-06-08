# Plan: Reverse tunnel / NAT workers (optional)

| Field | Value |
|-------|-------|
| **ID** | `net-optional-02` |
| **Priority** | Optional |
| **Status** | `pending` |
| **Depends on** | `net-high-03` |
| **Estimated gain** | Workers behind NAT can join internet coordinator |

## Problem

Workers on home networks cannot accept inbound connections. Standard design has workers **outbound** to coordinator (already true), but some NAT/firewall setups block long-lived HTTPS or require HTTP CONNECT proxies.

## Goal

Document and optionally automate outbound-only connectivity patterns: reverse SSH tunnel, Tailscale/WireGuard mesh, or `--proxy-url` for corporate proxies.

## Scope (this plan only)

- Worker `--proxy-url` (HTTP CONNECT)
- Worker `--dial-timeout`, `--keepalive` tuning
- Docs: Tailscale sidecar pattern (coordinator + workers on same tailnet)
- Optional helper: `scripts/net-tailscale-example.sh`

**Out of scope:** Built-in reverse tunnel server in coordinator (use external tools).

## Implementation steps

### Step 1 — HTTP proxy support

```go
transport := &http.Transport{
    Proxy: http.ProxyURL(proxyURL),
}
```

### Step 2 — Connection tuning

Flags for dial timeout (30s default), TCP keepalive, TLS handshake timeout — helps flaky NAT.

### Step 3 — Documentation section in design.md

Patterns:

| Scenario | Approach |
|----------|----------|
| Workers behind NAT | Outbound HTTPS to public coordinator (default) |
| No public IP | Tailscale mesh — coordinator at `100.x` |
| Corporate proxy | `--proxy-url http://proxy:8080` |
| Strict egress | SSH reverse tunnel to VPS |

### Step 4 — Example script

`scripts/net-tailscale-example.sh` — documents coordinator on tailnet IP.

## Files to touch

| File | Action |
|------|--------|
| `worker/client.go` | Proxy + timeouts |
| `cmd/btcfind-worker/main.go` | Flags |
| `plans/net/design.md` | Connectivity section |

## Verification

- [ ] Worker connects through local mitmproxy / squid test proxy
- [ ] Docs accurately describe outbound-only architecture

## Rollback

Remove proxy flags; direct dial only.