#!/usr/bin/env bash
# Resolve workflow-built verity CLI/API paths (linux or windows artifact layout).
verity_bin_path() {
  if [ -f "./bin/verity.exe" ]; then
    printf '%s' "./bin/verity.exe"
  elif [ -f "./bin/verity" ]; then
    printf '%s' "./bin/verity"
  else
    echo "FAIL: ./bin/verity or ./bin/verity.exe missing — download verity-binaries-linux-amd64 or verity-binaries-windows-amd64 from CI (see docs/guides/operator-validation.md)" >&2
    return 1
  fi
}

verity_api_bin_path() {
  if [ -f "./bin/verity-api.exe" ]; then
    printf '%s' "./bin/verity-api.exe"
  elif [ -f "./bin/verity-api" ]; then
    printf '%s' "./bin/verity-api"
  else
    echo "FAIL: ./bin/verity-api or ./bin/verity-api.exe missing — download verity-binaries-linux-amd64 or verity-binaries-windows-amd64 from CI (see docs/guides/operator-validation.md)" >&2
    return 1
  fi
}
