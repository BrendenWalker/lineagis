# Lineagis Architecture Overview

Lineagis is a **software supply-chain lineage graph engine**.  
It models software systems as a **directed provenance graph** connecting:

- commits
- builds
- artifacts
- dependencies
- deployments

The core idea:  
> Everything is a node. Everything meaningful is an edge.

---

# 1. System Architecture (High Level)

Lineagis is composed of five layers:

## 1. Ingestion Layer
Responsible for collecting raw supply-chain signals.

Sources:
- CI/CD pipelines (GitHub Actions, GitLab CI, Jenkins)
- SBOMs (CycloneDX, SPDX)
- Container registries (OCI/Docker)
- Git metadata

Output:
- Normalized “raw events”

---

## 2. Normalization Layer
Transforms heterogeneous inputs into a unified schema.

Responsibilities:
- Normalize identifiers (hashes, tags, commits)
- Deduplicate events
- Map external formats → internal model

Output:
- Canonical Lineagis objects

---

## 3. Graph Core (Central System)

This is the **heart of Lineagis**.

Responsibilities:
- Store nodes and edges
- Maintain DAG integrity
- Support traversal queries
- Enforce consistency rules

Can be:
- In-memory (MVP)
- Persistent graph DB (v2+)

---

## 4. Query Engine

Provides semantic access to the graph.

Examples:
- trace ancestry
- compute dependency impact
- find upstream root causes
- detect broken lineage chains

---

## 5. CLI + API Layer

Interfaces:
- CLI (primary MVP interface)
- REST/GraphQL API (v2+)
- Optional UI dashboard (later)

---

# 2. Repository Structure

```
lineagis/
│
├── cmd/
│   └── lineagis/              # CLI entrypoint
│
├── internal/
│   ├── core/
│   │   ├── graph/             # Graph implementation (nodes + edges)
│   │   ├── model/             # Data models (Artifact, Commit, etc.)
│   │   ├── engine/            # Lineage reasoning engine
│   │   └── query/             # Query execution logic
│   │
│   ├── ingest/
│   │   ├── cicd/              # GitHub Actions, GitLab, Jenkins
│   │   ├── sbom/              # CycloneDX, SPDX parsers
│   │   ├── registry/          # Docker/OCI ingestion
│   │   └── git/               # Git metadata ingestion
│   │
│   ├── normalize/
│   │   ├── mapper/            # Format → canonical mapping
│   │   ├── dedupe/            # Event deduplication
│   │   └── resolver/          # Identity resolution (hash/tag mapping)
│   │
│   ├── storage/
│   │   ├── memory/            # MVP in-memory graph store
│   │   ├── neo4j/             # Optional graph DB backend
│   │   └── postgres/          # Relational edge store option
│   │
│   └── api/
│       ├── http/              # REST API (v2+)
│       └── graphql/           # Optional GraphQL layer
│
├── pkg/
│   └── client/                # External SDK for embedding Lineagis
│
├── docs/
│   ├── ARCHITECTURE.md
│   ├── DATA_MODEL.md
│   ├── QUERY_LANGUAGE.md
│   └── INTEGRATIONS.md
│
├── examples/
│   ├── sbom.json
│   ├── github-action.yaml
│   └── sample-graph.json
│
└── tests/
    ├── ingest/
    ├── graph/
    ├── query/
    └── integration/
```

---

# 3. Core Data Model

Lineagis uses a **typed graph model**.

## 3.1 Node Types

### Base Node Interface

All nodes share:

- id (canonical identifier)
- type
- metadata
- timestamps

---

### Commit Node

Represents a git commit.

```json
{
  "id": "commit:abc123",
  "type": "commit",
  "metadata": {
    "repo": "org/service",
    "sha": "abc123",
    "author": "dev",
    "timestamp": "..."
  }
}
```

---

### Build Node

Represents a CI/CD build execution.

```json
{
  "id": "build:build-789",
  "type": "build",
  "metadata": {
    "system": "github-actions",
    "pipeline": "ci.yml",
    "status": "success"
  }
}
```

---

### Artifact Node

Represents build outputs.

```json
{
  "id": "artifact:sha256:deadbeef",
  "type": "artifact",
  "metadata": {
    "type": "container-image",
    "name": "app:1.2.3",
    "digest": "sha256:deadbeef"
  }
}
```

---

### Dependency Node

Represents external or internal dependencies.

```json
{
  "id": "dependency:npm:lodash@4.17.21",
  "type": "dependency",
  "metadata": {
    "ecosystem": "npm",
    "name": "lodash",
    "version": "4.17.21"
  }
}
```

---

### Deployment Node

Represents runtime deployment events.

```json
{
  "id": "deployment:prod-001",
  "type": "deployment",
  "metadata": {
    "env": "production",
    "cluster": "k8s-prod"
  }
}
```

---

## 3.2 Edge Types (Critical)

Edges define lineage.

| Edge Type | Meaning |
|----------|--------|
| produced_by | artifact → build |
| built_from | build → commit |
| depends_on | artifact → dependency |
| deployed_to | artifact → deployment |
| derived_from | artifact → artifact |

---

### Edge Example

```json
{
  "from": "artifact:sha256:deadbeef",
  "to": "build:build-789",
  "type": "produced_by",
  "metadata": {
    "timestamp": "..."
  }
}
```

---

# 4. Graph Model Rules

## Rule 1: Directed Acyclic Graph (DAG)
- No cycles allowed in provenance chain

## Rule 2: Immutable edges
- Once written, edges are not modified (only appended)

## Rule 3: Identity normalization
- Same artifact must resolve to same canonical ID

## Rule 4: Time-aware edges (optional v2)
- edges can carry timestamps for historical reconstruction

---

# 5. Query Model

Lineagis supports graph traversal queries.

## Core Queries

### Trace lineage
```
lineagis trace artifact@sha256:abc
```

Meaning:
> Walk all upstream edges to root commits

---

### Impact analysis
```
lineagis impact commit@abc123
```

Meaning:
> What artifacts and deployments are affected?

---

### Upstream dependency
```
lineagis upstream artifact@xyz
```

---

### Downstream dependency
```
lineagis downstream artifact@xyz
```

---

## Query Execution Model

- BFS/DFS traversal on DAG
- Filtered by edge type
- Optional constraints:
  - time window
  - environment
  - repository scope

---

# 6. Storage Model Options

## MVP (v1.0)
- In-memory adjacency list
- NetworkX-style graph model

## v1.1+
- Postgres edge table:
  - nodes table
  - edges table

## v2+
- Graph DB backend:
  - Neo4j
  - Dgraph
  - TigerGraph (optional enterprise path)

---

# 7. Execution Flow

### Example: ingest → trace

1. Ingest SBOM / CI event
2. Normalize into canonical nodes
3. Insert nodes into graph
4. Create edges between entities
5. Persist (or store in memory)
6. Query engine traverses graph
7. CLI formats output

---

# 8. Design Philosophy

Lineagis is built on three principles:

### 1. Everything is a graph
No isolated artifacts.

### 2. Traceability is the primary primitive
Not validation, not reporting.

### 3. Deterministic outputs
Same inputs → same graph → same results

---

# 9. What this enables (future direction)

- full software supply chain observability
- root cause analysis of vulnerabilities
- dependency blast-radius calculation
- cross-repo provenance graphs
- CI/CD anomaly detection

---

# End of Architecture