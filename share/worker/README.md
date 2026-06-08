# btcfind-worker — cluster worker package

You received this package to join a **btcfind** distributed search cluster as a **worker**. Your machine runs the key-generation hot loop locally; the **coordinator** (your friend's host) aggregates stats and records any hits.

You do **not** need `funded.tsv` or the full repo. On first connect the worker downloads a funded index cache (~900 MB) from the coordinator, then searches when the coordinator operator presses **Enter**.

---

## What's in this tarball

| File | Purpose |
|------|---------|
| `btcfind-worker` | Worker binary (Linux x86_64) |
| `ca.pem` | Trust anchor for the coordinator's TLS certificate |
| `run-worker.sh` | Example launcher — edit placeholders, then run |
| `test-connection.sh` | Optional curl checks before starting the worker |
| `README.md` | This file |

**Your friend must send separately (secure channel — Signal, etc.):**

1. **Coordinator URL** — e.g. `https://100.64.0.5:8443` or `https://their-host.example.com:8443`
2. **Auth token** — shared secret (`BTCFIND_AUTH_TOKEN`)

Never post the token in public chat or email.

---

## Requirements

- **Linux x86_64** (same as this binary). Other platforms: build from source (ask your friend for the repo).
- **Outbound HTTPS** to the coordinator host/port (no inbound ports needed on your side).
- **~2 GB free disk** (cache file + temp download).
- **curl** (optional, for `test-connection.sh`).

---

## Quick start

### 1. Extract

```bash
tar xzf btcfind-worker-share.tar.gz
cd btcfind-worker-share
chmod +x btcfind-worker run-worker.sh test-connection.sh
```

### 2. Edit `run-worker.sh`

Open the file and set:

- `COORDINATOR_URL` — from your friend
- `BTCFIND_AUTH_TOKEN` — from your friend
- `THREADS` — CPU threads to use (see below)

### 3. Test connectivity (recommended)

```bash
./test-connection.sh
```

Both checks should succeed before you run the worker.

### 4. Run the worker

```bash
./run-worker.sh
```

Or manually:

```bash
export BTCFIND_AUTH_TOKEN="token-from-friend"

./btcfind-worker \
  --coordinator "https://COORDINATOR_HOST:8443" \
  --tls-ca ./ca.pem \
  --auth-token "$BTCFIND_AUTH_TOKEN" \
  --threads 16 \
  --verbose
```

### 5. Wait for coordinator

When connected you'll see:

```
worker: registered id=w-… state=lobby
worker: connected — waiting for coordinator to press Enter to start
```

**This is normal.** The worker does nothing until your friend presses **Enter** on the coordinator. After that:

```
worker: search started
```

### 6. Stop

Press **Ctrl+C**. The worker exits cleanly.

---

## Choosing `--threads`

| Machine | Suggested |
|---------|-----------|
| Laptop, 4 cores | `--threads 4` |
| Desktop, 8 cores | `--threads 8` |
| Desktop, 16 cores | `--threads 16` |
| Server, 32+ cores | `--threads 32` (or match physical cores) |

`--threads 0` (default) uses all logical CPUs (`NumCPU()`). The coordinator does **not** override this — you choose per machine.

---

## All flags

```
Usage: btcfind-worker [flags]

Required:
  --coordinator URL     Coordinator base URL (https://host:port)

Auth (one required):
  --auth-token TOKEN    Bearer token (same as coordinator)
  BTCFIND_AUTH_TOKEN    Environment variable alternative

Performance:
  --threads N           Local worker threads (0 = all CPUs)

TLS:
  --tls-ca PATH         CA file to verify coordinator (use ./ca.pem)
  --tls-skip-verify     Skip TLS verify (debug only — insecure)
  --tls-cert PATH       Client cert (only if coordinator requires mTLS)
  --tls-key PATH        Client key (mTLS)

Network:
  --proxy-url URL       HTTP CONNECT proxy (corporate networks)
  --dial-timeout DUR    Connection timeout (default 30s)

Output:
  --verbose             Log connect, cache sync, lobby status
  --quiet               Minimal output (errors still shown)

Advanced / testing:
  --stats-interval DUR  Stats report interval (default 2s)
  --simulate-hit        Force a test hit (diagnostics)
  --simulate-hit-at N   Key index for simulate hit (default 1)
```

View built-in help:

```bash
./btcfind-worker --help
```

(pass a dummy `--coordinator` if needed)

---

## What happens under the hood

1. **Register** — worker joins the coordinator lobby.
2. **Cache sync** — downloads `funded.cache` (~900 MB) if missing or outdated; skips if already current (304).
3. **Lobby** — heartbeats every 10s; waits for `running` state.
4. **Search** — full local hot loop; zero per-key network I/O.
5. **Stats** — batches keys/sec to coordinator every ~2s.
6. **Hits** — rare; WIF sent to coordinator over TLS if a funded match occurs.

---

## Troubleshooting

### `auth token required`

Token not set. In the **same shell**:

```bash
export BTCFIND_AUTH_TOKEN="exact-token-from-friend"
echo "$BTCFIND_AUTH_TOKEN"   # must not be empty
```

Or pass `--auth-token "..."` directly.

---

### `connection refused` (retry loop)

- Coordinator is not running.
- Wrong host or port in `--coordinator`.
- Friend's firewall/router blocks the port (they must forward **TCP** to coordinator).

Ask friend to run: `curl -sk https://127.0.0.1:8443/health` on their machine (should print `ok`).

---

### `http 401` / unauthorized

Token mismatch. You and coordinator must use the **identical** token string.

---

### TLS / `x509: certificate signed by unknown authority`

Use the bundled CA:

```bash
--tls-ca ./ca.pem
```

Do **not** use `--tls-skip-verify` on the public internet unless debugging.

If still failing, coordinator URL hostname must match their certificate (ask friend to regenerate cert with correct hostname/IP).

---

### Worker sits at `waiting for coordinator to press Enter`

**Success — you're connected.** Tell your friend to press **Enter** on the coordinator terminal.

---

### Stuck on `downloading funded cache`

First run downloads ~900 MB. With `--verbose` you'll see percent progress. Same-machine LAN: 1–5 minutes. Internet: depends on bandwidth.

Ensure ~2 GB free disk space.

---

### `registration rejected (cluster full?)`

Coordinator hit `--max-workers` limit. Ask friend to raise it or disconnect another worker.

---

### Corporate proxy / strict egress

```bash
./btcfind-worker --coordinator ... --proxy-url http://proxy.company:8080 ...
```

Or use Tailscale (friend sets up a private mesh — no public port).

---

### Binary won't run (`Exec format error`)

This package is **Linux x86_64** only. On ARM Mac, Windows, etc., build from source:

```bash
git clone <repo-url> && cd Craigpwn
go build -o btcfind-worker ./cmd/btcfind-worker
```

(Go 1.25+ required.)

---

## Manual connectivity tests

Replace placeholders:

```bash
export COORD="https://YOUR_FRIEND_HOST:8443"
export TOKEN="your-token"

# 1. Liveness (no auth)
curl -sk --cacert ca.pem "$COORD/health"
# expect: ok

# 2. Registration (auth required)
curl -sk --cacert ca.pem \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"hostname":"my-pc","threads":8,"version":"1"}' \
  "$COORD/api/v1/register"
# expect: JSON with "worker_id":"w-..."
```

---

## Security notes

- **`ca.pem`** is public (trust anchor). Safe to bundle.
- **Auth token** is a password — treat like an API key.
- **Never** share your friend's `server-key.pem` or `ca-key.pem`.
- On hit, **WIF** crosses the network once (TLS). Only join clusters you trust.
- Prefer **Tailscale/WireGuard** between friends instead of exposing coordinator on the raw public internet.

---

## Typical session (copy-paste)

```bash
cd btcfind-worker-share
export BTCFIND_AUTH_TOKEN="paste-token-here"

./btcfind-worker \
  --coordinator "https://paste-host-here:8443" \
  --tls-ca ./ca.pem \
  --auth-token "$BTCFIND_AUTH_TOKEN" \
  --threads 16 \
  --verbose
```

Leave running until friend stops the cluster or you Ctrl+C.

---

## Questions?

Contact your friend (the coordinator operator). They control start/stop, hit logs (`wallets.txt`), and cluster settings (`--formats`, `--min-balance`).