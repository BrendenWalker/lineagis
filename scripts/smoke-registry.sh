#!/usr/bin/env bash
set -euo pipefail

REGISTRY_URL="${REGISTRY_URL:-http://localhost:5000}"
IMAGE="${SMOKE_IMAGE:-localhost:5000/smoke/test:latest}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for registry smoke" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

cat >"$tmpdir/Dockerfile" <<'EOF'
FROM scratch
COPY empty /empty
EOF
: >"$tmpdir/empty"

docker build -t "$IMAGE" "$tmpdir" >/dev/null

if command -v crane >/dev/null 2>&1; then
  docker save "$IMAGE" -o "$tmpdir/image.tar"
  crane push "$tmpdir/image.tar" "$IMAGE" --insecure
  crane pull "$IMAGE" "$tmpdir/pulled.tar" --insecure
elif docker push "$IMAGE" >/dev/null; then
  docker pull "$IMAGE" >/dev/null
  docker save "$IMAGE" -o "$tmpdir/pulled.tar"
else
  echo "registry push failed; install crane or configure docker for insecure localhost:5000" >&2
  exit 1
fi

if [ ! -s "$tmpdir/pulled.tar" ]; then
  echo "failed to pull pushed image" >&2
  exit 1
fi

echo "registry smoke ok: pushed and pulled $IMAGE via ${REGISTRY_URL}"
