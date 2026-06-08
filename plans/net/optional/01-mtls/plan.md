# Plan: mTLS + cert tooling (optional)

| Field | Value |
|-------|-------|
| **ID** | `net-optional-01` |
| **Priority** | Optional |
| **Status** | `pending` |
| **Depends on** | `net-high-04` |
| **Estimated gain** | Stronger auth for internet-facing coordinators |

## Problem

Shared bearer token leaks grant full cluster access. Public coordinators benefit from per-worker client certificates.

## Goal

Optional mTLS mode: coordinator requires client cert signed by cluster CA; provide `scripts/net-gen-certs.sh` for CA + server + worker cert generation.

## Scope (this plan only)

- `--mtls-ca`, `--mtls-require-client-cert` on coordinator
- Worker: `--tls-cert`, `--tls-key`, `--tls-ca`
- Token auth remains as second factor or fallback (`--auth-token` still required)

**Out of scope:** ACME/Let's Encrypt automation, HSM.

## Implementation steps

### Step 1 — Cert generation script

```bash
./scripts/net-gen-certs.sh --out ./certs --san coordinator.example.com
```

Outputs: `ca.pem`, `server.pem`, `server-key.pem`, `worker.pem`, `worker-key.pem`.

### Step 2 — Coordinator TLS config

```go
tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
tlsConfig.ClientCAs = caCertPool
```

### Step 3 — Worker client cert

Load worker cert/key for outbound TLS handshake.

### Step 4 — Documentation

When to use mTLS vs bearer-only; rotation procedure.

## Files to touch

| File | Action |
|------|--------|
| `scripts/net-gen-certs.sh` | New |
| `coordinator/server.go` | mTLS mode |
| `worker/client.go` | Client cert loading |
| `plans/net/design.md` | Security table update |

## Verification

- [ ] Worker without client cert rejected when mTLS required
- [ ] Worker with valid cert connects
- [ ] LAN bearer-only mode still works when mTLS disabled

## Rollback

Disable mTLS flags; revert to TLS + bearer from `net-high-04`.