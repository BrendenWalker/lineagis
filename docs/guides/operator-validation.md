# Operator validation (local stack)

Use this when validating the **operator deployment path** on your machine: full Docker Compose infra + workflow-built binaries. Do **not** `go build` from source for operator checks.

Contributors building features locally should still use [quickstart.md](quickstart.md) (`make build` from source).

Related acceptance criteria:

- `AC-ARCH-001` — stack connectivity (API, registry, MinIO, Postgres, CLI)
- `AC-POL-002b` / `AC-OV-005` — covered in CI (`publish-keyless-smoke`); optional re-run locally via GHA

## Prerequisites

- Docker + Compose v2
- `bash` (Git Bash on Windows; WSL optional)
- [GitHub CLI](https://cli.github.com/) (`gh`) authenticated
- [crane](https://github.com/google/go-containerregistry/tree/main/cmd/crane) on PATH (registry push/pull smoke)

## 1) Download CI-built binaries

The CI **`build`** job publishes two artifacts:

| Artifact | Platform | Use on |
|----------|----------|--------|
| `lineagis-binaries-linux-amd64` | Linux amd64 | Linux, WSL, CI |
| `lineagis-binaries-windows-amd64` | Windows amd64 | Windows (native) |

### Windows (Git Bash or PowerShell)

**Git Bash** (recommended — same scripts as CI):

```bash
gh run download --repo "$(gh repo view --json nameWithOwner -q .nameWithOwner)" \
  --name lineagis-binaries-windows-amd64 \
  --dir bin
```

**PowerShell:**

```powershell
gh run download --repo (gh repo view --json nameWithOwner -q .nameWithOwner) `
  --name lineagis-binaries-windows-amd64 --dir bin
```

Expected files: `bin/lineagis.exe`, `bin/lineagis-api.exe`.

### Linux / WSL

```bash
gh run download --repo "$(gh repo view --json nameWithOwner -q .nameWithOwner)" \
  --name lineagis-binaries-linux-amd64 \
  --dir bin
chmod +x bin/lineagis bin/lineagis-api
```

Pin a specific run when validating a release candidate:

```bash
gh run download --repo OWNER/REPO --run RUN_ID \
  --name lineagis-binaries-windows-amd64 --dir bin
```

## 2) Start stack and run smoke

From the repo root in **Git Bash**:

```bash
cp .env.example .env   # if needed
bash scripts/operator-stack-ci.sh
```

This starts Postgres, MinIO, and Zot from Compose, runs the workflow-built **`lineagis-api`** binary on your host (not the Dockerfile API image), then `scripts/smoke-stack.sh`.

**Success:** `=== all smoke checks passed ===`

## 3) Optional — re-run acceptance workflow locally

Policy and keyless signing tests run in CI only (OIDC + Fulcio). To confirm on your fork:

```bash
gh workflow run publish-keyless-smoke.yml
gh run watch
```

That workflow covers:

- `lineagis verify sha256:…` on a keyless-signed digest
- `docs/examples/policies/strict-release.json` with wrong-workflow push-time block
- `require-signatures` unsigned tag rejection

## Tear down

```bash
docker compose down
```

If `operator-stack-ci.sh` was interrupted, also run `docker compose down -v --remove-orphans`.

## CI coverage

| Check | Workflow / job |
|-------|----------------|
| `operator-stack` | `ci.yml` — full compose infra + Linux artifact binaries |
| `keyless-publish` | `publish-keyless-smoke.yml` — verify, strict-release, require-signatures |

See [mvp-v0.1-release.md](../sdlc/mvp-v0.1-release.md) and [phase1-must-test-mapping.md](../sdlc/phase1-must-test-mapping.md).
