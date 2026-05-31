# Artifact Publishing

## Summary

Artifact publishing covers OCI-compatible push and pull of software artifacts, immutable content digests, and semantic version tagging. Lineagis treats the OCI registry as the distribution layer while the Lineagis API registers artifacts and tags in the metadata database.

See [00-overview.md](00-overview.md#mvp-delivery-matrix).

## Goals

- Enable maintainers to publish packages, binaries, containers, and related files via OCI.
- Guarantee content-addressed immutability for every published version.
- Support semver tags for human-friendly consumption while preserving digest pull.

## Non-goals

- PyPI, npm, or Maven proxy semantics.
- Advanced search, indexing, or package dependency resolution.
- Multi-artifact bundle formats beyond OCI manifest capabilities (Deferred).
- Automatic garbage collection policies (operator concern).

## Personas

| Persona | Need |
|---------|------|
| **Maintainer** | Push release artifacts from CI or workstation. |
| **Consumer** | Pull by digest or semver tag. |
| **Operator** | Configure registry endpoint and namespace limits. |

## User stories

| ID | Priority | Story |
|----|----------|-------|
| US-PUB-001 | Must | As a maintainer, I want to push build outputs to Lineagis so that they are stored immutably. |
| US-PUB-002 | Must | As a maintainer, I want to tag a release with semver so that consumers can reference `v1.2.0`. |
| US-PUB-003 | Must | As a consumer, I want to pull by digest so that I get exactly the published bytes. |
| US-PUB-004 | Must | As a consumer, I want to pull by tag so that I receive the digest the tag currently points to. |
| US-PUB-005 | Should | As a maintainer, I want re-uploading identical content to be idempotent so that CI retries are safe. |

## Functional requirements

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-PUB-001 | Must | The system SHALL support OCI Distribution-compatible `push` and `pull` for artifact blobs and manifests. |
| FR-PUB-002 | Must | Every published manifest SHALL receive a unique digest computed per OCI rules. |
| FR-PUB-003 | Must | Blob content SHALL be immutable; the same digest SHALL always yield the same bytes. |
| FR-PUB-004 | Must | The CLI `publish` command SHALL upload local files or directories to the configured registry and register the result with the Lineagis API. |
| FR-PUB-005 | Must | The system SHALL support semantic version tags (e.g. `v1.2.0`, `1.2.0`) on artifacts. |
| FR-PUB-006 | Must | Tag resolution SHALL return the digest currently mapped to that tag. |
| FR-PUB-007 | Must | Re-pushing identical content SHALL NOT create a new digest. |
| FR-PUB-008 | Must | Moving a tag to a new digest SHALL be explicit via publish/tag API, not silent overwrite of blob content. |
| FR-PUB-009 | Should | Multi-file releases (e.g. `dist/*`) SHALL be published as a single OCI manifest referencing multiple layers or equivalent layout. Layout: [ADR-0001](../adr/0001-artifact-manifest-layout.md). |
| FR-PUB-010 | Should | Supported artifact types for MVP SHALL include at least: generic files (blobs), container images, and Python wheels. |

## Non-functional requirements

| ID | Requirement |
|----|-------------|
| NFR-PUB-001 | Publish operations SHALL be resumable or idempotent where OCI supports it. |
| NFR-PUB-002 | Digest algorithm for MVP SHALL be SHA-256. |

## Standards and references

- [OCI Distribution Spec](https://github.com/opencontainers/distribution-spec)
- [OCI Image Spec](https://github.com/opencontainers/image-spec)
- [ADR-0001: Artifact manifest layout](../adr/0001-artifact-manifest-layout.md) — OCI Artifact manifest, layer layout ([FR-PUB-009](#functional-requirements))
- [metadata-model.md](metadata-model.md) — tag and digest semantics
- [api.md](api.md) — `RegisterDigest`, `SetTag`

## Dependencies

- [architecture.md](architecture.md)
- [api.md](api.md)
- [metadata-model.md](metadata-model.md)
- [02-signing-verification.md](02-signing-verification.md) — signing after push
- [05-developer-experience.md](05-developer-experience.md) — `lineagis publish`

## Acceptance criteria

| ID | Criterion | Maps to |
|----|-----------|---------|
| AC-PUB-001 | Given a local file, when `lineagis publish` runs successfully, then the output includes a `sha256:` digest. | FR-PUB-002, FR-PUB-004 |
| AC-PUB-002 | Given the same file published twice, when digests are compared, then they are identical. | FR-PUB-007 |
| AC-PUB-003 | Given tag `v1.0.0` set on digest D, when pulling `artifact:v1.0.0`, then content matches pull by D. | FR-PUB-005, FR-PUB-006 |
| AC-PUB-004 | Given tag moved from D1 to D2, when pulling by D1 digest, then original content is unchanged. | FR-PUB-008 |
| AC-PUB-005 | Given OCI-compatible registry client, when push/pull without Lineagis CLI, then blob transfer succeeds (registry compatibility). | FR-PUB-001 |

## Open questions

| ID | Question |
|----|----------|
| OQ-PUB-002 | Maximum artifact size limits for MVP? |

## Resolved open questions

| ID | Question | Resolution |
|----|----------|------------|
| OQ-PUB-001 | Standard OCI artifact type vs custom manifest media type for generic packages? | [ADR-0001](../adr/0001-artifact-manifest-layout.md) — OCI Artifact manifest (`application/vnd.oci.artifact.manifest.v1+json`); one layer per file; layers sorted by `dev.lineagis.path`. |
