# MVP v0.1 release checklist

Use this checklist before tagging **v0.1.0**. It ties Must acceptance criteria to automated proof and documents known limitations.

Authoritative scope: [docs/specs/00-overview.md](../specs/00-overview.md).

## Acceptance criteria

| ID | Requirement | Proof |
|----|-------------|-------|
| AC-OV-001 | Publish stores OCI digest + semver tag | `publish-keyless-smoke` keyless publish step; `internal/publish/publish_test.go` |
| AC-OV-002 | Inspect reports signature validity; unsigned fails under `require-signatures` | `publish-keyless-smoke` inspect + unsigned steps; `cmd/verity/inspect_test.go` |
| AC-OV-004 | README Phase 1 bullets traced to passing tests | [phase1-must-test-mapping.md](phase1-must-test-mapping.md); **`keyless-publish` required on `main`** |
| AC-DX-001 | Quickstart path documented | [docs/guides/quickstart.md](../guides/quickstart.md) (local dev); [github-actions-publish.md](../guides/github-actions-publish.md) (production) |

## Required CI (branch protection)

On `main`, require these status checks (see [.github/BRANCH_PROTECTION.md](../../.github/BRANCH_PROTECTION.md)):

| Check | Workflow / job |
|-------|----------------|
| `lint` | `ci.yml` |
| `test` | `ci.yml` |
| `build` | `ci.yml` |
| `keyless-publish` | `publish-keyless-smoke.yml` |

`AC-OV-004` is satisfied only when `keyless-publish` is required and passing on every merge to `main`.

## Manual verification (release manager)

Automated proof is in CI — no local `go build` required for release sign-off:

```bash
make test
gh run list --workflow=ci.yml --limit 3          # confirm lint, test, build, operator-stack passed
gh run list --workflow=publish-keyless-smoke.yml --limit 3   # confirm keyless-publish passed
```

Optional operator-stack validation on your machine (download CI binaries, do not build from source):

```bash
# Windows: verity-binaries-windows-amd64
# Linux/WSL: verity-binaries-linux-amd64
# See docs/guides/operator-validation.md
gh run download --name verity-binaries-windows-amd64 --dir bin
bash scripts/operator-stack-ci.sh
```

## Known limitations (v0.1)

| Limitation | Notes |
|------------|-------|
| No `verity pull` | Consumers resolve by digest/tag via API + registry out-of-band |
| Configured policies on `SetTag` | All configured rules (`require-signatures`, `trusted-publishers`, `require-provenance`, `repository-ownership`) evaluated at tag time (FR-POL-012) |
| Local verify | `verity inspect` / `verity verify` default to local cosign verify; `--trust-api` opts out (FR-SIGN-005) |
| Trusted publishers | Fail-closed when rule is in policy; operator defines allowlist per namespace |
| RegisterDigest + signatures | When `require-signatures` is active, `RegisterDigest` requires a valid bundle in the same request |
| Dev token | `VERITY_DEV_TOKEN` for local compose only — disable in production |
| CVE / federation | Deferred per delivery matrix (#67, #68) |

## Operator minimums

- Serve API over **TLS** in production
- Configure **OIDC** (`VERITY_OIDC_ISSUER`, `VERITY_OIDC_AUDIENCE`); do not rely on `VERITY_DEV_TOKEN`
- Restrict policy and trusted-publisher writes to **operator** role
- Protect PostgreSQL and object storage credentials; metadata DB is in the trusted computing base
- Enable audit logging review for policy changes (`FR-POL-010`)

See [SECURITY.md](../../SECURITY.md).

## Out of scope for v0.1 tag

- Push-time enforcement for every configured policy type (`FR-POL-012`, v0.2)
- Honest inspect lines for all check types (partial; `NFR-OV-005` ongoing)
- CLI OIDC token acquisition
- CVE blocking, transparency log UI, federation

Track follow-up under **v0.2 — Attribution** and **v0.3 — Governance** milestones (see [00-overview.md](../specs/00-overview.md)).
