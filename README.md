<p align="center">
  <img src="images/logo2.png" alt="Lineagis logo" width="240">
</p>

# Lineagis

> Lineagis is an open-source **software supply-chain lineage and provenance engine**. It builds a unified directed graph across pipelines, artifacts, dependencies, and source code so teams can trace, reason about, and secure their software supply chain.

**Core idea:** everything is a node; everything meaningful is an edge.

---

## Overview

Modern supply chains span CI/CD, registries, SBOM tools, and application code — but the relationships between them are rarely connected in one queryable model.

Lineagis fills the **missing graph layer** for software supply chain security: a deterministic provenance engine that links commits, builds, artifacts, and dependencies into a single DAG you can traverse from the CLI. **Repository self-analysis** extends that graph with Go module structure — packages, symbols, imports, docs, tests, and CI workflows — so the tool dogfoods on itself.

Rather than another siloed scanner or registry, Lineagis is a **lineage graph** — cross-tool provenance and code structure you can query.

## Goals

* Unify supply-chain signals into one provenance graph
* Trace artifact ancestry from runtime back to source commits
* Analyze Go repositories into a deterministic code knowledge graph
* Detect broken or incomplete lineage chains and architecture violations
* Support package-level impact and dependency exploration
* Expose a developer-first CLI with deterministic, reproducible outputs
* Build on open standards (OCI, CycloneDX, SPDX, Sigstore)

---

## Core Concepts

### Provenance graph (v1.0)

Lineagis models supply-chain events as a **typed directed acyclic graph (DAG)**:

| Node type | Represents |
|-----------|------------|
| **Commit** | A git revision |
| **Build** | A CI/CD pipeline run |
| **Artifact** | A build output (image, package, binary) |
| **Dependency** | An internal or external dependency |

| Edge type | Meaning |
|-----------|---------|
| `produced_by` | artifact → build |
| `built_from` | build → commit |
| `depends_on` | artifact → dependency |

Same inputs → same graph → same query results.

### Code graph (v1.1+)

`lineagis analyze` adds a **code subgraph** merged with provenance in `lineage-graph/v2`:

| Node type | Represents |
|-----------|------------|
| **Module** | Go module (in-repo or external `go.mod` require) |
| **Package** | Go import path |
| **File** | Source or test file |
| **Symbol** | Exported func, method, struct, interface |
| **Doc** | Markdown under `docs/` |
| **Workflow** | GitHub Actions workflow |

| Edge type | Meaning |
|-----------|---------|
| `contains` | module → package → file → symbol |
| `imports` | package → package |
| `documents` | doc → package |
| `tests` | test file → package |
| `introduced_by` | package → commit (when provenance is present) |

Import cycles among packages are allowed; they are reported, not rejected.

### System layers

Lineagis is organized into five layers (see [Architecture Overview](docs/lineagis_architecture_overview.md)):

```text
  ┌─────────────────────────────────────────┐
  │  CLI + API                              │
  ├─────────────────────────────────────────┤
  │  Query Engine   (trace, why, impact)    │
  ├─────────────────────────────────────────┤
  │  Graph Core     (nodes, edges, DAG)     │
  ├─────────────────────────────────────────┤
  │  Normalization  (dedupe, identity resolve)│
  ├─────────────────────────────────────────┤
  │  Ingestion      (SBOM, git, build, Go AST)│
  └─────────────────────────────────────────┘
```

**Ingestion** collects signals from SBOMs (CycloneDX / SPDX), git/build sidecars, and Go source (via `go/packages`). **Normalization** maps heterogeneous formats to canonical Lineagis objects. The **graph core** stores nodes and edges with DAG integrity on the provenance subgraph. The **query engine** runs lineage and package traversals. The **CLI** is the primary interface.

### Inputs

* SBOM JSON (CycloneDX, SPDX)
* Git commit and build sidecars
* Go module trees (`go.mod`, packages, AST)
* Markdown docs and GitHub Actions workflows (via `analyze`)

---

## CLI

Graph state is stored in `.lineagis/graph.json` by default (override with `--graph-in` / `--graph-out` or `LINEAGIS_GRAPH_FILE`).

