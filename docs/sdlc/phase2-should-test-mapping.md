# Phase 2 Should mapping to tests

Traceability for Layer B (v0.2 Attribution): each **Should** attribution row in the [MVP delivery matrix](../specs/00-overview.md#mvp-delivery-matrix) maps to automated coverage.

## Mapping

- **SLSA-style provenance attestations**
  - `AC-PROV-001`, `FR-PROV-001`–`005`
  - Coverage:
    - `.github/workflows/publish-keyless-smoke.yml` (`Verify trust status`, `Inspect Layer B attestations`)
    - `internal/publish/publish_test.go`, `internal/provenance/provenance_test.go`

- **Source repo + commit linkage**
  - `AC-PROV-001`, `AC-PROV-004`
  - Coverage:
    - `publish-keyless-smoke` trust.json `repository`, `commit`, `workflow` fields
    - `GET /v1/namespaces/{ns}/artifacts?commit={sha}` — `internal/api/handlers_test.go` (`TestListArtifacts_byCommit`)

- **CI workflow identity in provenance**
  - `AC-PROV-001`, `FR-PROV-003`
  - Coverage:
    - `publish-keyless-smoke` trust + inspect JSON workflow checks
    - `internal/inspect/should_checklist_test.go`

- **SBOM attachment**
  - `AC-PROV-002`, `FR-PROV-007`–`008`
  - Coverage:
    - `publish-keyless-smoke` (`Publish with SPDX SBOM` step when `--sbom` used)
    - `internal/provenance/provenance_test.go` (`TestSBOMPredicateType`)

- **Provenance verification on inspect**
  - `AC-PROV-003`, `FR-PROV-006`, `FR-PROV-009`
  - Coverage:
    - `internal/api/attestations_test.go` (tampered statement/bundle)
    - `publish-keyless-smoke` (`provenance_verified` in trust.json)
    - `internal/inspect/should_checklist_test.go`

- **GitHub Actions golden path**
  - `AC-OV-003`, `FR-OV-008`, `FR-DX-001`–`004`
  - Coverage:
    - `docs/guides/github-actions-publish.md`
    - `.github/actions/verity-publish/`, `publish-keyless-smoke.yml`

- **Policy: trusted-publishers, repository-ownership, require-provenance**
  - `AC-POL-002b`, `AC-POL-003`, `FR-POL-007`, `FR-POL-012`
  - Coverage:
    - `internal/api/policy_eval_should_test.go`, `policy_eval_granular_test.go`
    - `publish-keyless-smoke` strict-release / wrong-workflow steps

- **Honest inspect (no ✓ for unevaluated checks)**
  - `NFR-OV-005`, `FR-PROV-012`
  - Coverage:
    - `internal/inspect/should_checklist_test.go` (`repository-ownership not configured`)

## How to run

```bash
make test
```

```bash
gh workflow run publish-keyless-smoke.yml
```

Notes:

- `AC-OV-003` requires `keyless-publish` as a **required status check** on `main` (see [.github/BRANCH_PROTECTION.md](../../.github/BRANCH_PROTECTION.md)).
- Release sign-off: [mvp-v0.2-release.md](mvp-v0.2-release.md).
