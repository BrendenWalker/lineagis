# Quickstart (local development)

> **Production maintainers:** use [GitHub Actions keyless publish](github-actions-publish.md) instead. This guide is for contributors running the stack locally with a dev token.

This walkthrough covers the Phase 1 **Must** path: publish an artifact and inspect trust output using `VERITY_DEV_TOKEN`.

Related acceptance criteria:

- `AC-DX-001` (quickstart publish path)
- `AC-OV-001` (publish stores digest + tag)
- `AC-OV-002` (inspect reports signature status)

## Prerequisites

- Go 1.23+
- Docker + Compose v2
- `bash` available on PATH (Git Bash on Windows)

## 1) Build binaries

```bash
make build
```

## 2) Start local stack

```bash
cp .env.example .env
make compose-up
```

## 3) Prepare artifact payload

```bash
mkdir -p dist
echo "quickstart $(date -u +%Y-%m-%dT%H:%M:%SZ)" > dist/release.txt
```

## 4) Publish (dev-only flags)

```bash
export VERITY_API_URL=http://localhost:8080
export VERITY_REGISTRY_URL=http://localhost:5000
export VERITY_TOKEN=dev-local-token
./bin/verity publish dist/ \
  --namespace gh/acme/quickstart \
  --artifact quickstart \
  --tag v0.1.0 \
  --skip-sign \
  --skip-provenance
```

| Flag / setting | Purpose |
|----------------|---------|
| `VERITY_DEV_TOKEN` / `VERITY_TOKEN` | Local API bearer — **not for production** |
| `--skip-sign` | Skip Fulcio when signing offline (dev only) |
| `--skip-provenance` | Skip SLSA provenance when not in CI |

**Do not use `--skip-sign` with `--tag` on a namespace that has `require-signatures` policy** — tagging unsigned digests is rejected.

Expected result:

- command exits `0`
- stdout includes a digest line like `sha256:...`

## 5) Inspect

Use the digest printed by `publish`:

```bash
./bin/verity inspect sha256:<digest> --namespace gh/acme/quickstart --artifact quickstart
```

Expected result:

- header: `Trust verified by Verity API (server-side Sigstore checks)`
- signature line reflects API trust status (unsigned artifacts show `✗` when policy requires signatures)
- Should lines may show `⚠` for missing provenance/SBOM

## 6) Tear down

```bash
make compose-down
```

## CI coverage

The required GitHub workflow `publish-keyless-smoke` runs keyless publish → inspect → `require-signatures` enforcement on every pull request to `main`. See [phase1-must-test-mapping.md](../sdlc/phase1-must-test-mapping.md).
