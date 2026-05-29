#!/usr/bin/env bash
# AC-ARCH-001: full operator stack smoke using workflow-built binaries (not Dockerfile API).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/verity-bin.sh
source "$SCRIPT_DIR/lib/verity-bin.sh"

VERITY_BIN="$(verity_bin_path)"
VERITY_API_BIN="$(verity_api_bin_path)"
chmod +x "$VERITY_BIN" "$VERITY_API_BIN" 2>/dev/null || true

test -f .env || cp .env.example .env

echo "=== Starting infra (postgres, minio, registry) ==="
docker compose up -d postgres minio minio-init registry --wait

export VERITY_DATABASE_URL="${VERITY_DATABASE_URL:-postgres://verity:verity@localhost:5432/verity?sslmode=disable}"
export VERITY_REGISTRY_URL="${VERITY_REGISTRY_URL:-http://localhost:5000}"
export VERITY_API_ADDR="${VERITY_API_ADDR:-:8080}"
export VERITY_MIGRATE_ON_STARTUP="${VERITY_MIGRATE_ON_STARTUP:-true}"
export VERITY_DEV_TOKEN="${VERITY_DEV_TOKEN:-dev-local-token}"
export VERITY_LOG_LEVEL="${VERITY_LOG_LEVEL:-info}"
export VERITY_LOG_FORMAT="${VERITY_LOG_FORMAT:-text}"

API_PORT="${VERITY_API_PORT:-8080}"
api_pid=""
cleanup() {
  if [ -n "$api_pid" ]; then
    kill "$api_pid" 2>/dev/null || true
    wait "$api_pid" 2>/dev/null || true
  fi
  docker compose down -v --remove-orphans 2>/dev/null || docker compose down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "=== Starting verity-api from ${VERITY_API_BIN} ==="
"$VERITY_API_BIN" &
api_pid=$!

for i in $(seq 1 60); do
  if curl -sf "http://localhost:${API_PORT}/readyz" | grep -qx ok; then
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "FAIL: verity-api not ready on port ${API_PORT}" >&2
    exit 1
  fi
  sleep 2
done

bash scripts/smoke-stack.sh
