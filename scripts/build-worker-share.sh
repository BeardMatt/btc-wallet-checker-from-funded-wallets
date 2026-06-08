#!/usr/bin/env bash
# Build btcfind-worker-share.tar.gz for distribution to remote workers.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="$ROOT/btcfind-worker-share"
TARBALL="${1:-$HOME/Desktop/btcfind-worker-share.tar.gz}"

cd "$ROOT"
go build -o btcfind-worker ./cmd/btcfind-worker

if [[ ! -f certs/ca.pem ]]; then
  echo "error: certs/ca.pem missing — run ./scripts/net-gen-certs.sh ./certs <hostname> [lan-ip]" >&2
  exit 1
fi

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"
cp btcfind-worker "$OUT_DIR/"
cp certs/ca.pem "$OUT_DIR/"
cp share/worker/README.md share/worker/run-worker.sh share/worker/test-connection.sh "$OUT_DIR/"
chmod +x "$OUT_DIR/btcfind-worker" "$OUT_DIR/run-worker.sh" "$OUT_DIR/test-connection.sh"

tar czf "$TARBALL" -C "$ROOT" btcfind-worker-share
ls -lh "$TARBALL"
echo "Wrote $TARBALL"
echo "Send tarball + auth token + coordinator URL to your friend (see share/worker/README.md)."