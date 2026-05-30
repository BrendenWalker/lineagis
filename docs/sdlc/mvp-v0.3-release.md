# MVP v0.3 release checklist

Use this checklist before tagging **v0.3.0**. It extends [mvp-v0.2-release.md](mvp-v0.2-release.md) with Layer C — Governance (**Should** items).

Authoritative scope: [docs/specs/00-overview.md](../specs/00-overview.md), [layer-c-v0.3-plan.md](layer-c-v0.3-plan.md).

## Acceptance criteria

| ID | Requirement | Proof |
|----|-------------|-------|
| AC-DX-PULL-001 | Pull by digest matches registry content | `internal/pull/pull_test.go`; manual `verity pull` |
| AC-DX-PULL-002 | Pull `--verify` fails on unsigned when policy requires signatures | `internal/pull/pull_test.go` (`TestPull_withVerify_failsWithoutSignatures`); `cmd/verity/pull_test.go` |
| AC-POL-006 | GitHub API ownership fail-closed when API unavailable | `internal/api/policy_eval_github_test.go`; `internal/api/policy_eval_v03_test.go` (`TestVerify_repositoryOwnership_*`) |
| AC-POL-007 | `require-digest-on-verify` blocks tag-only verify | `internal/api/policy_eval_v03_test.go` (`TestVerify_requireDigestOnVerify_rejectsTag`); `policy_eval_github_test.go` |
| FR-API-012 | Webhook delivery with HMAC | `internal/api/webhooks_test.go` |
| FR-DX-011 | `verity login` caches token | `internal/cliauth` tests |
| C5/C7 | Digest stderr warning + consumer guide | [consumer-getting-started.md](../guides/consumer-getting-started.md) |

## Required CI (branch protection)

On `main`, require (unchanged from v0.2):

| Check | Workflow / job |
|-------|----------------|
| `lint` | `ci.yml` |
| `test` | `ci.yml` |
| `build` | `ci.yml` |
| `keyless-publish` | `publish-keyless-smoke.yml` |

## Manual verification (release manager)

```bash
make test
verity login   # or export VERITY_TOKEN
verity pull gh/acme/widget/app@sha256:... -o ./out
verity verify sha256:... --namespace gh/acme/widget --artifact app
```

## Known limitations (v0.3)

| Limitation | Notes |
|------------|-------|
| CLI device OAuth | Env + GitHub Actions OIDC primary; interactive device flow deferred |
| Webhooks | At-least-once; no delivery UI |
| CVE / federation | Phase 3 Deferred |

## Related

- [layer-c-v0.3-plan.md](layer-c-v0.3-plan.md)
- [phase3-layer-c-test-mapping.md](phase3-layer-c-test-mapping.md) — automated traceability
- [mvp-v0.2-release.md](mvp-v0.2-release.md)
- [v0.3.0 release notes](../releases/v0.3.0.md)
