#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GOEXE="$(go env GOEXE)"
BIN="${ROOT}/bin/lineagis${GOEXE}"
OUT="${ROOT}/dist/release-artifacts"
rm -rf "${OUT}"
mkdir -p "${OUT}"

if [[ ! -x "${BIN}" ]]; then
  chmod +x "${BIN}"
fi

export LINEAGIS_GRAPH_FILE="${OUT}/.graph.json"
"${BIN}" analyze . --graph-out "${LINEAGIS_GRAPH_FILE}" --out "${OUT}/generated" --validate-arch
cp "${LINEAGIS_GRAPH_FILE}" "${OUT}/lineage.json"

for f in \
  "${OUT}/lineage.json" \
  "${OUT}/generated/architecture/overview.md" \
  "${OUT}/generated/reports/dependency-report.md" \
  "${OUT}/generated/diagrams/imports.dot"
do
  if [[ ! -f "${f}" ]]; then
    echo "smoke-release-artifacts: missing ${f}" >&2
    exit 1
  fi
done

echo "smoke-release-artifacts: ok"
