#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOEXE="$(go env GOEXE)"
BIN="${ROOT}/bin/lineagis${GOEXE}"
OUT="${ROOT}/.lineagis-smoke/analyze.json"
rm -rf "${ROOT}/.lineagis-smoke"
mkdir -p "${ROOT}/.lineagis-smoke"

if [[ ! -x "${BIN}" ]]; then
  chmod +x "${BIN}"
fi

"${BIN}" analyze . --format json --validate-arch > "${OUT}"

grep -q '"schema_version"' "${OUT}"
grep -q '"lineage-graph/v2"' "${OUT}"
grep -q '"nodes"' "${OUT}"
grep -q '"edges"' "${OUT}"

echo "smoke-analyze: ok"
