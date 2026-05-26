# Signing and Verification

## Summary

Signing and verification integrate Sigstore for keyless signing (especially from GitHub Actions) and cryptographic verification of artifact digests. Signatures are bound to manifest digests and surfaced in trust status and `verity inspect` output.

See [00-overview.md](00-overview.md#mvp-delivery-matrix).

## Goals

- Enable keyless signing without maintainer-managed long-lived keys for CI.
- Verify signatures on inspect and via API trust status.
- Fail closed when require-signature policy applies.

## Non-goals

- Custom PKI or enterprise HSM integration (Deferred).
- Hardware token signing flows for MVP.
- Replacing Sigstore transparency log (Rekor) with a custom log (Phase 3).
- Manual `gpg` armor signature support.

## Personas

| Persona | Need |
|---------|------|
| **Maintainer** | Sign automatically during publish from CI. |
| **Consumer** | Confirm artifact was signed by expected identity. |
| **Operator** | Configure Sigstore trust roots (Fulcio, Rekor). |

## User stories

| ID | Priority | Story |
|----|----------|-------|
| US-SIGN-001 | Must | As a maintainer, I want artifacts signed during publish so that consumers can detect tampering. |
| US-SIGN-002 | Must | As a CI workflow, I want keyless signing via GitHub OIDC so that I do not store signing keys in secrets. |
| US-SIGN-003 | Must | As a consumer, I want signature verification on inspect so that I trust the artifact integrity. |
| US-SIGN-004 | Should | As a consumer, I want to see which identity signed the artifact (e.g. GitHub Actions workflow). |
| US-SIGN-005 | Should | As the Verity API, I want to verify signatures when serving trust status so clients cannot skip verification. |

## Functional requirements

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-SIGN-001 | Must | The system SHALL integrate with Sigstore for signing and verification. |
| FR-SIGN-002 | Must | The system SHALL support keyless signing using OIDC identities. |
| FR-SIGN-003 | Must | Signatures SHALL cover the artifact manifest digest. |
| FR-SIGN-004 | Must | `verity publish` SHALL sign artifacts after upload unless signing is explicitly skipped and policy allows. |
| FR-SIGN-005 | Must | `verity inspect` SHALL verify signatures and report valid, invalid, or missing. |
| FR-SIGN-006 | Must | The Verity API SHALL verify signatures when computing trust status. |
| FR-SIGN-007 | Must | Unsigned artifacts SHALL fail push-time policy when require-signature is enabled. |
| FR-SIGN-008 | Should | Verified signatures SHALL expose signer identity claims (issuer, subject, repository, workflow). |
| FR-SIGN-009 | Should | Signature bundles SHALL be attachable via `AttachSignature` and stored per [metadata-model.md](metadata-model.md). |
| FR-SIGN-010 | Should | Provenance verification SHALL integrate with [03-provenance-metadata.md](03-provenance-metadata.md) (separate attestation checks). |
| FR-SIGN-011 | Deferred | Support for non-GitHub OIDC issuers beyond local dev stubs. |

## Non-functional requirements

| ID | Requirement |
|----|-------------|
| NFR-SIGN-001 | Verification SHALL use configurable Sigstore trust material (Fulcio roots, Rekor). |
| NFR-SIGN-002 | Signing failures SHALL abort publish when signing is required by policy or default. |

## Standards and references

- [Sigstore](https://docs.sigstore.dev/)
- [Cosign documentation](https://docs.sigstore.dev/cosign/overview/)
- [Fulcio](https://github.com/sigstore/fulcio)
- [Rekor](https://github.com/sigstore/rekor)
- [GitHub OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)

## Dependencies

- [01-artifact-publishing.md](01-artifact-publishing.md)
- [03-provenance-metadata.md](03-provenance-metadata.md)
- [04-policy-enforcement.md](04-policy-enforcement.md)
- [api.md](api.md)

## Acceptance criteria

| ID | Criterion | Maps to |
|----|-----------|---------|
| AC-SIGN-001 | Given publish from GitHub Actions with OIDC, when inspect runs, then output includes `✓ Signed by GitHub Actions`. | FR-SIGN-002, FR-SIGN-005, FR-SIGN-008 |
| AC-SIGN-002 | Given tampered artifact bytes with valid old signature, when verify runs, then signature is reported invalid. | FR-SIGN-005, FR-SIGN-006 |
| AC-SIGN-003 | Given unsigned artifact and require-signature policy, when `SetTag` is called, then publish fails with `POLICY_FAILED`. | FR-SIGN-007 |
| AC-SIGN-004 | Given valid keyless signature for digest D, when `GetTrustStatus` is called for D, then signatures are `valid`. | FR-SIGN-006 |

## Open questions

| ID | Question |
|----|----------|
| OQ-SIGN-001 | **Resolved (MVP):** Default to Sigstore public-good endpoints and TUF trust; operators override via `VERITY_SIGSTORE_*` (or `SIGSTORE_*`) — see [signing-local.md](../signing-local.md). |
| OQ-SIGN-002 | Allow cosign attach-signatures vs integrated sign-only in CLI? |
