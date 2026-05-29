# Verify in GitHub Actions

Downstream consumers and release pipelines should verify **pinned digests** before promoting or deploying artifacts.

## Reusable action

```yaml
- name: Verify release digest
  uses: ./.github/actions/verity-verify
  with:
    digest: sha256:abcdef0123456789...
    namespace: gh/${{ github.repository_owner }}/my-app
    artifact: my-app
    verity-api-url: https://verity.example.com
    verity-registry-url: https://registry.example.com
    verity-token: ${{ secrets.VERITY_TOKEN }}
```

The action runs `verity verify` (digest-required inspect) with local Sigstore verification by default. Set `local-verify: false` to trust API crypto only.

## Manual step

```yaml
      - name: Install Verity CLI
        run: go install ./cmd/verity

      - name: Verify pinned digest
        env:
          VERITY_API_URL: https://verity.example.com
          VERITY_REGISTRY_URL: https://registry.example.com
          VERITY_TOKEN: ${{ secrets.VERITY_TOKEN }}
        run: |
          verity verify sha256:abcdef0123456789... \
            --namespace gh/acme/widget \
            --artifact widget \
            --output json
```

## Best practices

- Always pin `sha256:…` in lockfiles and deploy configs; do not rely on mutable semver tags alone.
- Run verify in the same job (or a gated downstream job) before deployment.
- Configure namespace policy (`trusted-publishers`, `require-provenance`) on the Verity operator side — see [docs/examples/policies/](../examples/policies/).

See also [github-actions-publish.md](github-actions-publish.md) and [SECURITY.md](../../SECURITY.md).
