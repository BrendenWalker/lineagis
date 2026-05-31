#!/usr/bin/env bash
# AC-ARCH-001: full operator stack smoke using workflow-built binaries (not Dockerfile API).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=lib/lineagis-bin.sh
source "$SCRIPT_DIR/lib/lineagis-bin.sh"

LINEAGIS_BIN="$(lineagis_bin_path)"
LINEAGIS_API_BIN="$(lineagis_api_bin_path)"
chmod +x "$LINEAGIS_BIN" "$LINEAGIS_API_BIN" 2>/dev/null || true

test -f .env || cp .env.example .env

echo "=== Starting infra (postgres, minio, registry) ==="
docker compose up -d postgres minio minio-init registry --wait

export LINEAGIS_DATABASE_URL="${LINEAGIS_DATABASE_URL:-postgres://lineagis:lineagis@localhost:5432/lineagis?sslmode=disable}"
export LINEAGIS_REGISTRY_URL="${LINEAGIS_REGISTRY_URL:-http://localhost:5000}"
export LINEAGIS_API_ADDR="${LINEAGIS_API_ADDR:-:8080}"
export LINEAGIS_MIGRATE_ON_STARTUP="${LINEAGIS_MIGRATE_ON_STARTUP:-true}"
export LINEAGIS_DEV_TOKEN="${LINEAGIS_DEV_TOKEN:-dev-local-token}"
export LINEAGIS_LOG_LEVEL="${LINEAGIS_LOG_LEVEL:-info}"
export LINEAGIS_LOG_FORMAT="${LINEAGIS_LOG_FORMAT:-text}"

API_PORT="${LINEAGIS_API_PORT:-8080}"
api_pid=""
cleanup() {
  if [ -n "$api_pid" ]; then
    kill "$api_pid" 2>/dev/null || true
    wait "$api_pid" 2>/dev/null || true
  fi
  docker compose down -v --remove-orphans 2>/dev/null || docker compose down --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "=== Starting lineagis-api from ${LINEAGIS_API_BIN} ==="
"$LINEAGIS_API_BIN" &
api_pid=$!

for i in $(seq 1 60); do
  if curl -sf "http://localhost:${API_PORT}/readyz" | grep -qx ok; then
    break
  fi
  if [ "$i" -eq 60 ]; then
    echo "FAIL: lineagis-api not ready on port ${API_PORT}" >&2
    exit 1
  fi
  sleep 2
done

bash scripts/smoke-stack.sh
