# Security

## Reporting vulnerabilities

If you discover a security issue, please report it responsibly. Open a private security advisory on GitHub or contact the maintainers directly. Do not file public issues for undisclosed vulnerabilities.

## Threat model (summary)

Lineagis v1.0 is an **offline lineage graph CLI**. Trust decisions are only as good as the inputs you ingest (SBOMs, build metadata, commit sidecars). The tool does **not** verify signatures, scan for malware, or attest to registry contents in v1.0.

- **In scope:** integrity of graph construction (DAG rules, deterministic IDs), safe handling of local files, clear failure modes on broken lineage.
- **Out of scope (v1.0):** artifact signing, policy enforcement, remote registry/API trust.

## Operational guidance

| Practice | Rationale |
|----------|-----------|
| **Pin artifact digests** | Use `sha256:…` refs in `trace` / `why`; do not rely on mutable names alone |
| **Protect graph snapshots** | `.lineagis/graph.json` encodes your provenance model; treat like sensitive build metadata |
| **Validate inputs** | Ingest SBOMs and sidecars from trusted CI paths, not untrusted uploads |
| **Run `why` in CI** | Non-zero exit when lineage is incomplete before release |
| **Do not commit secrets** | Sidecars and SBOM paths should not embed tokens or credentials |

## What Lineagis does not protect against

- Compromised build pipelines or forged SBOMs
- Malicious dependencies that appear correctly in an SBOM
- Registry substitution without digest pinning (v1.1+ registry ingest)
- Vulnerability/CVE analysis

## Related documentation

- [Lineage MVP spec](docs/specs/lineage-engine-mvp.md)
- [Architecture overview](docs/lineagis_architecture_overview.md)
