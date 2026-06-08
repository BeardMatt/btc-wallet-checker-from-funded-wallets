# Plan: TLS + bearer auth

| Field | Value |
|-------|-------|
| **ID** | `net-high-04` |
| **Priority** | High |
| **Status** | `pending` |
| **Depends on** | `net-high-05` |
| **Estimated gain** | Internet-facing coordinator from day one |

## Problem

Serving funded cache and accepting worker registrations over plain HTTP exposes cluster control and WIF hit payloads to network observers. LAN-only is insufficient per approved design.

## Goal

Require TLS 1.2+ on the coordinator and bearer-token auth on all `/api/v1/*` routes (except `/health`). Workers validate server cert; dev LAN may use `--tls-skip-verify`.

## Scope (this plan only)

- Coordinator: `--tls-cert`, `--tls-key`, `--auth-token` (or `BTCFIND_AUTH_TOKEN`)
- Middleware: reject missing/invalid `Authorization: Bearer <token>`
- Worker client stubs: TLS config + auth header (used fully in `net-high-03`)
- `GET /health` — unauthenticated liveness only

**Out of scope:** mTLS (`net-optional-01`), rate limiting (`net-optional-03`).

## Implementation steps

### Step 1 — Auth middleware

```go
func RequireBearer(token string) func(http.Handler) http.Handler
```

Constant-time compare of token. Return `401` on failure; log client IP at warn level.

### Step 2 — TLS server wrapper

```go
srv := &http.Server{
    Addr:      cfg.Listen,
    Handler:   mux,
    TLSConfig: &tls.Config{MinVersion: tls.VersionTLS12},
}
srv.ListenAndServeTLS(certFile, keyFile)
```

Document self-signed cert generation for LAN:

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes
```

### Step 3 — Worker TLS client config

```go
type ClientConfig struct {
    CoordinatorURL string
    AuthToken      string
    TLSSkipVerify  bool  // dev only — prints warning
    CAFile         string // optional custom CA
}
```

### Step 4 — Secure defaults

- Coordinator refuses start without TLS cert/key in non-dev mode
- `--insecure` flag on coordinator allows HTTP **only** when explicitly passed (documented for local tests)

### Step 5 — Protect cache endpoint

Apply auth middleware to `/api/v1/cache` — cache bytes are not secret but downloading without auth enables abuse on public coordinators.

## Files to touch

| File | Action |
|------|--------|
| `coordinator/server.go` | TLS listen + mux |
| `coordinator/auth.go` | Bearer middleware |
| `coordinator/auth_test.go` | Token tests |
| `worker/client.go` | TLS + auth transport (stub) |

## Verification

```bash
# Should fail without token
curl -k https://localhost:8443/api/v1/cache

# Should succeed
curl -k -H "Authorization: Bearer $TOKEN" https://localhost:8443/api/v1/cache -o /tmp/cache

go test ./coordinator/... -run Auth
```

- [ ] Unauthenticated API returns 401
- [ ] `/health` returns 200 without auth
- [ ] TLS handshake succeeds with self-signed cert when CA trusted or skip-verify

## Rollback

Remove auth middleware; revert to plain HTTP test server from `net-high-05`.