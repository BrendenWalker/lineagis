# MVP v0.2 release checklist

Use this checklist before tagging **v0.2.0**. It extends [mvp-v0.1-release.md](mvp-v0.1-release.md) with Layer B — Attribution (**Should** items).

Authoritative scope: [docs/specs/00-overview.md](../specs/00-overview.md).

## Acceptance criteria

| ID | Requirement | Proof |
|----|-------------|-------|
| AC-OV-003 | GitHub Actions golden path (publish → verify → inspect) | `publish-keyless-smoke`; [github-actions-publish.md](../guides/github-actions-publish.md); [phase2-should-test-mapping.md](phase2-should-test-mapping.md) |
| AC-PROV-001 | Provenance shows repository, commit, workflow after keyless publish | `publish-keyless-smoke` trust.json assertions |
| AC-PROV-002 | SBOM attached path when `--sbom` used | `publish-keyless-smoke` SPDX step; `internal/provenance/provenance_test.go` |
| AC-PROV-003 | Tampered provenance fails verification | `internal/api/attestations_test.go` |
| AC-PROV-004 | Query artifacts by commit SHA | `GET …/artifacts?commit=`; `handlers_test.go` `TestListArtifacts_byCommit` |
| AC-POL-002b | Trusted-publishers blocks wrong workflow | `publish-keyless-smoke` strict-release steps |
| AC-POL-003 | Repository-ownership mismatch fails when configured | `internal/api/policy_eval_should_test.go` |

## Required CI (branch protection)

On `main`, require these status checks (unchanged from v0.1):

| Check | Workflow / job |
|-------|----------------|
| `lint` | `ci.yml` |
| `test` | `ci.yml` |
| `build` | `ci.yml` |
| `keyless-publish` | `publish-keyless-smoke.yml` |

`AC-OV-003` and `AC-OV-004` (Phase 2 traceability) require `keyless-publish` passing on every merge to `main`.

## Manual verification (release manager)

```bash
make test
gh run list --workflow=ci.yml --limit 3
gh run list --workflow=publish-keyless-smoke.yml --limit 3
```

Confirm smoke output includes `provenance_verified: true` and inspect shows Layer B lines per [00-overview.md](../specs/00-overview.md#inspect--verify-flow).

## Errata (v0.1 docs)

| v0.1 statement | v0.2 correction |
|----------------|-----------------|
| `FR-POL-012` listed as v0.2 follow-up | **Implemented** — all configured rules evaluated on `SetTag` (see `internal/api/policy_eval.go`, smoke wrong-workflow step) |
| README inspect example v0.1-only | Updated to Layer B target in [README.md](../../README.md) |
| Attestation lines “informational until provenance verify ships” | Provenance verify on inspect is **Should-complete** for v0.2 |

## Known limitations (v0.2)

| Limitation | Notes |
|------------|-------|
| SLSA level | Build **L1** only (OQ-PROV-001); reproducible builds deferred |
| SBOM | Optional on publish (OQ-PROV-002); warn on inspect when missing |
| Repository ownership | Provenance + namespace match only — no GitHub API (OQ-PROV-004) |
| `verity pull` / CLI OIDC | Deferred DX items, not Attribution theme |
| CVE / federation / transparency log | Phase 3 Deferred (#67, #68) |

## Out of scope for v0.2 tag

- **Layer C (v0.3):** webhooks, deeper governance
- **Deferred:** CVE blocking, federation, transparency-log UX
- **DX Should:** `verity login`, `verity pull`, `verity policy` CLI

## Related

- [layer-b-v0.2-plan.md](layer-b-v0.2-plan.md) — implementation plan
- [phase2-should-test-mapping.md](phase2-should-test-mapping.md) — traceability
- [mvp-v0.1-release.md](mvp-v0.1-release.md) — v0.1 bar
