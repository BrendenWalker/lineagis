<p align="center">
  <img src="images/logo2.png" alt="Lineagis logo" width="240">
</p>

# Lineagis

> Lineagis is an open-source **software supply-chain lineage and provenance engine**. It builds a unified directed graph across pipelines, artifacts, and dependencies so teams can trace, reason about, and secure their software supply chain.

**Core idea:** everything is a node; everything meaningful is an edge.

---

## Overview

Modern supply chains span CI/CD, registries, SBOM tools, and runtime environments — but the relationships between them are rarely connected in one queryable model.

Lineagis fills the **missing graph layer** for software supply chain security: a deterministic provenance engine that links commits, builds, artifacts, dependencies, and deployments into a single DAG you can traverse from the CLI.

Rather than another siloed scanner or registry, Lineagis is a **lineage graph** — cross-tool provenance you can query.

## Goals

* Unify supply-chain signals into one provenance graph
* Trace artifact ancestry from runtime back to source commits
* Detect broken or incomplete lineage chains
* Support impact and blast-radius analysis across dependencies
* Expose a developer-first CLI with deterministic, reproducible outputs
* Build on open standards (OCI, CycloneDX, SPDX, Sigstore)

---

## Core Concepts

### Provenance graph

Lineagis models software as a **typed directed acyclic graph (DAG)**:

| Node type | Represents |
|-----------|------------|
| **Commit** | A git revision |
| **Build** | A CI/CD pipeline run |
| **Artifact** | A build output (image, package, binary) |
| **Dependency** | An internal or external dependency |
| **Deployment** | A runtime deployment event |

| Edge type | Meaning |
|-----------|---------|
| `produced_by` | artifact → build |
| `built_from` | build → commit |
| `depends_on` | artifact → dependency |
| `deployed_to` | artifact → deployment |
| `derived_from` | artifact → artifact |

Same inputs → same graph → same query results.

### System layers

Lineagis is organized into five layers (see [Architecture Overview](docs/lineagis_architecture_overview.md)):

```text
  ┌─────────────────────────────────────────┐
  │  CLI + API                              │
  ├─────────────────────────────────────────┤
  │  Query Engine   (trace, impact, upstream)│
  ├─────────────────────────────────────────┤
  │  Graph Core     (nodes, edges, DAG)     │
  ├─────────────────────────────────────────┤
  │  Normalization  (dedupe, identity resolve)│
  ├─────────────────────────────────────────┤
  │  Ingestion      (CI, SBOM, registry, git)│
  └─────────────────────────────────────────┘
```

**Ingestion** collects raw signals from CI/CD, SBOMs (CycloneDX / SPDX), container registries, and git metadata. **Normalization** maps heterogeneous formats to canonical Lineagis objects. The **graph core** stores nodes and edges with DAG integrity. The **query engine** runs traversals (ancestry, impact, upstream/downstream). The **CLI** is the primary interface for MVP; REST/GraphQL and a UI come later.

### Inputs (target v1.0)

* SBOM JSON (CycloneDX, SPDX)
* Git commit metadata
* Build artifacts (hashes, images, packages)
* CI/CD pipeline events (GitHub Actions, GitLab CI, Jenkins)
* Container registry manifests (OCI/Docker)

---

## CLI (target v1.0)

The graph-first CLI is the north-star interface:

```bash
# Ingest supply-chain data
lineagis ingest sbom.json

# Trace lineage to root commits
lineagis trace artifact@sha256:abc123

# Explain why an artifact exists in the graph
lineagis why artifact@sha256:abc123

# Visualize the DAG (optional Graphviz output)
lineagis visualize artifact@sha256:abc123
```

Planned v1.1–v1.2 commands include `impact`, `upstream`, and `downstream` for cross-source graphs and dependency blast-radius queries. See [Design & Roadmap](docs/lineagis_design.md).

### Example: ingest and trace (v1.0)

```bash
lineagis ingest examples/sbom-cyclonedx.json examples/build-sidecar.json examples/commit-sidecar.json
lineagis trace artifact@sha256:abc123
lineagis why artifact@sha256:abc123
```

