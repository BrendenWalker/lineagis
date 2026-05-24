# Verity

> Verity is an open-source trust platform for publishing, verifying, and governing software artifacts with built-in supply chain security.

---

## Overview

Modern software supply chains are fragmented, opaque, and increasingly vulnerable to compromise.

Verity aims to provide a secure, open, and verifiable foundation for software artifact distribution using OCI-native infrastructure, cryptographic signing, provenance attestations, and policy enforcement.

Rather than acting as “another package repository,” Verity is designed as a trust layer for software releases.

## Goals

* Make software releases verifiable by default
* Help open-source maintainers publish trusted artifacts
* Improve visibility into software provenance and integrity
* Provide portable supply-chain security primitives
* Build on open standards and open infrastructure

---

# Core Concepts

## OCI-Native Distribution

Verity uses OCI registries as the underlying distribution and storage layer for artifacts.

This enables:

* content-addressed storage
* immutable artifact digests
* efficient replication
* shared ecosystem tooling
* standardized distribution APIs

Artifacts may include:

* packages
* binaries
* containers
* SBOMs
* provenance attestations
* signatures
* release metadata

---

## Supply Chain Trust

Verity is built around software trust primitives:

* cryptographic signing
* provenance verification
* CI/CD identity validation
* transparency and auditability
* policy enforcement

The goal is to answer questions like:

* Who built this artifact?
* Which repository produced it?
* Which workflow published it?
* Was the artifact modified?
* Does it meet organizational policy requirements?

---

# MVP Scope

The initial MVP focuses on a minimal but compelling trust workflow for open-source maintainers.

## Initial Features

### Artifact Publishing

* OCI artifact push/pull
* immutable digests
* semantic version tagging

### Signing & Verification

* Sigstore integration
* keyless signing
* signature verification
* provenance verification

### Provenance & Metadata

* SLSA-style attestations
* source repository linkage
* git commit tracking
* CI workflow identity
* SBOM attachment

### Policy Enforcement

Initial policy support:

* require signatures
* restrict trusted publishers
* verify repository ownership
* block known critical vulnerabilities

### Developer Experience

* CLI-first workflow
* GitHub Actions integration
* simple verification tooling

---

# Example Workflow

## Publish

```bash
verity publish dist/*
```

Automatically:

* signs artifacts
* generates provenance
* uploads metadata
* attaches attestations
* publishes to OCI storage

---

## Verify

```bash
verity inspect package.whl
```

Example output:

```text
✓ Signed by GitHub Actions
✓ Repository verified
✓ Maintainer verified
✓ SBOM attached
✓ Provenance verified
✓ No critical vulnerabilities detected
```

---

# Architecture

```text
                +-------------------+
                | Verity CLI        |
                +-------------------+
                         |
                         v
                +-------------------+
                | Verity API        |
                +-------------------+
                    |          |
                    v          v
           +-------------+   +----------------+
           | OCI Registry|   | Metadata DB    |
           +-------------+   +----------------+
                    |
                    v
           +------------------+
           | Object Storage   |
           +------------------+
```

---

# Technology Direction

## Planned Stack

### Backend

* Go

### Storage

* OCI Distribution Spec
* S3-compatible object storage

### Database

* PostgreSQL

### Identity & Signing

* Sigstore
* OIDC
* GitHub Actions identity

---

# Non-Goals (Initial MVP)

The initial release intentionally avoids:

* enterprise RBAC complexity
* multi-region replication
* billing/multi-tenancy
* ecosystem parity with Artifactory
* advanced package search
* Kubernetes operators
* proprietary extensions

The focus is trust, provenance, and verification.

---

# Development

[![CI](https://github.com/BrendenWalker/verity/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/BrendenWalker/verity/actions/workflows/ci.yml)

## Prerequisites

* Go 1.23 or newer
* [golangci-lint](https://golangci-lint.run/welcome/install/) v2 (for local linting)

```bash
choco install golangci-lint
```

## Build and test

```bash
make build    # produces bin/verity
make test     # race detector + coverage.out
make lint     # golangci-lint
```

Run the placeholder CLI:

```bash
./bin/verity --version
```

CI runs on every pull request and on pushes to `main` (build, test, lint). Coverage is uploaded as a workflow artifact when tests run.

---

# Documentation

Detailed MVP requirements live in [docs/specs/](docs/specs/README.md): an overview with delivery matrix (Must / Should / Deferred), foundation specs (architecture, API, metadata model), and feature specs for publishing, signing, provenance, policy, and developer experience.

---

# Roadmap

## Phase 1

* OCI-native artifact publishing
* metadata persistence
* CLI workflows
* signature support

## Phase 2

* provenance attestations
* GitHub Actions integration
* policy engine
* verification tooling

## Phase 3

* federation
* transparency logs
* reproducible build verification
* ecosystem adapters (PyPI/npm/etc.)

---

# License

Licensed under the Apache License 2.0.

---

# Vision

Verity aims to become open trust infrastructure for software distribution:

* open standards
* open governance
* transparent provenance
* verifiable releases
* secure software supply chains

Software should be verifiable by default.
