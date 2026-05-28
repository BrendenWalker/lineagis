# Quickstart (Phase 1 Must)

This guide walks a maintainer through the MVP Must path: publish an artifact and inspect trust output.

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

## 4) Publish

```bash
export VERITY_API_URL=http://localhost:8080
export VERITY_REGISTRY_URL=http://localhost:5000
export VERITY_TOKEN=dev-local-token
./bin/verity publish dist/ \
  --namespace gh/acme/quickstart \
  --artifact quickstart \
  --tag v0.1.0 \
  --skip-provenance
```

Local publish uses `VERITY_DEV_TOKEN`; use `--skip-provenance` when Fulcio is unavailable. In GitHub Actions, omit that flag to attach SLSA provenance automatically.

Expected result:

- command exits `0`
- stdout includes a digest line like `sha256:...`

## 5) Inspect

Use the digest printed by `publish`:

```bash
./bin/verity inspect sha256:<digest> --namespace gh/acme/quickstart --artifact quickstart
```

Expected result:

- output includes `Signed by GitHub Actions` (Must) and optional Should lines (provenance, SBOM)
- command exits `0` for signed artifacts

## 6) Tear down

```bash
make compose-down
```

## CI coverage

The GitHub workflow `publish-keyless-smoke` runs the same publish -> inspect path in automation, including a failure case for unsigned artifacts under `require-signatures` policy.
