# Phase 1 Must Mapping to Tests

Traceability for `AC-OV-004`: each README Phase 1 bullet maps to automated coverage.

Phase 1 bullets in `README.md`:

- OCI-native artifact publishing
- metadata persistence
- CLI workflows
- signature support

## Mapping

- **OCI-native artifact publishing**
  - `AC-OV-001`
  - Coverage:
    - `.github/workflows/publish-keyless-smoke.yml` (`Keyless publish` step)
    - `internal/publish/publish_test.go` (stable digest + repo layout)
    - `internal/registry/integration_test.go` (registry integration)

- **metadata persistence**
  - `AC-OV-001`
  - Coverage:
    - `internal/metadata/integration_test.go`
    - `internal/db/migrate_integration_test.go`
    - `.github/workflows/publish-keyless-smoke.yml` (`Verify trust status` API checks)

- **CLI workflows**
  - `AC-OV-001`, `AC-OV-002`
  - Coverage:
    - `cmd/verity/main_test.go`
    - `cmd/verity/inspect_test.go`
    - `.github/workflows/publish-keyless-smoke.yml` (`Inspect signed artifact via CLI`)

- **signature support**
  - `AC-OV-002`
  - Coverage:
    - `internal/signing/verify_test.go`
    - `internal/api/trust_signatures_test.go`
    - `.github/workflows/publish-keyless-smoke.yml`
      - `Verify signature attached`
      - `Inspect unsigned artifact fails under policy`
      - `Apply strict-release policy` / wrong-workflow push block / verify fail (`AC-POL-002b`, `AC-OV-005`)

- **Operator stack topology**
  - `AC-ARCH-001`
  - Coverage:
    - `.github/workflows/ci.yml` (`operator-stack` job + `scripts/operator-stack-ci.sh`)

## How to run

```bash
make test
```

```bash
gh workflow run publish-keyless-smoke.yml
```

Notes:

- **Operator machine validation:** download `verity-binaries-windows-amd64` (Windows) or `verity-binaries-linux-amd64` (Linux/WSL) from the CI `build` job — do not `go build` from source. See [operator-validation.md](../guides/operator-validation.md).
- `make smoke` remains a contributor convenience (builds API via Dockerfile); CI `operator-stack` is the authoritative AC-ARCH-001 check using workflow-built binaries.
- The workflow assertion for unsigned inspect under `require-signatures` directly covers the Must policy behavior in `AC-OV-002`.
- **AC-OV-004** requires the `keyless-publish` job in `publish-keyless-smoke.yml` to be a **required status check** on `main` (see [.github/BRANCH_PROTECTION.md](../../.github/BRANCH_PROTECTION.md)).
