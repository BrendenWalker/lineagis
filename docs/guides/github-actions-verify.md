# Verify in GitHub Actions

Downstream consumers and release pipelines should verify **pinned digests** before promoting or deploying artifacts.

## Reusable action

```yaml
- name: Verify release digest
  uses: ./.github/actions/lineagis-verify
  with:
    digest: sha256:abcdef0123456789...
    namespace: gh/${{ github.repository_owner }}/my-app
    artifact: my-app
    lineagis-api-url: https://lineagis.example.com
    lineagis-registry-url: https://registry.example.com
    lineagis-token: ${{ secrets.LINEAGIS_TOKEN }}
```

The action runs `lineagis verify` (digest-required inspect) with local Sigstore verification by default. Set `local-verify: false` to trust API crypto only.

## Manual step

```yaml
      - name: Install Lineagis CLI
        run: go install ./cmd/lineagis

      - name: Verify pinned digest
        env:
          LINEAGIS_API_URL: https://lineagis.example.com
          LINEAGIS_REGISTRY_URL: https://registry.example.com
          LINEAGIS_TOKEN: ${{ secrets.LINEAGIS_TOKEN }}
        run: |
          lineagis verify sha256:abcdef0123456789... \
            --namespace gh/acme/widget \
            --artifact widget \
            --output json
```

## Best practices

- Always pin `sha256:…` in lockfiles and deploy configs; do not rely on mutable semver tags alone.
- Run verify in the same job (or a gated downstream job) before deployment.
- Configure namespace policy (`trusted-publishers`, `require-provenance`) on the Lineagis operator side — see [docs/examples/policies/](../examples/policies/).

See also [github-actions-publish.md](github-actions-publish.md) and [SECURITY.md](../../SECURITY.md).
