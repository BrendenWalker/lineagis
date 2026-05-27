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

## How to run

```bash
make test
```

```bash
gh workflow run publish-keyless-smoke.yml
```

Notes:

- Local smoke: `make smoke` validates stack readiness and registry connectivity.
- The workflow assertion for unsigned inspect under `require-signatures` directly covers the Must policy behavior in `AC-OV-002`.
