# Verity MVP Specifications

Requirements-focused specifications for the Verity MVP. These documents define *what* the system must do—not implementation details such as OpenAPI schemas or database migrations.

## Reading order

1. [00-overview.md](00-overview.md) — MVP boundaries, delivery matrix, glossary, end-to-end flows
2. [architecture.md](architecture.md) — Components, trust boundaries, deployment assumptions
3. [metadata-model.md](metadata-model.md) — Entities and storage placement
4. [api.md](api.md) — Verity API resources, operations, and authentication
5. Feature specs (in dependency order):
   - [01-artifact-publishing.md](01-artifact-publishing.md)
   - [02-signing-verification.md](02-signing-verification.md)
   - [03-provenance-metadata.md](03-provenance-metadata.md)
   - [04-policy-enforcement.md](04-policy-enforcement.md)
   - [05-developer-experience.md](05-developer-experience.md)

## Spec template

Each spec document follows this structure (see [_template.md](_template.md)):

1. Summary
2. Goals
3. Non-goals
4. Personas
5. User stories
6. Functional requirements (`FR-<AREA>-<NNN>`)
7. Non-functional requirements (`NFR-<AREA>-<NNN>`)
8. Standards and references
9. Dependencies
10. Acceptance criteria
11. Open questions

## Delivery priority

Requirements are tagged in the [delivery matrix](00-overview.md#mvp-delivery-matrix):

| Tag | Meaning |
|-----|---------|
| **Must** | Required for MVP release |
| **Should** | Target for MVP; acceptable to defer with documented workaround |
| **Deferred** | Post-MVP (Phase 2+ or Phase 3) |

## Glossary

| Term | Definition |
|------|------------|
| **Artifact** | A publishable software deliverable (package, binary, container image, etc.) stored and distributed via OCI. |
| **Attestation** | A signed statement about an artifact, bound to its digest (e.g. SLSA provenance, SBOM). |
| **Digest** | Content-addressed identifier (e.g. `sha256:…`) for an immutable blob or manifest. |
| **Keyless signing** | Signing using ephemeral certificates from an OIDC identity (e.g. GitHub Actions) via Sigstore. |
| **Maintainer** | A project contributor authorized to publish artifacts for a repository or namespace. |
| **Policy** | A declarative rule evaluated at publish or verify time (e.g. require signature). |
| **Policy decision** | The result of evaluating one or more policies (`pass`, `fail`, `warn`). |
| **Provenance** | Metadata describing how and where an artifact was built (source repo, commit, workflow). |
| **Publisher** | The identity (OIDC subject, org, workflow) that performed a publish or sign operation. |
| **Registry** | OCI Distribution-compatible storage for manifests and blobs. |
| **SBOM** | Software Bill of Materials describing components in an artifact. |
| **Tag** | A mutable human-readable label (e.g. semver `v1.2.0`) pointing to a digest. |
| **Trust status** | Aggregated verification outcome for an artifact (signatures, provenance, policies). |
| **Verity API** | HTTP API mediating publish, metadata, policy, and trust operations. |

## External standards

| Standard | Use in Verity |
|----------|----------------|
| [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec) | Artifact push/pull, manifests, blobs |
| [OCI Image Spec](https://github.com/opencontainers/image-spec) | Manifest and descriptor formats |
| [Sigstore](https://docs.sigstore.dev/) | Keyless signing and signature verification |
| [SLSA Provenance](https://slsa.dev/spec/v1.0/provenance) | Build provenance attestations |
| [in-toto Statement](https://github.com/in-toto/attestation/tree/main/spec/v1) | Attestation envelope format |
| [GitHub OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect) | CI publisher identity |
| [SPDX](https://spdx.dev/) / [CycloneDX](https://cyclonedx.org/) | SBOM document formats (MVP: one format required) |

## Requirement ID conventions

- `FR-<AREA>-<NNN>` — Functional requirement (e.g. `FR-PUB-001`)
- `NFR-<AREA>-<NNN>` — Non-functional requirement
- `<AREA>` codes: `OV` overview, `ARCH` architecture, `API`, `META`, `PUB`, `SIGN`, `PROV`, `POL`, `DX`

Cross-cutting open questions are tracked in [00-overview.md#cross-spec-open-questions](00-overview.md#cross-spec-open-questions).

## Requirement traceability

Overview requirements map to feature specs as follows:

| Overview FR | Feature specs |
|-------------|---------------|
| FR-OV-001 – FR-OV-003 | [01-artifact-publishing.md](01-artifact-publishing.md) |
| FR-OV-004 | [02-signing-verification.md](02-signing-verification.md) |
| FR-OV-005 | [api.md](api.md), [architecture.md](architecture.md) |
| FR-OV-006 | [05-developer-experience.md](05-developer-experience.md) |
| FR-OV-007 | [03-provenance-metadata.md](03-provenance-metadata.md) |
| FR-OV-008 | [05-developer-experience.md](05-developer-experience.md) |
| FR-OV-009 | [04-policy-enforcement.md](04-policy-enforcement.md) |
| FR-OV-010 | [04-policy-enforcement.md](04-policy-enforcement.md) (Deferred) |

Inspect output line ownership:

| README inspect line | Primary spec |
|---------------------|--------------|
| Signed by GitHub Actions | 02-signing-verification |
| Repository verified | 03-provenance-metadata, 04-policy-enforcement |
| Maintainer verified | 04-policy-enforcement |
| SBOM attached | 03-provenance-metadata |
| Provenance verified | 03-provenance-metadata |
| No critical vulnerabilities | 04-policy-enforcement (Deferred) |

## Document index

| Document | FR prefix |
|----------|-----------|
| [00-overview.md](00-overview.md) | FR-OV |
| [architecture.md](architecture.md) | FR-ARCH |
| [api.md](api.md) | FR-API |
| [metadata-model.md](metadata-model.md) | FR-META |
| [01-artifact-publishing.md](01-artifact-publishing.md) | FR-PUB |
| [02-signing-verification.md](02-signing-verification.md) | FR-SIGN |
| [03-provenance-metadata.md](03-provenance-metadata.md) | FR-PROV |
| [04-policy-enforcement.md](04-policy-enforcement.md) | FR-POL |
| [05-developer-experience.md](05-developer-experience.md) | FR-DX |
