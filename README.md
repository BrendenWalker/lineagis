# Lineagis

> Lineagis is an open-source trust platform for publishing, verifying, and governing software artifacts with built-in supply chain security.

---

## Overview

Modern software supply chains are fragmented, opaque, and increasingly vulnerable to compromise.

Lineagis aims to provide a secure, open, and verifiable foundation for software artifact distribution using OCI-native infrastructure, cryptographic signing, provenance attestations, and policy enforcement.

Rather than acting as “another package repository,” Lineagis is designed as a trust layer for software releases.

## Goals

* Make software releases verifiable by default
* Help open-source maintainers publish trusted artifacts
* Improve visibility into software provenance and integrity
* Provide portable supply-chain security primitives
* Build on open standards and open infrastructure

---

# Core Concepts

## OCI-Native Distribution

Lineagis uses OCI registries as the underlying distribution and storage layer for artifacts.

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

Lineagis is built around software trust primitives:

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
| **CLI** | `lineagis publish`, `lineagis inspect` (text + JSON), non-zero exit on Must / configured-policy failures |
| **Policy** | `require-signatures` at tag time and inspect |
| **Policy** | `trusted-publishers` **fail-closed when you add the rule** (operator-defined allowlist; verify-time in v0.1) |
| **Honesty** | Inspect does not show `✓` for checks that were not evaluated |

## Optional in v0.1 (Should)

* SLSA-style provenance and SBOM attachment
* Repository ownership policy (fail-closed when rule configured)
* Push-time enforcement for all configured policies on `SetTag` (v0.2 — today `trusted-publishers` may not block tag)
* [GitHub Actions publish guide](docs/guides/github-actions-publish.md) and [composite action](.github/actions/lineagis-publish/action.yml) — **production golden path**

## Trusted publishers (operator-defined)

