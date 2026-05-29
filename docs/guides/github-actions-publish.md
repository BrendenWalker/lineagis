# Publish from GitHub Actions (recommended)

This is the **primary onboarding path** for open-source maintainers. It uses Verity keyless signing and SLSA provenance (FR-DX-001, FR-DX-009).

For local development with `VERITY_DEV_TOKEN`, see [quickstart.md](quickstart.md) (dev-only).

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

## Inspect in CI (publishers)

After publish, verify trust before promoting a release:

```yaml
      - name: Verify release trust
        env:
          VERITY_API_URL: https://verity.example.com
          VERITY_TOKEN: ${{ secrets.VERITY_TOKEN }}
        run: |
          verity inspect "${{ steps.publish.outputs.digest }}" \
            --namespace "gh/${{ github.repository_owner }}/my-app" \
            --artifact my-app \
            --output json
```

Exit code `1` when any **Must** check fails (`require-signatures`, invalid signature) or any **configured** policy rule fails (including `trusted-publishers`). Unconfigured checks show `—` or `⚠`, not `✓`. When provenance and SBOM attestations are present, inspect reports `✓ Provenance verified` and `✓ SBOM attached` after Sigstore verification (AC-PROV-003; see `internal/api/attestations_test.go`).

## Trusted publishers (optional, recommended)

Configure a namespace policy so only your release workflow can satisfy signing policy. The allowlist is **your** choice per Verity instance — not a global list maintained by Verity.

```json
{
  "document": {
    "rules": [
      { "type": "require-signatures" },
      {
        "type": "trusted-publishers",
        "config": {
          "publishers": [
            {
              "repository": "your-org/your-repo",
              "workflow": "release.yml"
            }
          ]
        }
      }
    ]
  }
}
```

Apply via operator API (`PutPolicy`). Pin both `repository` and `workflow` when possible; `repository` alone allows any workflow in that repo.

When this rule is present, `verity inspect` **fails** if the artifact was signed by a different workflow identity. See [04-policy-enforcement.md](../specs/04-policy-enforcement.md#trusted-publishers).

Human output includes:

```text
Signature verified locally (Sigstore/Rekor)
✓ Signed by github.com/org/repo (release.yml)
— Repository verified (repository-ownership not configured)
✓ Provenance verified
```

See [phase2-should-test-mapping.md](../sdlc/phase2-should-test-mapping.md) for CI proof.

## Consuming releases (verify-only)

Downstream projects can pin a digest and verify before use:

```yaml
jobs:
  verify-artifact:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Verity CLI
        run: go install ./cmd/verity

      - name: Inspect pinned digest
        env:
          VERITY_API_URL: https://verity.example.com
          VERITY_TOKEN: ${{ secrets.VERITY_TOKEN }}
        run: |
          verity inspect sha256:abcdef0123456789... \
            --namespace gh/acme/widget \
            --artifact widget \
            --output json
```

**Best practices:**

- Pin `sha256:…` digests in documentation and lockfiles; do not rely on mutable tags alone.
- Treat `verity inspect` as **identity and integrity** checks, not malware scanning.
- Local Sigstore verification is the default in `verity inspect` / `verity verify`; use `--trust-api` only when registry access is unavailable.

See [SECURITY.md](../../SECURITY.md), [mvp-v0.2-release.md](../sdlc/mvp-v0.2-release.md), and [mvp-v0.1-release.md](../sdlc/mvp-v0.1-release.md).

For verify-only consumers, see [github-actions-verify.md](github-actions-verify.md).
