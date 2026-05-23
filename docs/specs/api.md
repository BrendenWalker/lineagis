# Verity API

## Summary

The Verity API is an HTTPS JSON API that registers artifacts, indexes trust metadata, stores policies, evaluates rules, and returns aggregated trust status. It does not replace the OCI Distribution API for blob transfer; clients push/pull content directly to the registry while using the Verity API for control-plane operations.

See [00-overview.md](00-overview.md), [metadata-model.md](metadata-model.md), and [architecture.md](architecture.md).

## Goals

- Define a stable resource and operation model for MVP feature specs.
- Specify authentication and authorization requirements (OIDC / GitHub Actions).
- Provide a consistent error taxonomy for clients and CI.

## Non-goals

- OpenAPI/Swagger document generation (future work).
- Full OCI Distribution API reimplementation.
- Public unauthenticated publish.
- Fine-grained enterprise RBAC (roles beyond namespace-scoped operator/maintainer/reader).

## Personas

| Persona | API usage |
|---------|-----------|
| **Maintainer** | Register artifacts, attach signatures/attestations, trigger policy check on publish. |
| **Consumer** | Read trust status and artifact metadata (read-only token or anonymous read if enabled by operator). |
| **Operator** | Manage policies, trusted publishers, namespace settings. |

## Resource model

| Resource | Identifier | Description |
|----------|------------|-------------|
| `Namespace` | `name` | Trust and publishing boundary. |
| `Artifact` | `namespace/name` | Logical artifact (e.g. `gh/acme/widget`). |
| `Tag` | `namespace/name:tag` | Semver or label pointing to a digest. |
| `Digest` | `sha256:…` | Immutable manifest reference. |
| `Signature` | `digest` + `id` | Sigstore signature bundle for a digest. |
| `Attestation` | `digest` + `id` | in-toto Statement (provenance, SBOM, etc.). |
| `Policy` | `namespace/id` | Versioned policy document. |
| `PolicyDecision` | `digest` + evaluation id | Result of policy run. |
| `TrustStatus` | `namespace/name@digest` or tag | Aggregated verification report. |
| `Publisher` | OIDC `issuer` + `subject` | Trusted or observed publisher identity. |

## Operations

### Namespace and artifact

| Operation | Method (informative) | Priority | Description |
|-----------|----------------------|----------|-------------|
| `GetNamespace` | GET | Must | Return namespace config and policy summary. |
| `CreateOrUpdateArtifact` | PUT | Must | Register logical artifact under namespace. |
| `GetArtifact` | GET | Must | Return artifact metadata and tags. |
| `ListArtifacts` | GET | Should | List artifacts in namespace (paginated). |

### Tags and digests

| Operation | Method (informative) | Priority | Description |
|-----------|----------------------|----------|-------------|
| `RegisterDigest` | POST | Must | Record manifest digest after OCI push; link to artifact. |
| `SetTag` | PUT | Must | Map semver tag to digest; enforce policy on push. |
| `GetTag` | GET | Must | Resolve tag to digest. |
| `GetDigest` | GET | Must | Return digest metadata and references. |

### Trust metadata

| Operation | Method (informative) | Priority | Description |
|-----------|----------------------|----------|-------------|
| `AttachSignature` | POST | Must | Store signature bundle reference for digest. |
| `AttachAttestation` | POST | Should | Store attestation envelope for digest. |
| `ListAttestations` | GET | Should | List attestations for digest. |
| `GetTrustStatus` | GET | Must | Return aggregated trust report for digest or tag. |
| `Verify` | POST | Must | Run verify-time policy and signature checks; return decision. |

### Policy and publishers

| Operation | Method (informative) | Priority | Description |
|-----------|----------------------|----------|-------------|
| `GetPolicy` | GET | Must | Return active policy for namespace. |
| `PutPolicy` | PUT | Must | Create new policy version (operator). |
| `EvaluatePolicy` | POST | Must | Evaluate policies for digest (push or verify phase). |
| `ListPublishers` | GET | Should | List trusted publishers for namespace. |
| `PutPublisher` | PUT | Should | Add/update trusted publisher (operator). |