Not a global safe-project list. Per namespace, operators configure which **signing identities** (e.g. GitHub `repository` + `workflow` from the Sigstore certificate) may satisfy policy when the `trusted-publishers` rule is enabled. See [docs/specs/04-policy-enforcement.md](docs/specs/04-policy-enforcement.md#trusted-publishers).

## v0.3 (Layer C — Governance)

| Area | Capabilities |
|------|----------------|
| **Consumer CLI** | `lineagis login`, `lineagis pull`, digest-pin warnings |
| **Policy** | Optional `require-digest-on-verify`, GitHub API `verify_with_github_api` on repository-ownership |
| **Integrations** | Namespace webhooks (`tag.set`, `policy.updated`, `verify.*`) |

See [consumer getting started](docs/guides/consumer-getting-started.md) and [mvp-v0.3-release.md](docs/sdlc/mvp-v0.3-release.md).

## Not in MVP (Deferred)

* CVE / vulnerability blocking
* Federation and transparency-log UX

## What `lineagis inspect` proves (and does not)

**Proves:** cryptographic signature validity (local cosign verify by default, plus API policy), tamper evidence for the registered digest, and active namespace policy results.

**Does not prove:** that the artifact is safe, malware-free, or free of vulnerable dependencies. A validly signed malicious artifact is still malicious. Compromised CI can produce valid signatures and provenance.

Pin releases by digest (`sha256:…`), not mutable tags alone.

---

# Example Workflow

## Publish (GitHub Actions — recommended)

Use keyless signing from CI. See [docs/guides/github-actions-publish.md](docs/guides/github-actions-publish.md).

```bash
lineagis publish dist/* --namespace gh/org/app --artifact app --tag v1.0.0
```

In GitHub Actions (with `id-token: write`), publish typically:

* signs the artifact digest with Sigstore
* attaches SLSA provenance (when not skipped)
* registers metadata and sets the semver tag

## Publish (local development only)

Local stack uses `LINEAGIS_DEV_TOKEN` and often `--skip-sign` / `--skip-provenance` when Fulcio is unavailable. **Do not use dev tokens or skip flags in production.** See [docs/guides/quickstart.md](docs/guides/quickstart.md).

---

## Verify and consume (v0.3)

```bash
lineagis login
lineagis pull gh/org/app/app@sha256:<digest> -o ./out --verify
lineagis inspect sha256:<digest> --namespace gh/org/app --artifact app
```

Consumer guide: [docs/guides/consumer-getting-started.md](docs/guides/consumer-getting-started.md).

## Verify

```bash
lineagis inspect sha256:<digest> --namespace gh/org/app --artifact app
```

Example output (Layer B / v0.2):

```text
Signature verified locally (Sigstore/Rekor)
✓ Signed by github.com/org/repo (release.yml) ref refs/heads/main
— Repository verified (repository-ownership not configured)
✓ Maintainer verified
✓ SBOM attached
✓ Provenance verified
✓ Published via workflow release (refs/heads/main) run 12345
```

Must checks and any **configured** policy rules must pass for exit code `0`. Unconfigured checks show `—` or `⚠`, not `✓`. Provenance and SBOM lines reflect cryptographic verification when attestations are present.

---

# Architecture

```text
                +-------------------+
                | Lineagis CLI        |
                +-------------------+
                         |
                         v
                +-------------------+
                | Lineagis API        |
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

[![CI](https://github.com/BrendenWalker/lineagis/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/BrendenWalker/lineagis/actions/workflows/ci.yml)

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

Run the CLI on Windows as `.\bin\lineagis.exe --version` (Git Bash: `./bin/lineagis.exe --version`).

## Build and test

```bash
make build    # produces bin/lineagis and bin/lineagis-api
make test     # race detector + coverage.out
make lint     # golangci-lint
```

Run the CLI:

```bash
./bin/lineagis --version
```

Publish a release directory (requires stack up and tokens from `.env.example`):

```bash
export LINEAGIS_API_URL=http://localhost:8080
export LINEAGIS_REGISTRY_URL=http://localhost:5000
export LINEAGIS_TOKEN=dev-local-token
./bin/lineagis publish dist/ --namespace gh/acme/widget --artifact widget --tag v1.0.0
```

On success the command prints the manifest digest (`sha256:…`, AC-PUB-001). Registry repository is `{namespace}/{artifact}` (e.g. `gh/acme/widget/widget`).

CI runs on every pull request and on pushes to `main`. Required status checks: `lint`, `test`, `build`, and `keyless-publish` (acceptance: publish → inspect → `require-signatures`). See [.github/BRANCH_PROTECTION.md](.github/BRANCH_PROTECTION.md).

## Local development stack

Start Postgres, MinIO (S3-compatible storage), a [Zot](https://zotregistry.dev/) OCI registry (artifact manifest support per ADR-0001), and the Lineagis API:

```bash
cp .env.example .env   # optional; defaults match .env.example
make compose-up
```

| Service | URL | Purpose |
|---------|-----|---------|
| Lineagis API | http://localhost:8080 (or `$LINEAGIS_API_PORT`) | API (`GET /healthz`, `GET /readyz`, `/v1/...` control plane) |
| OCI Registry | http://localhost:5000 | Zot registry (S3 backend via MinIO; OCI Artifact manifests) |
| PostgreSQL | localhost:5432 | Metadata database |
| MinIO API | http://localhost:9000 | S3-compatible object storage |
| MinIO Console | http://localhost:9001 | MinIO web UI |

Environment variables (see [`.env.example`](.env.example)):

| Variable | Default | Used by |
|----------|---------|---------|
| `POSTGRES_USER` | `lineagis` | PostgreSQL |
| `POSTGRES_PASSWORD` | `lineagis` | PostgreSQL |
| `POSTGRES_DB` | `lineagis` | PostgreSQL |
| `MINIO_ROOT_USER` | `minioadmin` | MinIO, registry S3 backend |
| `MINIO_ROOT_PASSWORD` | `minioadmin` | MinIO, registry S3 backend |
| `MINIO_REGISTRY_BUCKET` | `registry` | MinIO init (registry blob bucket) |
| `LINEAGIS_API_ADDR` | `:8080` | Lineagis API listen address |
| `LINEAGIS_API_PORT` | `8080` | Host port mapped to the API container |
| `LINEAGIS_DATABASE_URL` | `postgres://lineagis:lineagis@localhost:5432/lineagis?sslmode=disable` | Lineagis API (local); compose sets internal URL |
| `LINEAGIS_REGISTRY_URL` | `http://localhost:5000` | Lineagis API registry connectivity check |
| `LINEAGIS_LOG_LEVEL` | `info` | Lineagis API structured logging |
| `LINEAGIS_LOG_FORMAT` | `text` (local), `json` (compose) | Lineagis API log format |
| `LINEAGIS_MIGRATE_ON_STARTUP` | `true` | Run goose migrations on API startup |
| `LINEAGIS_DEV_TOKEN` | `dev-local-token` (compose) | Local dev bearer for API writes (OQ-API-002) |
| `LINEAGIS_OIDC_ISSUER` | (none) | GitHub Actions OIDC issuer (e.g. `https://token.actions.githubusercontent.com`) |
| `LINEAGIS_OIDC_AUDIENCE` | (none) | Expected JWT `aud` when OIDC is enabled |
| `LINEAGIS_API_URL` | `http://localhost:8080` | Lineagis CLI API base URL |
| `LINEAGIS_TOKEN` | (none) | Lineagis CLI bearer token (`LINEAGIS_DEV_TOKEN` fallback) |

If port 8080 is already in use, set `LINEAGIS_API_PORT=18080` in `.env` before `make compose-up`.

Verify health endpoints:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
curl http://localhost:5000/v2/
```

Run the full operator-stack smoke test (AC-ARCH-001). Requires Git Bash on Windows (`bash` on PATH):

**Contributors (from source):**

```bash
make build          # optional; verifies CLI hop
make smoke          # compose-up + scripts/smoke-stack.sh (API built via Dockerfile)
```

**Operators (workflow-built binaries — do not `go build` locally):**

Windows (Git Bash):

```bash
gh run download --name lineagis-binaries-windows-amd64 --dir bin
bash scripts/operator-stack-ci.sh
```

Linux / WSL:

```bash
gh run download --name lineagis-binaries-linux-amd64 --dir bin
chmod +x bin/lineagis bin/lineagis-api
bash scripts/operator-stack-ci.sh
```

See [docs/guides/operator-validation.md](docs/guides/operator-validation.md). CI runs the same path in the `operator-stack` job.

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

If you previously ran the stack with Docker Distribution (`registry:2`), reset registry storage before using Zot: `docker compose down -v` (clears MinIO/Postgres volumes) or delete objects under the MinIO `registry` bucket. Distribution and Zot use incompatible S3 key layouts. After reset, `make compose-up` and re-run integration tests with `LINEAGIS_TEST_REGISTRY_URL=http://localhost:5000`.

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

Lineagis aims to become open trust infrastructure for software distribution:

* open standards
* open governance
* transparent provenance
* verifiable releases
* secure software supply chains

Software should be verifiable by default.
