#!/usr/bin/env bash
# Run the standard btcfind benchmark and print parseable metrics.
# Usage: ./.grok/skills/benchmark/scripts/run-benchmark.sh

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
cd "$ROOT"

go build -o btcfind . 2>&1

START=$(date +%s.%N)
OUTPUT=$(./btcfind 50000 8 2>&1)
END=$(date +%s.%N)

WALL=$(python3 -c "print(round($END - $START, 3))")
KEYS_PER_SEC=$(echo "$OUTPUT" | sed -n 's/.*Average \([0-9.]*\) keys per second.*/\1/p')
HOT_LOOP=$(echo "$OUTPUT" | sed -n 's/.*Took \([0-9.]*\)s\.\.\. Average.*/\1/p')
STARTUP=$(python3 -c "print(round($WALL - float('$HOT_LOOP' or 0), 3))")

echo "$OUTPUT"
echo "---"
echo "BENCHMARK_JSON_START"
python3 -c "
import json
print(json.dumps({
    'command': './btcfind 50000 8',
    'num_keys': 50000,
    'threads': 8,
    'keys_per_second': float('$KEYS_PER_SEC' or 0),
    'hot_loop_seconds': float('$HOT_LOOP' or 0),
    'wall_seconds': $WALL,
    'startup_seconds': $STARTUP,
}))
"
echo "BENCHMARK_JSON_END"