#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:${VERITY_API_PORT:-8080}}"
REGISTRY_URL="${REGISTRY_URL:-http://localhost:5000}"
MINIO_URL="${MINIO_URL:-http://localhost:9000}"
POSTGRES_HOST="${POSTGRES_HOST:-localhost}"
POSTGRES_PORT="${POSTGRES_PORT:-5432}"
POSTGRES_USER="${POSTGRES_USER:-verity}"
POSTGRES_DB="${POSTGRES_DB:-verity}"

pass() { echo "PASS: $*" >&2; }
fail() { echo "FAIL: $*" >&2; exit 1; }

curl_ok() {
  local url="$1"
  local label="$2"
  local body
  body="$(curl -fsS "$url")" || fail "$label unreachable at $url"
  pass "$label ($url)"
  printf '%s' "$body"
}

echo "=== Verity operator stack smoke (AC-ARCH-001) ==="

health_body="$(curl_ok "$API_URL/healthz" "host -> api /healthz")"
[ "$health_body" = "ok" ] || fail "unexpected /healthz body: $health_body"

ready_body="$(curl_ok "$API_URL/readyz" "host -> api /readyz (api -> db, api -> registry)")"
[ "$ready_body" = "ok" ] || fail "unexpected /readyz body: $ready_body"

curl_ok "$REGISTRY_URL/v2/" "host -> registry"

curl_ok "$MINIO_URL/minio/health/live" "host -> minio"

if command -v pg_isready >/dev/null 2>&1; then
  pg_isready -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null \
    || fail "postgres not ready at $POSTGRES_HOST:$POSTGRES_PORT"
  pass "host -> postgres ($POSTGRES_HOST:$POSTGRES_PORT)"
elif command -v docker >/dev/null 2>&1; then
  docker compose exec -T postgres pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB" >/dev/null \
    || fail "postgres not ready in compose"
  pass "host -> postgres (via docker compose exec)"
else
  echo "SKIP: pg_isready and docker not available; postgres hop not verified"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REGISTRY_URL="$REGISTRY_URL" bash "$SCRIPT_DIR/smoke-registry.sh"
pass "registry -> s3 (push/pull via crane)"

if [ -x "./bin/verity" ] || [ -x "./bin/verity.exe" ]; then
  verity_bin="./bin/verity"
  [ -x "./bin/verity.exe" ] && verity_bin="./bin/verity.exe"
  "$verity_bin" --version >/dev/null || fail "verity cli --version failed"
  pass "cli available ($verity_bin --version)"
else
  echo "SKIP: bin/verity not built; run make build for cli hop"
fi

echo "=== all smoke checks passed ==="
