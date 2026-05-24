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
* Docker Engine and Compose v2 (for the local dev stack)

```bash
choco install golangci-lint
```

### Windows notes

Go installs to `C:\Program Files\Go\bin`. If `go` is not found after install, **restart your terminal** (or Cursor) so it picks up the updated PATH.

**Git Bash** — if `go` still is not found, add this to `~/.bashrc`:

```bash
export PATH="$PATH:/c/Program Files/Go/bin"
```

**GNU Make** — use MinGW's `mingw32-make` (not Embarcadero `make`). Ensure its `bin` directory is on PATH *before* Delphi/Embarcadero, then verify:

```powershell
mingw32-make --version   # should say "GNU Make"
```

If `make` still resolves to Embarcadero, call `mingw32-make` explicitly:

```powershell
mingw32-make test
mingw32-make build
```

**PowerShell 5.1** does not support `&&` to chain commands. Use separate lines, or `;`:

```powershell
mingw32-make test; mingw32-make build
```

On Windows, run tests without the race detector (requires CGO). CI on Linux passes `TEST_FLAGS=-race`.

Run the CLI on Windows as `.\bin\verity.exe --version` (Git Bash: `./bin/verity.exe --version`).

## Build and test

```bash
make build    # produces bin/verity and bin/verity-api
make test     # race detector + coverage.out
make lint     # golangci-lint
```

Run the placeholder CLI:

```bash
./bin/verity --version
```

CI runs on every pull request and on pushes to `main` as three status checks: `lint`, `test`, and `build`. Coverage is uploaded as a workflow artifact when tests run. To block merges until CI passes, require those checks on `main` — see [.github/BRANCH_PROTECTION.md](.github/BRANCH_PROTECTION.md).

## Local development stack

Start Postgres, MinIO (S3-compatible storage), an OCI Distribution registry, and the Verity API:

```bash
cp .env.example .env   # optional; defaults match .env.example
make compose-up
```

| Service | URL | Purpose |
|---------|-----|---------|
| Verity API | http://localhost:8080 (or `$VERITY_API_PORT`) | API (`GET /healthz`, `GET /readyz`) |
| OCI Registry | http://localhost:5000 | Distribution-compatible registry (S3 backend) |
| PostgreSQL | localhost:5432 | Metadata database |
| MinIO API | http://localhost:9000 | S3-compatible object storage |
| MinIO Console | http://localhost:9001 | MinIO web UI |

Environment variables (see [`.env.example`](.env.example)):

| Variable | Default | Used by |
|----------|---------|---------|
| `POSTGRES_USER` | `verity` | PostgreSQL |
| `POSTGRES_PASSWORD` | `verity` | PostgreSQL |
| `POSTGRES_DB` | `verity` | PostgreSQL |
| `MINIO_ROOT_USER` | `minioadmin` | MinIO, registry S3 backend |
| `MINIO_ROOT_PASSWORD` | `minioadmin` | MinIO, registry S3 backend |
| `MINIO_REGISTRY_BUCKET` | `registry` | MinIO init (registry blob bucket) |
| `VERITY_API_ADDR` | `:8080` | Verity API listen address |
| `VERITY_API_PORT` | `8080` | Host port mapped to the API container |
| `VERITY_DATABASE_URL` | `postgres://verity:verity@localhost:5432/verity?sslmode=disable` | Verity API (local); compose sets internal URL |
| `VERITY_REGISTRY_URL` | `http://localhost:5000` | Verity API registry connectivity check |
| `VERITY_LOG_LEVEL` | `info` | Verity API structured logging |
| `VERITY_LOG_FORMAT` | `text` (local), `json` (compose) | Verity API log format |
| `VERITY_MIGRATE_ON_STARTUP` | `true` | Run goose migrations on API startup |

If port 8080 is already in use, set `VERITY_API_PORT=18080` in `.env` before `make compose-up`.

Verify health endpoints:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:5000/v2/
```

Run the full operator-stack smoke test (AC-ARCH-001). Requires Git Bash on Windows (`bash` on PATH):

```bash
make build          # optional; verifies CLI hop
make smoke          # compose-up + scripts/smoke-stack.sh
```

Registry push/pull only (requires [crane](https://github.com/google/go-containerregistry/tree/main/cmd/crane)):

```bash
go install github.com/google/go-containerregistry/cmd/crane@latest
make smoke-registry
```

Stop the stack:

```bash
make compose-down
```

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
