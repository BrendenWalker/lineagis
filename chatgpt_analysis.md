# Verity security analysis

## Executive summary

Verity is best understood as a **software-release trust layer**, not a
package repository. Its goal is to make artifacts verifiable by binding
them to immutable OCI digests, Sigstore signatures, provenance
attestations, SBOMs, and namespace policies. The README and specs describe
artifact publishing, signing, provenance, policy enforcement, and
inspect-time verification as the central product workflow.

Important caveat: much of the strongest security value is still
**specification / roadmap**, not fully proven implementation. The repo
currently has concrete CLI/API scaffolding and API route wiring, but the
documented security model depends on features like Sigstore verification,
provenance validation, trusted publisher policy, repository ownership
checks, and SBOM/CVE evaluation.

---

# What Verity does

Verity provides a workflow where maintainers publish artifacts through a
CLI/API flow, store content in OCI-compatible infrastructure, attach trust
metadata, and allow consumers or CI systems to run `verity inspect` before
accepting the artifact.

Its architecture separates content distribution from trust metadata:

- OCI registry stores blobs/manifests
- Verity API manages trust metadata
- PostgreSQL stores artifact metadata, tags, policies, signatures,
provenance, and trust state

The intended publish flow is:

1. Upload artifact content to an OCI registry
2. Register digest/tag metadata with Verity
3. Sign the artifact digest
4. Attach signatures, provenance, and SBOM metadata
5. Evaluate policy before the tag becomes trusted

The intended verify flow is:

1. Resolve artifact by digest or tag
2. Verify signatures and attestations
3. Evaluate active policies
4. Return a trust report

---

# Supply-chain risks mitigated

| Risk | How Verity mitigates it | How it can be bypassed or weakened |
|---|---|---|
| Registry tampering | Artifacts are addressed by immutable SHA-256 OCI
digests; signatures cover manifest digests | Consumers using mutable tags
without digest verification remain vulnerable to tag movement |
| Unsigned malicious releases | `require-signatures` policy rejects
unsigned artifacts | Weak/misconfigured policy or optional verification
bypasses protection |
| Long-lived signing-key theft | Sigstore keyless signing avoids static
maintainer keys | Compromised CI workflows can still generate valid
signatures |
| Maintainer or registry-token compromise | Requires valid signing identity
+ policy + provenance | Repo/workflow compromise still allows valid
malicious builds |
| Fake provenance claims | Provenance attestations bind builds to artifact
digests | Unvalidated provenance becomes decorative metadata |
| Dependency opacity | SBOMs expose dependency inventory | Dishonest or
incomplete SBOMs can still pass |
| Untrusted publisher identity | Trusted publisher policies restrict valid
signers | Broad or missing publisher rules weaken identity guarantees |
| Silent tag overwrite | Audit logging + digest pinning reduce impact |
Mutable tags remain dangerous for non-pinned consumers |

---

# Core weaknesses

## 1. Trusting identity is not the same as trusting code

Verity proves:

> "This artifact was signed by this identity"  

It does **not** inherently prove:

> "This artifact is safe"  

If the trusted repository, maintainer account, or CI pipeline is
compromised, attackers can produce fully valid signed malicious artifacts.

This is the single largest limitation of modern supply-chain security
systems.

---

## 2. CI/CD compromise remains a critical attack path

An attacker who gains control over:

- GitHub Actions workflows
- Repository secrets
- Build infrastructure
- Merge permissions
- Release automation

can usually produce:

- valid signatures
- valid provenance
- valid SBOMs
- fully trusted artifacts

without triggering cryptographic verification failures.

Verity mitigates identity spoofing better than build compromise.

---

## 3. Many important controls are roadmap items

Several high-value security features are currently:

- partially implemented
- documented only
- planned for future releases

Examples include:

- repository ownership verification
- trusted publisher enforcement
- provenance validation
- SBOM enforcement
- CVE blocking
- transparency-log UX
- reproducible build verification

This means the *design* is stronger than the current implementation
maturity.

---

## 4. Trust reports can create false confidence

There is a risk that users interpret:

- signed
- verified
- trusted

as equivalent to:

- secure
- safe
- malware-free

That is not true.

A validly signed malicious artifact remains malicious.

The strongest value Verity provides is:

- attribution
- accountability
- tamper evidence
- policy enforcement

not malware prevention.

---

## 5. Metadata and policy infrastructure become critical assets

The Verity API and metadata database become part of the trusted computing
base.

If attackers can alter:

- trusted publisher lists
- namespace ownership
- tag mappings
- trust state
- policy configuration

they may weaken or bypass enforcement without touching artifact content.

This means:

- audit logging
- RBAC
- immutability
- change tracking
- policy write protection

become security-critical.

---

# Most important circumvention paths

## 1. Compromise the trusted CI workflow

Most realistic attack path.

If the build pipeline is trusted, compromised workflows produce trusted
malware.

---

## 2. Abuse broad publisher policy

Overly broad rules may trust:

- entire organizations
- all workflows
- multiple repositories

instead of exact trusted build identities.

---

## 3. Exploit tag mutability

Consumers using:

```bash
my-package:latest
```

instead of digest-pinned references remain vulnerable to:

- rollback attacks
- substitution attacks
- malicious tag movement

---

## 4. Skip verification entirely

If users pull artifacts directly from OCI registries without running:

```bash
verity inspect
```

the security model becomes optional.

Adoption discipline is essential.

---

## 5. Poison metadata

Attackers targeting:

- Verity API auth
- policy storage
- namespace ownership
- trust cache
- signature metadata

may weaken enforcement without modifying artifacts themselves.

---

## 6. Submit fake SBOMs or provenance

If attestations are:

- stored
- displayed
- but not strongly validated

then they become untrusted claims rather than trustworthy evidence.

---

## 7. Attack upstream dependencies

Verity does not inherently prevent:

- malicious transitive dependencies
- typosquatting
- compromised base images
- malicious build tools
- poisoned package mirrors

It primarily secures:

- release identity
- artifact integrity
- provenance traceability

---

# Security model strengths

Despite limitations, Verity demonstrates a strong modern security
architecture.

Its strongest design decisions are:

- OCI digest-addressed artifacts
- Sigstore keyless signing
- OIDC-based identity
- provenance attestations
- policy-driven trust enforcement
- separation of registry and trust metadata
- namespace ownership concepts
- inspect-time verification workflows

These align closely with:

- Sigstore
- SLSA
- in-toto
- modern container security practices

---

# Overall assessment

Verity is directionally strong and conceptually modern.

It combines the correct primitives for contemporary software supply-chain
security:

- immutable digests
- cryptographic signing
- identity-bound provenance
- policy enforcement
- metadata inspection
- trust evaluation

However, its current maturity appears closer to:

> "well-designed security platform specification with early implementation"  

than:

> "production-hardened trust infrastructure"  

The biggest remaining risks are:

1. trusted CI compromise
2. policy misconfiguration
3. optional verification workflows
4. metadata integrity attacks
5. insufficient provenance validation
6. over-trusting signed artifacts

The next critical security improvements should focus on:

- fail-closed verification
- strict trusted publisher enforcement
- immutable audit logging
- repository ownership validation
- reproducible build verification
- mandatory provenance validation
- stronger policy scoping
- digest-pinned consumption
- hardened metadata integrity protections

If implemented correctly, Verity could become a compelling lightweight
trust layer for OCI-distributed software artifacts and CI-driven software
release verification.