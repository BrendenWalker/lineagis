# Publish from GitHub Actions

This guide shows how to publish and sign artifacts from GitHub Actions using Verity keyless signing and SLSA provenance (FR-DX-001, FR-DX-009).

## Prerequisites

- Verity API reachable from the workflow runner
- Namespace `gh/<owner>/<repo>` (lowercase owner) registered on first publish
- API accepts GitHub OIDC tokens **or** a maintainer token for development

## Workflow permissions

Grant OIDC to the job so Sigstore and the Verity API can authenticate the workflow:

```yaml
permissions:
  contents: read
  id-token: write
```

## Environment variables

| Variable | Description |
|----------|-------------|
| `VERITY_API_URL` | Verity API base URL |
| `VERITY_TOKEN` | Bearer token (dev) **or** omit when API OIDC is configured |
| `VERITY_REGISTRY_URL` | OCI registry URL |
| `VERITY_OIDC_ISSUER` | API OIDC issuer (when using GitHub token to API) |
| `VERITY_OIDC_AUDIENCE` | API OIDC audience |

Sigstore uses ambient `ACTIONS_ID_TOKEN_REQUEST_TOKEN` automatically in GitHub Actions.

## Example job

```yaml
jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      id-token: write
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod

      - name: Build artifacts
        run: mkdir -p dist && go build -o dist/app ./cmd/verity

      - name: Publish with Verity
        env:
          VERITY_API_URL: https://verity.example.com
          VERITY_REGISTRY_URL: https://registry.example.com
        run: |
          go run ./cmd/verity publish dist/ \
            --namespace "gh/${{ github.repository_owner }}/my-app" \
            --artifact my-app \
            --tag "${{ github.ref_name }}"
```

Provenance is generated automatically in GitHub Actions. To attach an SBOM:

```bash
verity publish dist/ --namespace gh/org/app --artifact app --sbom sbom.json
```

## Reusable action

See [.github/actions/verity-publish/action.yml](../../.github/actions/verity-publish/action.yml) for a composite action wrapper.

## Inspect in CI

```bash
verity inspect dist/ --namespace gh/org/app --artifact app --output json
```

Exit code `1` when any **Must** check fails (`require-signatures`, invalid signature). **Should** lines (provenance, SBOM, repository) are informational unless policy rules fail.
