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

# MVP Scope (v0.1)

The initial release delivers **Layer A — Integrity** (see [docs/specs/00-overview.md](docs/specs/00-overview.md)). Authoritative phasing uses Must / Should / Deferred.

## Guaranteed in v0.1 (Must)

| Area | Capabilities |
|------|----------------|
| **Publishing** | OCI artifact push, immutable `sha256:` digests, semver tags |
| **Signing** | Sigstore keyless signing (GitHub Actions), server-side signature verification on trust status |
| **CLI** | `verity publish`, `verity inspect` (text + JSON), non-zero exit on Must / configured-policy failures |
| **Policy** | `require-signatures` at tag time and inspect |
| **Policy** | `trusted-publishers` **fail-closed when you add the rule** (operator-defined allowlist; verify-time in v0.1) |
| **Honesty** | Inspect does not show `✓` for checks that were not evaluated |

## Optional in v0.1 (Should)

* SLSA-style provenance and SBOM attachment
* Repository ownership policy (fail-closed when rule configured)
* Push-time enforcement for all configured policies on `SetTag` (v0.2 — today `trusted-publishers` may not block tag)
* [GitHub Actions publish guide](docs/guides/github-actions-publish.md) and [composite action](.github/actions/verity-publish/action.yml) — **production golden path**

## Trusted publishers (operator-defined)

Not a global safe-project list. Per namespace, operators configure which **signing identities** (e.g. GitHub `repository` + `workflow` from the Sigstore certificate) may satisfy policy when the `trusted-publishers` rule is enabled. See [docs/specs/04-policy-enforcement.md](docs/specs/04-policy-enforcement.md#trusted-publishers).

## Not in v0.1 (Deferred)

* CVE / vulnerability blocking
* Federation and transparency-log UX
* `verity pull`, CLI OIDC token exchange, offline cosign verify on inspect

## What `verity inspect` proves (and does not)

**Proves:** cryptographic signature validity (via the Verity API), tamper evidence for the registered digest, and active namespace policy results.

**Does not prove:** that the artifact is safe, malware-free, or free of vulnerable dependencies. A validly signed malicious artifact is still malicious. Compromised CI can produce valid signatures and provenance.

Pin releases by digest (`sha256:…`), not mutable tags alone.

---

# Example Workflow

## Publish (GitHub Actions — recommended)

Use keyless signing from CI. See [docs/guides/github-actions-publish.md](docs/guides/github-actions-publish.md).

```bash
verity publish dist/* --namespace gh/org/app --artifact app --tag v1.0.0
```

In GitHub Actions (with `id-token: write`), publish typically:

* signs the artifact digest with Sigstore
* attaches SLSA provenance (when not skipped)
* registers metadata and sets the semver tag

## Publish (local development only)

Local stack uses `VERITY_DEV_TOKEN` and often `--skip-sign` / `--skip-provenance` when Fulcio is unavailable. **Do not use dev tokens or skip flags in production.** See [docs/guides/quickstart.md](docs/guides/quickstart.md).

---

## Verify

```bash
verity inspect sha256:<digest> --namespace gh/org/app --artifact app
```

Example output (v0.1):

```text
Trust verified by Verity API (server-side Sigstore checks)
✓ Signed by GitHub Actions
⚠ Repository not verified (no provenance repository)
⚠ Maintainer not verified (signature missing or invalid)
⚠ SBOM not attached
⚠ Provenance not attached
```

Must checks and any **configured** policy rules must pass for exit code `0`. Unconfigured checks show `—` or `⚠`, not `✓`. Attestation lines are informational until provenance verify ships (Layer B).

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

Run the CLI:

```bash
./bin/verity --version
```

Publish a release directory (requires stack up and tokens from `.env.example`):

```bash
export VERITY_API_URL=http://localhost:8080
export VERITY_REGISTRY_URL=http://localhost:5000
export VERITY_TOKEN=dev-local-token
./bin/verity publish dist/ --namespace gh/acme/widget --artifact widget --tag v1.0.0
```

On success the command prints the manifest digest (`sha256:…`, AC-PUB-001). Registry repository is `{namespace}/{artifact}` (e.g. `gh/acme/widget/widget`).

CI runs on every pull request and on pushes to `main`. Required status checks: `lint`, `test`, `build`, and `keyless-publish` (acceptance: publish → inspect → `require-signatures`). See [.github/BRANCH_PROTECTION.md](.github/BRANCH_PROTECTION.md).

## Local development stack

Start Postgres, MinIO (S3-compatible storage), a [Zot](https://zotregistry.dev/) OCI registry (artifact manifest support per ADR-0001), and the Verity API:

```bash
cp .env.example .env   # optional; defaults match .env.example
make compose-up
```

| Service | URL | Purpose |
|---------|-----|---------|
| Verity API | http://localhost:8080 (or `$VERITY_API_PORT`) | API (`GET /healthz`, `GET /readyz`, `/v1/...` control plane) |
| OCI Registry | http://localhost:5000 | Zot registry (S3 backend via MinIO; OCI Artifact manifests) |
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
| `VERITY_DEV_TOKEN` | `dev-local-token` (compose) | Local dev bearer for API writes (OQ-API-002) |
| `VERITY_OIDC_ISSUER` | (none) | GitHub Actions OIDC issuer (e.g. `https://token.actions.githubusercontent.com`) |
| `VERITY_OIDC_AUDIENCE` | (none) | Expected JWT `aud` when OIDC is enabled |
| `VERITY_API_URL` | `http://localhost:8080` | Verity CLI API base URL |
| `VERITY_TOKEN` | (none) | Verity CLI bearer token (`VERITY_DEV_TOKEN` fallback) |

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

### Registry migration (Distribution → Zot)

If you previously ran the stack with Docker Distribution (`registry:2`), reset registry storage before using Zot: `docker compose down -v` (clears MinIO/Postgres volumes) or delete objects under the MinIO `registry` bucket. Distribution and Zot use incompatible S3 key layouts. After reset, `make compose-up` and re-run integration tests with `VERITY_TEST_REGISTRY_URL=http://localhost:5000`.

---

# Documentation

Detailed MVP requirements live in [docs/specs/](docs/specs/README.md): an overview with delivery matrix (Must / Should / Deferred), foundation specs (architecture, API, metadata model), and feature specs for publishing, signing, provenance, policy, and developer experience.

For execution-focused docs:

- [GitHub Actions publish (recommended)](docs/guides/github-actions-publish.md)
- [Quickstart — local dev only](docs/guides/quickstart.md)
- [v0.1 release checklist](docs/sdlc/mvp-v0.1-release.md)
- [Phase 1 Must mapping to tests](docs/sdlc/phase1-must-test-mapping.md)
- [Security](SECURITY.md)

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
