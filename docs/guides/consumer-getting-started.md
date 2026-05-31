# Consumer getting started

This guide is for **consumers** who download and verify artifacts published through Lineagis. Maintainers should use [quickstart.md](quickstart.md) or [github-actions-publish.md](github-actions-publish.md).

## Prerequisites

- Lineagis CLI (`make build` or release binary)
- Read access to the Lineagis API and OCI registry
- A bearer token or GitHub Actions OIDC (see `lineagis login`)

## 1) Authenticate

```bash
export LINEAGIS_API_URL=https://lineagis.example.com
export LINEAGIS_REGISTRY_URL=https://registry.example.com

# Option A: pre-shared token
export LINEAGIS_TOKEN=your-reader-or-maintainer-token
lineagis login

# Option B: GitHub Actions (permissions.id-token: write)
# login is a no-op when ACTIONS_ID_TOKEN_* is set; token is fetched automatically
```

`lineagis login` writes `~/.lineagis/config` (mode `0600`) when a token is available from the environment or GitHub Actions.

## 2) Pull an artifact

Pin by digest when possible:

```bash
lineagis pull gh/acme/widget/app@sha256:abc123... -o ./release
```

Pull by tag (mutable; Lineagis warns on inspect):

```bash
lineagis pull gh/acme/widget/app:v1.0.0 -o ./release
```

Verify before writing files:

```bash
lineagis pull gh/acme/widget/app@sha256:abc123... -o ./release --verify
```

## 3) Verify trust

```bash
lineagis verify sha256:abc123... --namespace gh/acme/widget --artifact app
```

`lineagis verify` requires a digest reference (alias for `inspect --require-digest`).

In CI, use the [lineagis-verify action](../.github/actions/lineagis-verify/action.yml) and pass a pinned digest.

## 4) Operator policy template

Ask your operator to start from [strict-release.json](../examples/policies/strict-release.json) for production namespaces.

Optional rules in v0.3:

- `require-digest-on-verify` — reject tag-only verify in CI
- `repository-ownership` with `verify_with_github_api: true` — live GitHub repo check

## Related

- [SECURITY.md](../../SECURITY.md)
- [mvp-v0.3-release.md](../sdlc/mvp-v0.3-release.md)
