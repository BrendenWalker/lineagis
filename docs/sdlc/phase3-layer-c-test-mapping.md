# Layer C (v0.3 Governance) — test mapping

Traceability for **Layer C** Should items: consumer DX, webhooks, digest-pin policy, and GitHub API ownership depth.

Authoritative scope: [00-overview.md](../specs/00-overview.md), [layer-c-v0.3-plan.md](layer-c-v0.3-plan.md).

## Mapping

- **Consumer CLI: login**
  - `FR-DX-011`, `AC-DX-011` (informal)
  - Coverage: `internal/cliauth/token_test.go`

- **Consumer CLI: pull**
  - `AC-DX-PULL-001`, `AC-DX-PULL-002`, `FR-DX-012`
  - Coverage:
    - `internal/pull/pull_test.go` (`TestPull_byDigest_writesLayers`, `TestPull_resolveTag_thenPull`, `TestPull_withVerify_failsWithoutSignatures`)
    - `internal/pull/ref_test.go`
    - `cmd/verity/pull_test.go` (`TestRunPull_verifyFailsWithoutSignatures`)

- **Digest-pin UX**
  - `FR-POL-014`, `AC-POL-007`, C5
  - Coverage:
    - `internal/api/policy_eval_github_test.go` (`TestEvaluatePolicyDocument_requireDigestOnVerify_*`)
    - `internal/api/policy_eval_v03_test.go` (`TestVerify_requireDigestOnVerify_rejectsTag`)
    - `internal/api/policy_eval_digest_test.go` (verify-phase only)
    - [consumer-getting-started.md](../guides/consumer-getting-started.md)

- **GitHub API repository ownership**
  - `FR-POL-013`, `AC-POL-006`
  - Coverage:
    - `internal/api/policy_eval_v03_test.go` (`TestVerify_repositoryOwnership_githubNotConfigured`, `TestVerify_repositoryOwnership_githubAPIError`)
    - `internal/github/client_test.go`

- **Webhooks**
  - `FR-API-012`
  - Coverage: `internal/api/webhooks_test.go` (`TestDeliverWebhook_hmac`)

- **Operator hardening docs (C6)**
  - Coverage: [SECURITY.md](../../SECURITY.md)

## How to run

```bash
go test ./internal/api/... ./internal/pull/... ./cmd/verity/... ./internal/cliauth/...
```

Database-backed policy tests require `VERITY_TEST_DATABASE_URL`:

```bash
export VERITY_TEST_DATABASE_URL=postgres://verity:verity@localhost:5432/verity?sslmode=disable
go test ./internal/api/... -run 'Verify_repositoryOwnership|Verify_requireDigest'
```

Release sign-off: [mvp-v0.3-release.md](mvp-v0.3-release.md).
