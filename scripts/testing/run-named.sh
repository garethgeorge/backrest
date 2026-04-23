#!/usr/bin/env bash
#
# Run a named instance of backrest with its own data directory and ports.
# Multiple instances can run side-by-side for testing sync, multihost, etc.
#
# Usage: ./run-named.sh <name> [backend-port] [vite-port]
#
# Examples:
#   ./run-named.sh alice            # backend :9901, vite :5181
#   ./run-named.sh bob              # backend :9902, vite :5182
#   ./run-named.sh alice 9910 5190  # explicit ports
#
# Data is stored in /tmp/backrest-<name>/ and persists across runs.

set -euo pipefail

BASEDIR="$(cd "$(dirname "$0")/../.." && pwd)"
NAME="${1:?Usage: $0 <name> [backend-port] [vite-port]}"

# Derive deterministic ports from name if not provided.
# Hash the name to a number in a small range to avoid collisions.
name_hash() {
  printf '%s' "$1" | cksum | awk '{print $1 % 100}'
}

OFFSET=$(name_hash "$NAME")
BACKEND_PORT="${2:-$((9900 + OFFSET))}"
VITE_PORT="${3:-$((5180 + OFFSET))}"

DATADIR="/tmp/backrest-${NAME}"
mkdir -p "$DATADIR"

PIDS=()

cleanup() {
  echo ""
  echo "Shutting down instance '$NAME'..."
  for pid in "${PIDS[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
  echo "Done."
}

trap cleanup EXIT INT TERM

echo "=== backrest instance: $NAME ==="
echo "  data dir:     $DATADIR"
echo "  backend:      http://127.0.0.1:${BACKEND_PORT}"
echo "  webui (vite): http://localhost:${VITE_PORT}"
echo ""

# Start the Go backend
(
  cd "$BASEDIR"
  go run ./cmd/backrest \
    -bind-address "127.0.0.1:${BACKEND_PORT}" \
    -config-file "${DATADIR}/config.json" \
    -data-dir "${DATADIR}/data"
) &
PIDS+=($!)

# Start the vite dev server pointing at this backend
(
  cd "$BASEDIR/webui"
  UI_BACKEND_URL="http://127.0.0.1:${BACKEND_PORT}" \
    npx vite --port "$VITE_PORT" --strictPort
) &
PIDS+=($!)

# Wait for any child to exit — if one dies, the trap cleans up the other.
wait -n 2>/dev/null || true