## Authentication and authorization

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-API-001 | Must | Protected write operations SHALL require a valid OIDC bearer token. |
| FR-API-002 | Must | The API SHALL validate token issuer, audience, and expiry. |
| FR-API-003 | Must | GitHub Actions tokens SHALL be accepted for publish and attach operations when `repository` and `ref` claims match the target namespace rules. |
| FR-API-004 | Must | Policy and publisher configuration SHALL require operator role for the namespace. |
| FR-API-005 | Should | Read-only trust status MAY be allowed without auth per namespace config (default: authenticated read). |
| FR-API-006 | Must | On auth failure, the API SHALL return an error and perform no side effects. |

**Roles (MVP):**

| Role | Permissions |
|------|-------------|
| `operator` | Policy, publishers, namespace config |
| `maintainer` | Register digest, set tag, attach signature/attestation |
| `reader` | Get artifact, trust status, verify |

## Functional requirements

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-API-007 | Must | `SetTag` SHALL run push-time policy evaluation before accepting the tag. |
| FR-API-008 | Must | `GetTrustStatus` SHALL include signature validity, policy decisions, and attestation presence flags. |
| FR-API-009 | Must | All mutable resources SHALL be versioned or audit-logged on change. |
| FR-API-010 | Should | `AttachAttestation` SHALL validate envelope format (in-toto Statement) before persistence. |
| FR-API-011 | Should | API responses SHALL include stable error codes (see taxonomy). |
| FR-API-012 | Deferred | Webhook notifications on policy failure. |

## Error taxonomy

Errors return JSON: `{ "code", "message", "details" }`.

| Code | Category | Example |
|------|----------|---------|
| `AUTH_REQUIRED` | Authentication | Missing bearer token |
| `AUTH_INVALID` | Authentication | Expired or invalid OIDC token |
| `FORBIDDEN` | Authorization | Maintainer cannot change policy |
| `NOT_FOUND` | Resource | Unknown artifact or digest |
| `CONFLICT` | Resource | Tag move conflict or duplicate registration |
| `POLICY_FAILED` | Policy | Require-signature not met |
| `VALIDATION_FAILED` | Input | Malformed attestation envelope |
| `REGISTRY_ERROR` | Dependency | Registry unreachable |
| `INTERNAL` | Server | Unexpected failure |

## Non-functional requirements

| ID | Requirement |
|----|-------------|
| NFR-API-001 | API SHALL use HTTPS only in production deployments. |
| NFR-API-002 | Idempotent retries on `RegisterDigest` with same digest SHALL NOT create duplicate rows. |
| NFR-API-003 | Policy evaluation on `SetTag` SHOULD complete within 10s for MVP-sized policy sets. |

## Standards and references

- [GitHub OIDC claims](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect#understanding-the-oidc-token)
- [in-toto Statement v1](https://github.com/in-toto/attestation/tree/main/spec/v1)

## Dependencies

- [metadata-model.md](metadata-model.md)
- [04-policy-enforcement.md](04-policy-enforcement.md)
- Feature specs 01–05

## Acceptance criteria

| ID | Criterion | Maps to |
|----|-----------|---------|
| AC-API-001 | Given no token, when calling `SetTag`, then `AUTH_REQUIRED` is returned. | FR-API-001, FR-API-006 |
| AC-API-002 | Given valid maintainer token and unsigned digest with require-signature policy, when calling `SetTag`, then `POLICY_FAILED` and tag is not updated. | FR-API-007 |
| AC-API-003 | Given signed digest with passing policies, when calling `GetTrustStatus`, then response includes `signatures: valid` and overall `pass`. | FR-API-008 |

## Open questions

| ID | Question |
|----|----------|
| OQ-API-001 | REST path layout: `/v1/namespaces/{ns}/artifacts/{name}` vs alternative? |
| OQ-API-002 | Support API keys for local dev in addition to OIDC? |
