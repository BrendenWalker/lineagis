# Policy Enforcement

## Summary

Policy enforcement provides declarative, versioned rules evaluated at publish (push) and verify (inspect) time. MVP Must support **require-signatures** and **fail-closed evaluation** for every rule an operator adds to a namespace policy.

See [00-overview.md](00-overview.md#mvp-delivery-matrix).

## Goals

- Block unsigned publishes when configured.
- Allow operators to restrict which **signing identities** may produce trusted tags (trusted publishers).
- Return actionable pass/fail results to CLI and API.

## Non-goals

- Enterprise RBAC and hierarchical org policies.
- Complex rule DSL (Rego/CUE full engines Deferred; MVP uses structured JSON rules).
- Real-time CVE feed integration for MVP (critical CVE blocking is Deferred).
- Policy simulation UI.
- A global vendor allowlist — each Verity instance defines its own namespace policies.

## Trusted publishers

**Trusted publishers** is a namespace policy rule, not a global trust list. The **operator** configures which signing identities are allowed for that namespace.

| Term | Meaning |
|------|---------|
| **Signing identity** | Who signed the artifact digest, extracted from the Sigstore bundle (today: GitHub Actions `repository` + optional `workflow` from the Fulcio certificate). |
| **Allowlist** | `publishers` array in the `trusted-publishers` rule `config` — each entry may set `repository` and/or `workflow`. |
| **API OIDC identity** | Who called the Verity API (`issuer` + `subject`). Separate from signing identity; used for authz on metadata writes. |

Example (operator-defined):

```json
{
  "rules": [
    { "type": "require-signatures" },
    {
      "type": "trusted-publishers",
      "config": {
        "publishers": [
          { "repository": "acme/widget", "workflow": "release.yml" }
        ]
      }
    }
  ]
}
```

**Semantics:**

- If the `trusted-publishers` rule is **absent**, no publisher allowlist is enforced.
- If the rule is **present**, evaluation is **fail-closed**: at least one signature on the digest must match an allowlist entry; empty `publishers` fails (misconfiguration).
- Narrow the allowlist when possible: `repository` only trusts any workflow in that repo; add `workflow` to pin a single pipeline.

**Not proven:** trusted publishers does not mean “safe code” — only that the signature came from an identity the operator chose.

## Configured-policy semantics (Option A)

When a policy rule appears in the active namespace policy document:

1. The rule **SHALL** be evaluated whenever policy runs for that phase (push or verify).
2. A failing rule **SHALL** produce `POLICY_FAILED`, block trust, and cause non-zero `verity inspect` exit.
3. There is **no** warn-only mode for configured rules.

Rules not in the document are not evaluated and MUST NOT fail the overall result.

## Personas

| Persona | Need |
|---------|------|
| **Operator** | Define namespace policies and signing-identity allowlists. |
| **Maintainer** | Understand why publish failed. |
| **Consumer** | See policy results on inspect. |

## Initial policies (from README)

| Policy | Priority | Description |
|--------|----------|-------------|
| **require-signatures** | Must | Reject tag/verify when artifact digest has no valid signature. |
| **trusted-publishers** | Must (when rule configured) | Fail when no signature matches the operator allowlist. |
| **repository-ownership** | Should (when rule configured) | Fail when provenance repository does not match namespace-linked repository. |
| **block-critical-cves** | Deferred | Reject when SBOM/vuln scan reports critical CVEs. |

## User stories

| ID | Priority | Story |
|----|----------|-------|
| US-POL-001 | Must | As an operator, I want to require signatures so that unsigned artifacts cannot be tagged. |
| US-POL-002 | Must | As an operator, I want to allow only signing identities I configure (trusted publishers) so that other workflows cannot satisfy policy when that rule is enabled. |
| US-POL-003 | Should | As an operator, I want repository ownership verified so that provenance cannot claim arbitrary repos when that rule is enabled. |
| US-POL-004 | Must | As a maintainer, I want clear policy failure messages so that I can fix CI configuration. |
| US-POL-005 | Must | As a consumer, I want inspect to fail when any configured policy fails so that I can gate CI on real enforcement. |

## Functional requirements

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-POL-001 | Must | Policies SHALL be versioned and scoped to a namespace. |
| FR-POL-002 | Must | Policy documents SHALL be declarative JSON (syntax detailed in implementation; schema TBD). |
| FR-POL-003 | Must | Push-time evaluation SHALL run on `SetTag` before the tag is advertised as trusted. |
| FR-POL-004 | Must | Verify-time evaluation SHALL run on `Verify` and `GetTrustStatus`. |
| FR-POL-005 | Must | Policy **require-signatures** SHALL fail when no valid signature exists for the digest. |
| FR-POL-006 | Must | Policy **trusted-publishers**, when present in the active policy, SHALL fail when no signature’s signing identity matches the allowlist. |
| FR-POL-007 | Should | Policy **repository-ownership**, when present, SHALL fail when provenance repository does not match namespace-linked repository. |
| FR-POL-008 | Deferred | Policy **block-critical-cves** SHALL fail when critical CVEs are detected in attached SBOM. |
| FR-POL-009 | Must | Policy failures SHALL return `POLICY_FAILED` with rule id and remediation hint. |
| FR-POL-010 | Must | Policy changes SHALL be audit-logged with operator identity and timestamp. |
| FR-POL-011 | Must | Multiple configured rules SHALL compose; overall result fails if any configured rule fails. |
| FR-POL-012 | Should | All configured rules SHALL be evaluated at push-time on `SetTag` (v0.2 target; v0.1 may evaluate some rules only at verify-time — see release checklist). |

## Evaluation model

| Phase | Trigger | Rules run |
|-------|---------|-----------|
| **Push-time** | `SetTag`, end of `publish` | All configured rules (target); v0.1: `require-signatures` at minimum |
| **Verify-time** | `inspect`, `Verify` | All configured rules |

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
| AC-POL-002 | Given trusted-publishers configured with allowlisted workflow, when evaluating a matching signed digest, then evaluation passes. | FR-POL-006 |
| AC-POL-002b | Given trusted-publishers configured, when digest is signed only by a non-allowlisted workflow, then `POLICY_FAILED` and inspect exits non-zero. | FR-POL-006, US-POL-002 |
| AC-POL-003 | Given provenance claiming repo R1 but namespace linked to R2, when repository-ownership enabled, then evaluation fails with repository mismatch message. | FR-POL-007 |
| AC-POL-004 | Given inspect on passing artifact, when all configured policies pass, then trust report shows policy section all pass. | FR-POL-004, FR-POL-011 |
| AC-POL-005 | Given policy update by operator, when audit log queried, then change event includes policy version and operator id. | FR-POL-010 |

## Resolved open questions

| ID | Question | Resolution |
|----|----------|------------|
| OQ-POL-002 | Warn vs fail for configured policies? | **Fail-closed (Option A):** any rule in the active policy document hard-fails; no warn-only mode. |
| OQ-POL-001 | Push-time vs verify-time defaults? | **Both** for configured rules; full push-time for every rule type is **Should** (`FR-POL-012`, v0.2). v0.1 documents verify-time-only gap for `trusted-publishers` / `repository-ownership` on `SetTag`. |

## Open questions

| ID | Question | Cross-ref |
|----|----------|-----------|
| OQ-POL-003 | CVE data source and severity threshold when block-critical-cves is implemented? | Deferred |
| OQ-POL-004 | Generic OIDC signing identity matching beyond GitHub Actions workflow extensions? | [02-signing-verification.md](02-signing-verification.md) |
