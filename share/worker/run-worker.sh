#!/usr/bin/env bash
# Edit the three settings below, then: ./run-worker.sh

set -euo pipefail

# --- required: from coordinator operator ---
COORDINATOR_URL="${COORDINATOR_URL:-https://REPLACE_HOST:8443}"
BTCFIND_AUTH_TOKEN="${BTCFIND_AUTH_TOKEN:-REPLACE_TOKEN}"

# --- optional ---
THREADS="${THREADS:-0}"   # 0 = use all CPUs
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

if [[ "$COORDINATOR_URL" == *"REPLACE_HOST"* ]]; then
  echo "error: edit run-worker.sh and set COORDINATOR_URL (or export it)" >&2
  exit 1
fi
if [[ "$BTCFIND_AUTH_TOKEN" == "REPLACE_TOKEN" ]]; then
  echo "error: edit run-worker.sh and set BTCFIND_AUTH_TOKEN (or export it)" >&2
  exit 1
fi

export BTCFIND_AUTH_TOKEN

exec "$SCRIPT_DIR/btcfind-worker" \
  --coordinator "$COORDINATOR_URL" \
  --tls-ca "$SCRIPT_DIR/ca.pem" \
  --auth-token "$BTCFIND_AUTH_TOKEN" \
  --threads "$THREADS" \
  --verbose