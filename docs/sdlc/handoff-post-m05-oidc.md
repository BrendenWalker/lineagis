# Hand-off prompts — post M05 OIDC (#8 partial)

Copy a block into a new agent session. Branch from `main` after the OIDC PR merges unless noted.

---

## 1. Finish E05 / issue #8 (operator role + remaining API core)

```
Continue Lineagis post-M05 OIDC. main includes PR for story/8-oidc-auth (GitHub OIDC + dev bearer).

Done: JWT verify (go-oidc), LINEAGIS_OIDC_ISSUER/AUDIENCE, dev token fallback, gh/* namespace repository/ref checks on RegisterDigest/SetTag/putArtifact.
Open: epic #8 still open — operator role gate, GetArtifact/GetTag/GetTrustStatus, ListArtifacts (Should).

Read AGENTS.md, docs/specs/api.md, docs/specs/architecture.md.
Goal: close remaining E05 DoD on branch story/8-api-core-roles (or split PRs).
Branch from main. No push/merge unless I ask.
```

---

## 2. Push-time policy on SetTag — issue #43

```
Continue Lineagis. main @ <sha> includes M05 OIDC and M06 publish.

Done: SetTag calls PushPolicy.AllowSetTag before commit; AllowAllPolicy stub.
Open: #43 — real evaluator stub (e.g. require-signature) wired for FR-API-007 / AC-API-002.

Read docs/specs/04-policy-enforcement.md, internal/api/policy.go, handlers putSetTag.
Goal: SetTag returns POLICY_FAILED when policy rejects; tests with DB.
Branch story/43-settag-policy from main. No push/merge unless I ask.
```

---

## 3. Cosign keyless signing in publish — issue #44

```
Continue Lineagis. main includes lineagis publish + API RegisterDigest/SetTag; OIDC for API auth.

Open: #44 / E08 — integrate Sigstore/cosign so publish signs manifest digest by default (FR-SIGN-001–004).

Read docs/specs/02-signing-verification.md, cmd/lineagis/publish.go, internal/publish/.
Goal: after OCI push, sign digest and AttachSignature (API) or document interim CLI-only attach.
Branch story/44-cosign-publish from main. No push/merge unless I ask.
```

---

## 4. lineagis inspect + E2E — issue #55

```
Continue Lineagis. main has publish; signing/policy may be partial.

Open: #55 — AC-OV-001/002 automated e2e: publish → inspect signed artifact.
Blocked until: lineagis inspect command (FR-DX-002, FR-SIGN-005) exists.

Read docs/specs/05-developer-experience.md, docs/specs/00-overview.md inspect flow.
Goal: minimal inspect checklist + integration test or smoke script in CI.
Branch story/55-inspect-e2e from main. No push/merge unless I ask.
```