### Provenance (v1.0)

```bash
# Ingest supply-chain sidecars
lineagis ingest examples/sbom-cyclonedx.json examples/build-sidecar.json examples/commit-sidecar.json

# Trace lineage to root commits
lineagis trace artifact@sha256:abc123

# Explain why an artifact exists in the graph
lineagis why artifact@sha256:abc123

# Visualize the DAG (Graphviz DOT)
lineagis visualize artifact@sha256:abc123 --format dot
```

### Self-analysis (v1.1+)

```bash
# Analyze a Go module (merges with any ingested provenance)
lineagis analyze .
lineagis analyze . --format json
lineagis analyze . --format dot > imports.dot

# Validate architecture rules (lineagis.arch.yaml) and emit reports
lineagis analyze . --validate-arch --out generated

# Regenerate report artifacts from the saved graph
lineagis report --out generated

# Package-level exploration
lineagis why package github.com/BrendenWalker/lineagis/internal/core/graph
lineagis impact package github.com/BrendenWalker/lineagis/internal/core/graph
lineagis explain dependency golang.org/x/tools
```

`analyze --out generated` writes architecture markdown, dependency reports, import diagrams, and `lineage.json` under `generated/`. See [self-analysis.md](docs/specs/self-analysis.md).

### Example: provenance + code in one session

```bash
lineagis ingest examples/sbom-cyclonedx.json examples/build-sidecar.json examples/commit-sidecar.json
lineagis analyze . --validate-arch
lineagis trace artifact@sha256:abc123
lineagis impact package github.com/BrendenWalker/lineagis/internal/core/graph
```

---

## Roadmap

| Version | Focus | Status |
|---------|-------|--------|
| **v1.0** | Graph MVP | **Shipped** — SBOM/git/build ingest, `trace` / `why`, JSON + Graphviz |
| **v1.1** | Self-analysis | **Shipped** — `analyze`, code graph, docs/tests/workflows, CI self-analysis |
| **v1.2** | Reports & exploration | **Shipped** — generated reports, arch rules, package `why` / `impact` / `explain`, release artifacts |
| **v1.3+** | Multi-source & scale | Registry/attestation ingest, persistent graph DB, REST API, UI |

Full roadmap: [docs/lineagis_design.md](docs/lineagis_design.md).

---

## Architecture

The engine is **CLI-only and offline-capable**: ingest → in-memory graph → query. No API or database is required.

Repository layout: [docs/lineagis_architecture_overview.md#2-repository-structure](docs/lineagis_architecture_overview.md#2-repository-structure).

| Layer | Choice |
|-------|--------|
| Language | Go |
| Graph store | In-memory; snapshot file (`.lineagis/graph.json`, schema `lineage-graph/v2`) |
| Provenance ingest | CycloneDX / SPDX JSON, git/build sidecars |
| Code ingest | `go/packages`, `go.mod`, docs, GitHub Actions YAML |
| Architecture rules | `lineagis.arch.yaml` (layer import constraints) |

---

## Development

[![CI](https://github.com/BrendenWalker/lineagis/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/BrendenWalker/lineagis/actions/workflows/ci.yml)

### Prerequisites

* Go 1.25 or newer (see `go.mod`)
* [golangci-lint](https://golangci-lint.run/welcome/install/) v2 (for local linting)
* Bash (for smoke scripts; Git Bash on Windows)

### Build and test

```bash
make build                 # bin/lineagis
make test-lineage          # graph engine + conformance tests
make lint
make smoke-lineage         # ingest → trace → why smoke
make smoke-analyze         # analyze . with architecture validation
make smoke-release-artifacts  # analyze + generated/ + lineage.json
```

Run the CLI:

```bash
./bin/lineagis --version
```

CI runs on every pull request and push to `main`. Required checks: `lint`, `test`, `build`, `smoke-lineage`, `self-analysis`. See [.github/BRANCH_PROTECTION.md](.github/BRANCH_PROTECTION.md).

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
| [Self-analysis spec](docs/specs/self-analysis.md) | FR-SA / AC-SA requirements, `analyze`, reports, arch rules |
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
