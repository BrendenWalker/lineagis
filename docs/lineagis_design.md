# Lineagis – Unified Software Lineage & Provenance Engine

**Lineagis** is a deterministic supply-chain integrity and provenance engine.  
It builds a unified graph across pipelines, artifacts, and dependencies to help teams trace, reason, and secure their software supply chain.

---

## Table of Contents
1. Project Rename
2. MVP Roadmap (v1.0 → v1.2)
3. Integration Plan
4. MVP-to-v2 Roadmap
5. Early Adoption / Proof-of-Concept
6. Documentation / Onboarding
7. Long-term Strategic Positioning
8. Next Actionable Steps

---

## Project Rename

- GitHub: `BrendenWalker/verity` → `BrendenWalker/lineagis` *(repo rename pending)*
- Topics: supply-chain, provenance, SBOM, lineage, verification

- Packages:
  - Python: `verity` → `lineagis`
  - Go: `github.com/BrendenWalker/lineagis`
  - CLI: `verity` → `lineagis`

- Docs:
  - Update README, CONTRIBUTING, docs
  - Optional branding refresh (logo/colors)

---

## MVP Roadmap (v1.0 → v1.2)

### Core Modules

| Module | Function |
|--------|----------|
| Graph Core | DAG of commits, builds, artifacts, dependencies |
| Ingestion | CI/CD, SBOMs, container registries |
| Lineage Engine | Builds relationships and ancestry queries |
| Verification Layer | Basic checks (missing nodes, signatures) |
| Query Interface | CLI commands |
| Output | JSON + human-readable summary + optional Graphviz output |

---

### V1.0 – Core MVP

- Inputs:
  - SBOM JSON (CycloneDX / SPDX)
  - Git commit metadata
  - Build artifacts (hashes, images, packages)

- Graph model:
  - Nodes: commit → build → artifact → dependency
  - Edges: produced_by, depends_on

- CLI:

  lineagis ingest sbom.json  
  lineagis trace artifact@sha256:abc123  
  lineagis why artifact@sha256:abc123  

- Output:
  - JSON + readable summary
  - Optional Graphviz DAG visualization

---

### V1.1 – Multi-Source Integration

- Repository self-analysis (`lineagis analyze .`) — dogfood code + provenance graph ([self-analysis.md](specs/self-analysis.md))
- GitHub Actions / GitLab CI ingestion
- Container registry ingestion (Docker/OCI)
- Basic anomaly detection:
  - missing artifacts
  - unexpected dependencies
- Incremental persistence

---

### V1.2 – Expanded Capability

- Cross-source lineage graph (SBOM + CI + containers)
- CLI expansion:

  lineagis impact commit@xyz  
  lineagis upstream artifact@sha256:abc123  
  lineagis downstream artifact@sha256:abc123  

- Reporting:
  - JSON/YAML exports
  - Optional HTML DAG view

- Optional:
  - Sigstore integration for attestations

---

## Integration Plan

### CI/CD
- GitHub Actions
- GitLab CI
- Jenkins
- Azure DevOps / CircleCI (optional)

### Artifact Systems
- Docker / OCI registries
- npm / PyPI / Maven
- Artifactory / Nexus (optional)

### SBOM / Security Tools
- CycloneDX
- SPDX
- Syft
- Grype

### Storage / Graph Layer
- MVP: in-memory DAG (e.g., NetworkX)
- Scale: Neo4j / Dgraph / Postgres edge model
- Optional GraphQL layer

---

## MVP → v2 Roadmap

| Phase | Goal | Deliverables |
|------|------|--------------|
| v2.0 | Ecosystem expansion | Full CI/CD + persistent graph + UI |
| v2.1 | Advanced queries | Impact analysis, upstream/downstream, risk scoring |
| v2.2 | Automation | Alerts, anomaly detection, policy hooks |
| v2.3 | Multi-repo graphs | Cross-project lineage |
| v2.4 | Compliance layer | SOC2 / SLSA reporting |

---

## Early Adoption

- Ingest real CI pipeline + SBOM data
- Demonstrate:
  - commit → build → artifact → runtime traceability
  - missing dependency detection
- Collect feedback:
  - CLI usability
  - graph expressiveness
  - ingestion coverage

---

## CLI Examples

  # Ingest SBOM  
  lineagis ingest sbom.json  

  # Trace lineage  
  lineagis trace myapp:1.2.3  

  # Upstream commits  
  lineagis upstream artifact@sha256:abc123  

  # Visualize graph  
  lineagis visualize artifact@sha256:abc123  

---

## Strategic Positioning

- “Missing graph layer for software supply chain security”
- Differentiators:
  - cross-tool provenance
  - unified lineage graph
  - queryable dependency ancestry
  - developer-first CLI

---

## Next Steps

1. ~~Rename repo to Lineagis~~ (codebase rename done; GitHub repo rename pending)
2. Define MVP v1.0 (graph ingest + trace)
3. Build v1.1 integrations (CI/CD + SBOM + containers)
4. Add persistence layer
5. Release MVP
6. Expand to v2 (multi-repo + automation + UI)