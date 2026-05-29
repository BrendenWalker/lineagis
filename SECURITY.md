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
- `verity inspect` / `verity verify` default to **local** Sigstore verification against registry manifest bytes; use `--trust-api` to skip local crypto and rely on API trust status only.
- Keyless certificate identity matchers are derived from namespace `trusted-publishers` policy when configured; set `VERITY_PERMISSIVE_KEYLESS_IDENTITY=1` only for local dev.

## Policy

- **`require-signatures`:** Blocks semver tagging and fails inspect when no valid signature exists for the digest.
- **`trusted-publishers`:** When the rule is in your namespace policy, only operator-configured signing identities pass at **tag time and inspect** (fail-closed). Pin `repository`, `workflow`, optional `ref` and `issuer` — avoid broad org wildcards.
- **`require-provenance`:** When configured, fails if provenance is missing or signature verification failed.
- **`repository-ownership`:** When configured, fails if provenance repository does not match the namespace.
- **Push-time enforcement:** `require-signatures` applies on `RegisterDigest` (bundle required) and `SetTag`; other rules run on `SetTag` and inspect (FR-POL-012). Use `verity verify` with a pinned digest in CI.

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
