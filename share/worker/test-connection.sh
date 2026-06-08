#!/usr/bin/env bash
# Quick checks before running btcfind-worker. Edit or export COORDINATOR_URL and BTCFIND_AUTH_TOKEN.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
COORDINATOR_URL="${COORDINATOR_URL:-https://REPLACE_HOST:8443}"
BTCFIND_AUTH_TOKEN="${BTCFIND_AUTH_TOKEN:-REPLACE_TOKEN}"
CA="$SCRIPT_DIR/ca.pem"

if [[ "$COORDINATOR_URL" == *"REPLACE_HOST"* ]] || [[ "$BTCFIND_AUTH_TOKEN" == "REPLACE_TOKEN" ]]; then
  echo "Set COORDINATOR_URL and BTCFIND_AUTH_TOKEN in the environment or edit this script." >&2
  exit 1
fi

echo "==> Health check (no auth)"
curl -sk --cacert "$CA" "${COORDINATOR_URL%/}/health"
echo ""
echo ""

echo "==> Register test (with auth)"
curl -sk --cacert "$CA" \
  -H "Authorization: Bearer $BTCFIND_AUTH_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"hostname":"connectivity-test","threads":1,"version":"1"}' \
  "${COORDINATOR_URL%/}/api/v1/register"
echo ""
echo ""
echo "If both succeeded, run ./run-worker.sh"