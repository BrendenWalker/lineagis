# Policy Enforcement

## Summary

Policy enforcement provides declarative, versioned rules evaluated at publish (push) and verify (inspect) time. MVP Must support require-signatures; additional policies are Should or Deferred per the delivery matrix.

See [00-overview.md](00-overview.md#mvp-delivery-matrix).

## Goals

- Block unsigned publishes when configured.
- Allow operators to restrict trusted publishers and verify repository ownership (Should).
- Return actionable pass/fail results to CLI and API.

## Non-goals

- Enterprise RBAC and hierarchical org policies.
- Complex rule DSL (Rego/CUE full engines Deferred; MVP uses structured JSON rules).
- Real-time CVE feed integration for MVP (critical CVE blocking is Deferred).
- Policy simulation UI.

## Personas

| Persona | Need |
|---------|------|
| **Operator** | Define namespace policies. |
| **Maintainer** | Understand why publish failed. |
| **Consumer** | See policy results on inspect. |

## Initial policies (from README)

| Policy | Priority | Description |
|--------|----------|-------------|
| **require-signatures** | Must | Reject publish/tag when artifact digest has no valid signature. |
| **trusted-publishers** | Should | Allow only identities in namespace publisher allowlist. |
| **repository-ownership** | Should | Require provenance repository claim to match registered repo for namespace. |
| **block-critical-cves** | Deferred | Reject or warn when SBOM/vuln scan reports critical CVEs. |

## User stories

| ID | Priority | Story |
|----|----------|-------|
| US-POL-001 | Must | As an operator, I want to require signatures so that unsigned artifacts cannot be tagged. |
| US-POL-002 | Should | As an operator, I want to allow only trusted GitHub workflows to publish. |
| US-POL-003 | Should | As an operator, I want repository ownership verified so that provenance cannot claim arbitrary repos. |
| US-POL-004 | Must | As a maintainer, I want clear policy failure messages so that I can fix CI configuration. |
| US-POL-005 | Should | As a consumer, I want inspect to show policy results so that I see governance status. |

## Functional requirements

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-POL-001 | Must | Policies SHALL be versioned and scoped to a namespace. |
| FR-POL-002 | Must | Policy documents SHALL be declarative JSON (syntax detailed in implementation; schema TBD). |
| FR-POL-003 | Must | Push-time evaluation SHALL run on `SetTag` before the tag is advertised as trusted. |
| FR-POL-004 | Must | Verify-time evaluation SHALL run on `Verify` and `GetTrustStatus`. |
| FR-POL-005 | Must | Policy **require-signatures** SHALL fail when no valid signature exists for the digest. |
| FR-POL-006 | Should | Policy **trusted-publishers** SHALL fail when signer identity is not in the allowlist. |
| FR-POL-007 | Should | Policy **repository-ownership** SHALL fail when provenance repository does not match namespace-linked repository. |
| FR-POL-008 | Deferred | Policy **block-critical-cves** SHALL fail when critical CVEs are detected in attached SBOM. |
| FR-POL-009 | Must | Policy failures SHALL return `POLICY_FAILED` with rule id and remediation hint. |
| FR-POL-010 | Must | Policy changes SHALL be audit-logged with operator identity and timestamp. |
| FR-POL-011 | Should | Multiple policies SHALL compose with all Must/Should rules evaluated; overall fail if any required rule fails. |

## Evaluation model (MVP default)

| Phase | Trigger | Rules run |
|-------|---------|-----------|
| **Push-time** | `SetTag`, end of `publish` | require-signatures (Must); trusted-publishers, repository-ownership (Should) |
| **Verify-time** | `inspect`, `Verify` | All active policies |

See OQ-OV-001 if operators need verify-only or push-only modes.

## Non-functional requirements

| ID | Requirement |
|----|-------------|
| NFR-POL-001 | Policy evaluation SHALL be deterministic for the same digest and policy version. |
| NFR-POL-002 | Policy documents SHALL be auditable (human-readable JSON). |

## Standards and references

- [README policy list](../../README.md#policy-enforcement)
- [api.md](api.md) — `PutPolicy`, `EvaluatePolicy`
- [02-signing-verification.md](02-signing-verification.md)
- [03-provenance-metadata.md](03-provenance-metadata.md)

## Dependencies

- [api.md](api.md)
- [metadata-model.md](metadata-model.md)
- [02-signing-verification.md](02-signing-verification.md)
- [03-provenance-metadata.md](03-provenance-metadata.md)

## Acceptance criteria

| ID | Criterion | Maps to |
|----|-----------|---------|
| AC-POL-001 | Given require-signatures enabled, when tagging unsigned digest, then `POLICY_FAILED` and tag not updated. | FR-POL-003, FR-POL-005 |
| AC-POL-002 | Given signed digest from allowlisted workflow, when trusted-publishers enabled, then evaluation passes. | FR-POL-006 |
| AC-POL-003 | Given provenance claiming repo R1 but namespace linked to R2, when repository-ownership enabled, then evaluation fails with repository mismatch message. | FR-POL-007 |
| AC-POL-004 | Given inspect on passing artifact, when policies pass, then trust report shows policy section all pass. | FR-POL-004, FR-POL-011 |
| AC-POL-005 | Given policy update by operator, when audit log queried, then change event includes policy version and operator id. | FR-POL-010 |

## Open questions

| ID | Question | Cross-ref |
|----|----------|-----------|
| OQ-POL-001 | Push-time vs verify-time only defaults? | OQ-OV-001 |
| OQ-POL-002 | Warn vs fail for Should policies not met on MVP release? |
| OQ-POL-003 | CVE data source and severity threshold when block-critical-cves is implemented? | Deferred |
