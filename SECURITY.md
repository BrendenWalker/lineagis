# Security

## Reporting vulnerabilities

If you discover a security issue, please report it responsibly. Open a private security advisory on GitHub or contact the maintainers directly. Do not file public issues for undisclosed vulnerabilities.

## Threat model (summary)

Lineagis separates **OCI blob storage** from **trust metadata** (PostgreSQL). The Lineagis API and metadata database are part of the trusted computing base (TCB):

- Namespace and artifact records
- Tag → digest mappings
- Signature bundles and attestations
- Active policy documents and evaluation results

An attacker who can modify policy, tags, or trust state without detection may weaken enforcement even if registry blobs are intact.

Lineagis provides **integrity, identity, and policy attribution** — not malware prevention. Compromised CI can produce valid signatures and provenance for malicious artifacts.

## Production deployment guidance

| Practice | Rationale |
|----------|-----------|
| **Disable `LINEAGIS_DEV_TOKEN`** | Dev bearer bypasses OIDC; never expose in production |
| **Require TLS** for API and registry endpoints | Protect tokens and metadata in transit |
| **Configure GitHub OIDC** | `LINEAGIS_OIDC_ISSUER`, `LINEAGIS_OIDC_AUDIENCE` for maintainer publish paths |
| **Restrict operator APIs** | Policy and publisher configuration require operator role |
| **Pin consumer references** | Use `sha256:…` digests; mutable semver tags alone are vulnerable to substitution |
| **Run `lineagis inspect` in CI** | Fail the job on Must check failures (`--output json`) |
| **Protect database backups** | Metadata tampering affects trust decisions |
| **Encrypt PostgreSQL at rest** | Stolen disks expose tags, policies, and signature metadata |
| **Restrict policy writes** | Only operator role may `PutPolicy` or configure webhooks |
| **Review audit logs** | After policy, tag, or webhook changes (`GET …/audit`) |
| **Webhook secrets** | Use HMAC secrets on HTTPS endpoints; rotate on compromise |
| **Optional GitHub API token** | `LINEAGIS_GITHUB_TOKEN` for `verify_with_github_api`; scope to `repo` read |

## Control-plane hardening (v0.3)

### Compromised operator account

An operator who can change namespace policy, webhooks, or trusted publishers can weaken enforcement for future publishes. Mitigations:

- Separate operator credentials from maintainer CI tokens
- Require human review for policy changes (PR on policy JSON in git)
- Monitor audit events and webhook `policy.updated` deliveries
- Restore policy from versioned backups if tampering is suspected

### Database and backups

Treat PostgreSQL backups like signing keys: encrypt at rest, restrict access, and test restore procedures. Restoring an old backup can roll back policy or tag state.

### Network

- Terminate TLS at the API and registry
- Optional mTLS between Lineagis API and private registry (deployment-specific)
- Rate-limit policy mutation endpoints at the ingress when exposed publicly

## Signing and verification

- Publish from **GitHub Actions** with `permissions.id-token: write` for keyless Sigstore signing.
- `lineagis inspect` / `lineagis verify` default to **local** Sigstore verification against registry manifest bytes; use `--trust-api` to skip local crypto and rely on API trust status only.
- Keyless certificate identity matchers are derived from namespace `trusted-publishers` policy when configured; set `LINEAGIS_PERMISSIVE_KEYLESS_IDENTITY=1` only for local dev.

## Policy

- **`require-signatures`:** Blocks semver tagging and fails inspect when no valid signature exists for the digest.
- **`trusted-publishers`:** When the rule is in your namespace policy, only operator-configured signing identities pass at **tag time and inspect** (fail-closed). Pin `repository`, `workflow`, optional `ref` and `issuer` — avoid broad org wildcards.
- **`require-provenance`:** When configured, fails if provenance is missing or signature verification failed.
- **`repository-ownership`:** When configured, fails if provenance repository does not match the namespace. Optional `verify_with_github_api: true` requires live GitHub REST verification (fail-closed if API unavailable).
- **`require-digest-on-verify`:** When configured, rejects verify/inspect by semver tag; use `sha256:…` in CI.
- **Push-time enforcement:** `require-signatures` applies on `RegisterDigest` (bundle required) and `SetTag`; other rules run on `SetTag` and inspect (FR-POL-012). Use `lineagis verify` with a pinned digest in CI.

Policy changes should be auditable (`FR-POL-010`). Review audit logs after policy or namespace configuration updates.

## What Lineagis does not protect against

- Compromised build pipelines or repository write access
- Malicious but correctly signed artifacts
- Incomplete or dishonest SBOMs
- Dependency vulnerabilities (CVE blocking is Deferred)
- Consumers who skip `lineagis inspect` and pull only from a registry

## Related documentation

- [MVP v0.3 release checklist](docs/sdlc/mvp-v0.3-release.md)
- [Consumer getting started](docs/guides/consumer-getting-started.md)
- [MVP v0.1 release checklist](docs/sdlc/mvp-v0.1-release.md)
- [Specs overview](docs/specs/00-overview.md)
- [Policy enforcement spec](docs/specs/04-policy-enforcement.md)
