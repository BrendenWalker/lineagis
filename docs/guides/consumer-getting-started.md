# Consumer getting started

This guide is for **consumers** who download and verify artifacts published through Verity. Maintainers should use [quickstart.md](quickstart.md) or [github-actions-publish.md](github-actions-publish.md).

## Prerequisites

- Verity CLI (`make build` or release binary)
- Read access to the Verity API and OCI registry
- A bearer token or GitHub Actions OIDC (see `verity login`)

## 1) Authenticate

```bash
export VERITY_API_URL=https://verity.example.com
export VERITY_REGISTRY_URL=https://registry.example.com

# Option A: pre-shared token
export VERITY_TOKEN=your-reader-or-maintainer-token
verity login

# Option B: GitHub Actions (permissions.id-token: write)
# login is a no-op when ACTIONS_ID_TOKEN_* is set; token is fetched automatically
```

`verity login` writes `~/.verity/config` (mode `0600`) when a token is available from the environment or GitHub Actions.

## 2) Pull an artifact

Pin by digest when possible:

```bash
verity pull gh/acme/widget/app@sha256:abc123... -o ./release
```

Pull by tag (mutable; Verity warns on inspect):

```bash
verity pull gh/acme/widget/app:v1.0.0 -o ./release
```

Verify before writing files:

```bash
verity pull gh/acme/widget/app@sha256:abc123... -o ./release --verify
```

## 3) Verify trust

```bash
verity verify sha256:abc123... --namespace gh/acme/widget --artifact app
```

`verity verify` requires a digest reference (alias for `inspect --require-digest`).

In CI, use the [verity-verify action](../.github/actions/verity-verify/action.yml) and pass a pinned digest.

## 4) Operator policy template

Ask your operator to start from [strict-release.json](../examples/policies/strict-release.json) for production namespaces.

Optional rules in v0.3:

- `require-digest-on-verify` — reject tag-only verify in CI
- `repository-ownership` with `verify_with_github_api: true` — live GitHub repo check

## Related

- [SECURITY.md](../../SECURITY.md)
- [mvp-v0.3-release.md](../sdlc/mvp-v0.3-release.md)
