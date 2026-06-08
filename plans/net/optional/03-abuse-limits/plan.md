# Plan: Public coordinator abuse limits (optional)

| Field | Value |
|-------|-------|
| **ID** | `net-optional-03` |
| **Priority** | Optional |
| **Status** | `pending` |
| **Depends on** | `net-high-04` |
| **Estimated gain** | Protect public coordinators from cache scraping / DoS |

## Problem

Internet-facing coordinators expose a ~900MB cache download and registration endpoints. Abuse can exhaust bandwidth, disk read I/O, and CPU.

## Goal

Optional rate limits and connection caps for coordinators exposed on the public internet.

## Scope (this plan only)

- `--max-workers N` (default unlimited)
- Per-IP rate limit on `/register` and `/cache` (token bucket)
- Optional IP allowlist `--allow-cidr`
- Reject registration when cluster full

**Out of scope:** DDoS protection at CDN layer (document Cloudflare/nginx fronting).

## Implementation steps

### Step 1 — Worker cap

```go
if len(registry) >= cfg.MaxWorkers {
    http.Error(w, "cluster full", http.StatusServiceUnavailable)
}
```

### Step 2 — Rate limiter middleware

Per-IP limits (defaults):

| Endpoint | Limit |
|----------|-------|
| `/register` | 10/min |
| `/cache` | 2/hour per IP |
| `/stats`, `/hit` | 120/min |

Use in-memory map + cleanup goroutine (single coordinator instance).

### Step 3 — IP allowlist

```bash
--allow-cidr 10.0.0.0/8,192.168.1.0/24
```

When set, bypass rate limits for allowed CIDRs.

### Step 4 — Logging

Log blocked requests at info level with IP + endpoint.

### Step 5 — Documentation

Recommend VPN or Tailscale for trusted workers instead of public exposure.

## Files to touch

| File | Action |
|------|--------|
| `coordinator/limits.go` | Rate limit + cap |
| `coordinator/server.go` | Middleware chain |
| `coordinator/limits_test.go` | Token bucket tests |

## Verification

- [ ] 11th register/min from same IP → 429
- [ ] `--max-workers 2` rejects third worker with 503
- [ ] Allowlisted IP bypasses rate limit
- [ ] Legitimate 3-worker cluster unaffected with defaults

## Rollback

Remove limits middleware; open registration from `net-high-02`.