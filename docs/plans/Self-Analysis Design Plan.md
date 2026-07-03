# Lineagis Self-Analysis Design Plan

> **Normative requirements:** [docs/specs/self-analysis.md](../specs/self-analysis.md) (`FR-SA-*`, `AC-SA-*`, SA-P1–SA-P10). This document is the informative vision; implement against the spec.

## Vision

Lineagis should become its own flagship example by continuously analyzing and documenting itself.

Rather than describing what Lineagis *can* do, the repository should **demonstrate** those capabilities on every commit, every release, and every pull request.

The end goal is for a new visitor to understand the project simply by exploring the generated lineage graph.

> **Everything in this repository should be explainable by Lineagis.**

This makes the repository:

* Living documentation
* A continuously validated architectural model
* A reference implementation
* A showcase for best practices
* A regression test for the lineage engine itself

---

# Goals

## Primary Goal

Use Lineagis to analyze the Lineagis repository.

The repository should be capable of answering questions such as:

* Why does this package exist?
* What imports it?
* What depends on it?
* Which tests validate it?
* Which documentation describes it?
* Which CLI commands expose it?
* Which commit introduced it?
* What breaks if it changes?

---

## Secondary Goals

* Demonstrate graph-based software analysis
* Produce living architecture documentation
* Detect architectural drift
* Validate dependency boundaries
* Generate release artifacts automatically
* Showcase practical benefits of provenance and lineage

---

# Guiding Principles

## Everything is a Node

The graph should contain first-class representations of:

* Go modules
* Packages
* Files
* Structs
* Interfaces
* Functions
* Methods
* Tests
* Benchmarks
* Documentation
* Examples
* Make targets
* GitHub Actions
* Releases
* Commits
* Build artifacts
* External dependencies

---

## Everything is Connected

Relationships should include:

* imports
* implements
* calls
* creates
* reads
* writes
* depends_on
* tests
* documents
* builds
* deploys
* introduced_by
* modified_by
* generated_from

---

## Documentation Should Be Generated

Architecture documentation should be derived from the lineage graph rather than maintained manually.

Generated documentation should become the canonical source of truth.

---

# Phase 1 — Repository Self-Analysis

Introduce a top-level command such as:

```bash
lineagis analyze .
```

or

```bash
lineagis ingest repo .
```

The command should:

* Parse the repository
* Build a lineage graph
* Store graph objects
* Generate reports
* Produce visualization artifacts

This should become the primary demonstration of the project.

---

# Phase 2 — Repository Knowledge Graph

Expand the graph beyond source code.

## Source Code

Capture:

* packages
* files
* imports
* functions
* methods
* interfaces
* structs

---

## Build System

Capture:

* Make targets
* build outputs
* generated files

---

## CI/CD

Capture:

* workflows
* jobs
* artifacts
* releases

---

## Documentation

Treat documentation as graph objects.

Examples:

* README
* Architecture docs
* ADRs
* Examples

Documentation should link to:

* packages
* commands
* APIs
* implementation

Broken documentation links become broken lineage edges.

---

# Phase 3 — Dependency Intelligence

Dependencies should become first-class graph objects.

Instead of simply listing modules from `go.mod`, provide contextual information.

Example:

```text
Dependency

github.com/spf13/cobra

Purpose
CLI framework

Imported By
cmd/
api/

Used By
17 files

Introduced
Commit abc123

Impact
Removing this dependency disables CLI support.
```

Each dependency should answer:

* Why is it here?
* Who uses it?
* When was it introduced?
* What breaks if removed?
* Can it be replaced?

---

# Phase 4 — Living Architecture

Generate architecture documentation directly from the graph.

Examples:

* Package hierarchy
* Import graph
* Layer diagrams
* Dependency graph
* Call graph
* Data flow graph

Generated metrics:

* Package count
* Function count
* Dependency depth
* Cycles
* Strongly connected components
* Fan-in
* Fan-out

Architecture documentation should never become stale.

---

# Phase 5 — Self-Generated Reports

Create a generated directory such as:

```text
generated/
    architecture/
    lineage/
    reports/
    diagrams/
```

Artifacts may include:

* architecture.svg
* dependency-graph.svg
* imports.csv
* lineage.json
* blast-radius.md
* dead-code.md
* orphan-packages.md
* dependency-report.md

These should be regenerated automatically.

---

# Phase 6 — CI Integration

Extend CI so that every pull request runs repository self-analysis.

Pipeline:

```
lint
↓

test
↓

build
↓

self-analysis
↓

validate architecture
↓

publish artifacts
```

CI should verify:

* no dependency cycles
* no orphan packages
* no unused dependencies
* architecture rules
* documentation consistency

Generated reports should be attached to pull requests.

---

# Phase 7 — Architectural Fitness Functions

Define explicit architectural rules.

Examples:

```
cmd

↓

api

↓

query

↓

graph

↓

storage
```

Rules:

* cmd cannot import storage
* graph cannot import cli
* storage cannot import cmd
* examples cannot import internal packages directly

Lineagis should validate these automatically.

---

# Phase 8 — Impact Analysis

Every pull request should generate an impact report.

Example:

```
Modified Package

internal/graph

Impacts

12 packages

97 functions

38 tests

3 CLI commands

2 generated reports
```

This demonstrates one of the strongest practical benefits of lineage analysis.

---

# Phase 9 — Repository Explorer

Provide graph queries that explain the repository.

Examples:

```
Why does this package exist?

Who depends on it?

Which tests cover it?

Which documentation references it?

Which commands expose it?

Which release introduced it?
```

Potential CLI:

```bash
lineagis why package graph

lineagis trace package graph

lineagis impact package graph

lineagis explain dependency github.com/spf13/cobra
```

---

# Phase 10 — Release Artifacts

Every release should publish generated lineage assets.

Examples:

* lineage.json
* architecture.svg
* dependency-report.md
* blast-radius.md
* call-graph.graphml
* import-graph.svg

This demonstrates deterministic analysis across versions.

---

# Recommended Repository Structure

```
generated/
    architecture/
    lineage/
    reports/
    diagrams/

examples/
    self-analysis/

docs/
    architecture/
    lineage/

cmd/

internal/

pkg/
```

---

# Long-Term Vision

The repository should evolve into a complete software knowledge graph.

Graph nodes may include:

* packages
* files
* functions
* interfaces
* tests
* documentation
* workflows
* releases
* commits
* dependencies
* generated artifacts

Relationships should allow exploration of the entire project.

Example questions:

* What depends on this package?
* What introduced this dependency?
* Which tests validate this feature?
* Which documentation explains this component?
* What changed between releases?
* What is the blast radius of this modification?

---

# Success Criteria

The implementation should enable the repository to continuously explain itself.

A successful implementation will:

* Generate its own architecture documentation
* Explain every dependency
* Validate architectural rules
* Produce impact analysis automatically
* Publish lineage artifacts during CI
* Detect architectural drift
* Serve as the primary demonstration of Lineagis capabilities

---

# Ultimate Objective

The strongest demonstration of Lineagis is not a synthetic example—it is Lineagis itself.

A single command:

```bash
lineagis analyze .
```

should produce a complete, navigable representation of the repository, including source code, dependencies, documentation, build system, CI workflows, releases, and provenance.

If Lineagis can continuously explain **why Lineagis is built the way it is**, it becomes more than a provenance engine—it becomes a living, self-documenting software knowledge graph and the definitive reference implementation of its own design philosophy.
