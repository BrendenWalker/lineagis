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

```bash
make test
make build
gh workflow run publish-keyless-smoke.yml   # or confirm latest PR run passed
```

Optional local stack:

```bash
make smoke
```

## Known limitations (v0.1)

| Limitation | Notes |
|------------|-------|
| Inspect trusts the Verity API | No local cosign verification in CLI (`OQ-ARCH-002` deferred) |
| No `verity pull` | Consumers resolve by digest/tag via API + registry out-of-band |
| Should policies verify-time only | `trusted-publishers`, `repository-ownership` do not block `SetTag` |
| Unsigned digest registration | OCI push + `RegisterDigest` can succeed before sign; `require-signatures` blocks **tagging** |
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

- Push-time enforcement of Should policies
- CLI OIDC token acquisition
- CVE blocking, transparency log UI, federation

Track follow-up work under the **v0.1 hardening / post-v0.1** milestone on GitHub.
