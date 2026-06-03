#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
# Use go env GOEXE so Windows MSYS/PowerShell builds share one binary name.
GOEXE="$(go env GOEXE)"
BIN="${ROOT}/bin/lineagis${GOEXE}"
GRAPH="${ROOT}/.lineagis-smoke/graph.json"
rm -rf "${ROOT}/.lineagis-smoke"
export LINEAGIS_GRAPH_FILE="${GRAPH}"

if [[ ! -x "${BIN}" ]]; then
  chmod +x "${BIN}"
fi

"${BIN}" ingest \
  "${ROOT}/examples/sbom-cyclonedx.json" \
  "${ROOT}/examples/build-sidecar.json" \
  "${ROOT}/examples/commit-sidecar.json"

"${BIN}" trace artifact@sha256:abc123 --format json | grep -q lineage-trace/v1
"${BIN}" why artifact@sha256:abc123
"${BIN}" visualize artifact@sha256:abc123 --format dot | grep -q digraph

echo "smoke-lineage: ok"