Graph state is stored in `.lineagis/graph.json` by default (override with `--graph-in` / `--graph-out` or `LINEAGIS_GRAPH_FILE`). See [lineage-engine-mvp.md](docs/specs/lineage-engine-mvp.md).

---

## Roadmap

| Version | Focus | Highlights |
|---------|-------|------------|
| **v1.0** | Graph MVP | In-memory DAG, SBOM/git/artifact ingest, `trace` / `why`, JSON + Graphviz output |
| **v1.1** | Multi-source | GitHub Actions / GitLab CI / registry ingestion, anomaly detection, persistence |
| **v1.2** | Cross-graph queries | `impact`, `upstream`, `downstream`, HTML/YAML exports, optional Sigstore attestations |
| **v2.x** | Scale & automation | Persistent graph DB, UI, alerts, multi-repo graphs, compliance reporting |

Full roadmap and integration plan: [docs/lineagis_design.md](docs/lineagis_design.md).

---

## Architecture

v1.0 is a **CLI-only, offline-capable** graph engine: ingest → in-memory DAG → query. No API or database required for MVP.

Repository layout: [docs/lineagis_architecture_overview.md#2-repository-structure](docs/lineagis_architecture_overview.md#2-repository-structure).

### Technology

| Layer | Choice (v1.0) |
|-------|----------------|
| Language | Go |
| Graph store | In-memory DAG; snapshot file (`.lineagis/graph.json`) |
| Ingest | CycloneDX / SPDX JSON, git/build sidecars |
| Graph store (scale) | Postgres, Neo4j (v1.1+, per roadmap) |

---

## Development

[![CI](https://github.com/BrendenWalker/lineagis/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/BrendenWalker/lineagis/actions/workflows/ci.yml)

### Prerequisites

* Go 1.23 or newer
* [golangci-lint](https://golangci-lint.run/welcome/install/) v2 (for local linting)
* Bash (for `make smoke-lineage`; Git Bash on Windows)

### Build and test

```bash
make build          # bin/lineagis
make test-lineage   # graph engine + conformance tests
make lint
make smoke-lineage  # ingest → trace → why smoke
```

Run the CLI:

```bash
./bin/lineagis --version
```

CI runs on every pull request and push to `main`. Required checks: `lint`, `test`, `build`, `smoke-lineage`. See [.github/BRANCH_PROTECTION.md](.github/BRANCH_PROTECTION.md).

<details>
<summary>Windows development notes</summary>

Go installs to `C:\Program Files\Go\bin`. Restart your terminal after install if `go` is not found.

**Git Bash** — add to `~/.bashrc` if needed:

```bash
export PATH="$PATH:/c/Program Files/Go/bin"
```

**GNU Make** — use MinGW's `mingw32-make` (not Embarcadero `make`):

```powershell
mingw32-make test; mingw32-make build
```

PowerShell 5.1 does not support `&&`. On Windows, run tests without `-race` (requires CGO). Use `.\bin\lineagis.exe --version`.

</details>

---

## Documentation

| Document | Description |
|----------|-------------|
| [Architecture Overview](docs/lineagis_architecture_overview.md) | Graph model, layers, queries, storage options |
| [Design & Roadmap](docs/lineagis_design.md) | MVP v1.0–v1.2, integrations, strategic positioning |
| [Lineage MVP spec](docs/specs/lineage-engine-mvp.md) | FR-LIN / AC-LIN requirements and conformance fixtures |
| [Specs index](docs/specs/README.md) | Specification index |
| [Security](SECURITY.md) | Vulnerability reporting |

---

## License

Licensed under the Apache License 2.0.

---

## Vision

Lineagis aims to be the **queryable graph layer** the supply chain has been missing:

* cross-tool provenance in one model
* traceability as the primary primitive
* deterministic outputs for automation and compliance
* open standards, open governance, developer-first tooling

Software supply chains should be **observable**, not opaque.
