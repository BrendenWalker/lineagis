# Security

## Reporting vulnerabilities

If you discover a security issue, please report it responsibly. Open a private security advisory on GitHub or contact the maintainers directly. Do not file public issues for undisclosed vulnerabilities.

## Threat model (summary)

Verity separates **OCI blob storage** from **trust metadata** (PostgreSQL). The Verity API and metadata database are part of the trusted computing base (TCB):

- Namespace and artifact records
- Tag → digest mappings
- Signature bundles and attestations
- Active policy documents and evaluation results

An attacker who can modify policy, tags, or trust state without detection may weaken enforcement even if registry blobs are intact.

Verity provides **integrity, identity, and policy attribution** — not malware prevention. Compromised CI can produce valid signatures and provenance for malicious artifacts.

## Production deployment guidance

| Practice | Rationale |
|----------|-----------|
| **Disable `VERITY_DEV_TOKEN`** | Dev bearer bypasses OIDC; never expose in production |
| **Require TLS** for API and registry endpoints | Protect tokens and metadata in transit |
| **Configure GitHub OIDC** | `VERITY_OIDC_ISSUER`, `VERITY_OIDC_AUDIENCE` for maintainer publish paths |
| **Restrict operator APIs** | Policy and publisher configuration require operator role |
| **Pin consumer references** | Use `sha256:…` digests; mutable semver tags alone are vulnerable to substitution |
| **Run `verity inspect` in CI** | Fail the job on Must check failures (`--output json`) |
| **Protect database backups** | Metadata tampering affects trust decisions |

## Signing and verification

- Publish from **GitHub Actions** with `permissions.id-token: write` for keyless Sigstore signing.
- `verity inspect` performs **server-side** Sigstore verification via the API. Consumers must trust the API endpoint or add separate verification (post-v0.1: local cosign verify).
- Keyless certificate identity pinning for trust-status crypto checks uses permissive matchers in some dev paths; production identity enforcement relies on policy (`require-signatures`, `trusted-publishers` when enabled).

## Policy

- **`require-signatures` (Must):** Blocks semver tagging and fails inspect when no valid signature exists for the digest.
- **Should policies** (`trusted-publishers`, `repository-ownership`): Evaluated at verify time in v0.1; configure explicitly and treat inspect `⚠` lines accordingly.

Policy changes should be auditable (`FR-POL-010`). Review audit logs after policy or namespace configuration updates.

## What Verity does not protect against

- Compromised build pipelines or repository write access
- Malicious but correctly signed artifacts
- Incomplete or dishonest SBOMs
- Dependency vulnerabilities (CVE blocking is Deferred)
- Consumers who skip `verity inspect` and pull only from a registry

## Related documentation

- [MVP v0.1 release checklist](docs/sdlc/mvp-v0.1-release.md)
- [Specs overview](docs/specs/00-overview.md)
- [Policy enforcement spec](docs/specs/04-policy-enforcement.md)
