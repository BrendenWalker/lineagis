# Provenance and Metadata

## Summary

Provenance and metadata cover SLSA-style build attestations, linkage to source repository and git commit, CI workflow identity, and SBOM attachment—all bound to artifact digests and exposed during inspect.

See [00-overview.md](00-overview.md#mvp-delivery-matrix). Most requirements are **Should** for MVP; Must items focus on metadata persistence hooks.

## Goals

- Answer: who built this, from which repo/commit, and which workflow published it.
- Attach and verify SBOMs alongside artifacts.
- Support inspect output aligned with the README example.

## Non-goals

- Full SLSA Level 4 reproducible build verification (Phase 3).
- Generating SBOMs from source (Verity accepts or stores SBOMs; generation may be CI responsibility).
- Vulnerability scanning (see policy spec; CVE blocking is Deferred).
- Custom attestation types beyond provenance and SBOM for MVP.

## Personas

| Persona | Need |
|---------|------|
| **Maintainer** | Automatic provenance on publish from CI context. |
| **Consumer** | Readable provenance and SBOM presence on inspect. |
| **Operator** | Query artifacts by repository or commit (Should). |

## User stories

| ID | Priority | Story |
|----|----------|-------|
| US-PROV-001 | Should | As a maintainer, I want provenance generated on publish so that consumers see build origin. |
| US-PROV-002 | Should | As a consumer, I want inspect to show repository and commit so that I can audit source. |
| US-PROV-003 | Should | As a consumer, I want inspect to show CI workflow identity so that I know which pipeline published. |
| US-PROV-004 | Should | As a maintainer, I want to attach an SBOM so that consumers know components included. |
| US-PROV-005 | Should | As a consumer, I want provenance cryptographically verified like signatures. |
| US-PROV-006 | Must | As the Verity API, I want to persist provenance index fields for trust aggregation. |

## Functional requirements

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-PROV-001 | Should | The system SHALL generate or accept SLSA Provenance v1 attestations in in-toto Statement format. |
| FR-PROV-002 | Should | Provenance SHALL include source repository URI and git commit identifier when built from git. |
| FR-PROV-003 | Should | Provenance SHALL include CI workflow identity (name, ref, run id) for GitHub Actions publishes. |
| FR-PROV-004 | Should | Attestations SHALL be bound to the artifact manifest digest. |
| FR-PROV-005 | Should | `verity publish` SHALL upload provenance and register it via `AttachAttestation`. |
| FR-PROV-006 | Should | `verity inspect` SHALL report provenance verification result (`✓ Provenance verified` or failure reason). |
| FR-PROV-007 | Should | The system SHALL support SBOM attachment as an attestation or OCI referrer with SPDX **or** CycloneDX (one format required for MVP). |
| FR-PROV-008 | Should | `verity inspect` SHALL report `✓ SBOM attached` when an SBOM is present and valid. |
| FR-PROV-009 | Should | Provenance signatures SHALL be verified using the same identity model as artifact signatures. |
| FR-PROV-010 | Must | The metadata DB SHALL store parsed provenance fields per [metadata-model.md](metadata-model.md). |
| FR-PROV-011 | Should | `GetTrustStatus` SHALL include provenance and SBOM presence in the trust report. |
| FR-PROV-012 | Should | Inspect SHALL report `✓ Repository verified` when repository claim matches configured ownership rules. |
| FR-PROV-013 | Must (when rule configured) | Inspect SHALL report publisher allowlist result when **trusted-publishers** is configured (`✓ Publisher allowed` or failure); SHALL NOT show `✓` when the rule is absent. |

## Non-functional requirements

| ID | Requirement |
|----|-------------|
| NFR-PROV-001 | Attestation payloads larger than 1 MiB SHOULD be stored as registry blobs with DB index only. |
| NFR-PROV-002 | Provenance verification errors SHALL name missing or invalid fields. |

## Standards and references

- [SLSA Provenance v1](https://slsa.dev/spec/v1.0/provenance)
- [in-toto Statement v1](https://github.com/in-toto/attestation/tree/main/spec/v1)
- [SPDX](https://spdx.dev/specifications/)
- [CycloneDX](https://cyclonedx.org/specification/overview/)
- [02-signing-verification.md](02-signing-verification.md)

## Dependencies

- [01-artifact-publishing.md](01-artifact-publishing.md)
- [02-signing-verification.md](02-signing-verification.md)
- [04-policy-enforcement.md](04-policy-enforcement.md)
- [api.md](api.md)
- [metadata-model.md](metadata-model.md)

## Acceptance criteria

| ID | Criterion | Maps to |
|----|-----------|---------|
| AC-PROV-001 | Given publish from GitHub Actions on `main` at commit C, when inspect runs, then provenance shows repository, commit C, and workflow name. | FR-PROV-002, FR-PROV-003, FR-PROV-006 |
| AC-PROV-002 | Given SBOM attached at publish, when inspect runs, then output includes `✓ SBOM attached`. | FR-PROV-007, FR-PROV-008 |
| AC-PROV-003 | Given modified provenance bytes after sign, when verify runs, then provenance is reported invalid. | FR-PROV-009 |
| AC-PROV-004 | Given provenance for digest D stored via API, when querying metadata by commit, then artifact D is returned (Should). | FR-PROV-010 |

## Resolved open questions (v0.2)

| ID | Decision | Notes |
|----|----------|-------|
| OQ-PROV-001 | **SLSA Build L1** for MVP | Provenance present, signed, and verified on inspect; higher levels deferred. |
| OQ-PROV-002 | **SBOM optional** on publish | Inspect reports `⚠ SBOM not attached` when absent; operators may require via future policy. |
| OQ-PROV-003 | **SPDX and CycloneDX accepted** | `SBOMPredicateType` detects format; guides show one SPDX example. |
| OQ-PROV-004 | **Provenance claim + namespace match** (no GitHub API) | Default in v0.2: provenance URI vs namespace. v0.3: optional `verify_with_github_api` on `repository-ownership` (`FR-POL-013`). |

## Open questions

None for v0.3 Governance scope beyond deferred non-GitHub hosts (`FR-SIGN-011`).
