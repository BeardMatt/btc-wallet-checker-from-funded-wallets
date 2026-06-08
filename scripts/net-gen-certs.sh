#!/usr/bin/env bash
# Generate CA, server, and worker TLS certs for btcfind cluster mTLS.
set -euo pipefail

OUT="${1:-./certs}"
SAN="${2:-localhost}"
EXTRA_IP="${3:-}"

mkdir -p "$OUT"

SAN_EXT="subjectAltName=DNS:${SAN},DNS:localhost,IP:127.0.0.1"
if [[ -n "$EXTRA_IP" ]]; then
  SAN_EXT="${SAN_EXT},IP:${EXTRA_IP}"
fi

openssl genrsa -out "$OUT/ca-key.pem" 4096 2>/dev/null
openssl req -x509 -new -nodes -key "$OUT/ca-key.pem" -sha256 -days 3650 \
  -out "$OUT/ca.pem" -subj "/CN=btcfind-cluster-ca"

openssl genrsa -out "$OUT/server-key.pem" 4096 2>/dev/null
openssl req -new -key "$OUT/server-key.pem" -out "$OUT/server.csr" \
  -subj "/CN=btcfind-coordinator"
openssl x509 -req -in "$OUT/server.csr" -CA "$OUT/ca.pem" -CAkey "$OUT/ca-key.pem" \
  -CAcreateserial -out "$OUT/server.pem" -days 825 -sha256 \
  -extfile <(printf "%s" "$SAN_EXT")

openssl genrsa -out "$OUT/worker-key.pem" 4096 2>/dev/null
openssl req -new -key "$OUT/worker-key.pem" -out "$OUT/worker.csr" \
  -subj "/CN=btcfind-worker"
openssl x509 -req -in "$OUT/worker.csr" -CA "$OUT/ca.pem" -CAkey "$OUT/ca-key.pem" \
  -CAcreateserial -out "$OUT/worker.pem" -days 825 -sha256

rm -f "$OUT"/*.csr "$OUT"/ca.srl

echo "Wrote certs to $OUT:"
echo "  Coordinator: --tls-cert $OUT/server.pem --tls-key $OUT/server-key.pem"
echo "  Coordinator mTLS: --mtls-ca $OUT/ca.pem --mtls-require-client-cert"
echo "  Worker: --tls-ca $OUT/ca.pem --tls-cert $OUT/worker.pem --tls-key $OUT/worker-key.pem"