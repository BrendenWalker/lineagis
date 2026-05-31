#!/usr/bin/env bash
# Resolve workflow-built lineagis CLI/API paths (linux or windows artifact layout).
lineagis_bin_path() {
  if [ -f "./bin/lineagis.exe" ]; then
    printf '%s' "./bin/lineagis.exe"
  elif [ -f "./bin/lineagis" ]; then
    printf '%s' "./bin/lineagis"
  else
    echo "FAIL: ./bin/lineagis or ./bin/lineagis.exe missing — download lineagis-binaries-linux-amd64 or lineagis-binaries-windows-amd64 from CI (see docs/guides/operator-validation.md)" >&2
    return 1
  fi
}

lineagis_api_bin_path() {
  if [ -f "./bin/lineagis-api.exe" ]; then
    printf '%s' "./bin/lineagis-api.exe"
  elif [ -f "./bin/lineagis-api" ]; then
    printf '%s' "./bin/lineagis-api"
  else
    echo "FAIL: ./bin/lineagis-api or ./bin/lineagis-api.exe missing — download lineagis-binaries-linux-amd64 or lineagis-binaries-windows-amd64 from CI (see docs/guides/operator-validation.md)" >&2
    return 1
  fi
}
