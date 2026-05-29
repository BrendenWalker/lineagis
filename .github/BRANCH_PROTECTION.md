# Branch protection — required status checks

CI runs on every pull request, on pushes to `main`, and on [merge queue](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue) entries. Each job reports a separate GitHub status check you can require before merging to `main`.

## Required checks

In **Settings → Branches → Branch protection rules → `main`**, enable **Require status checks to pass before merging** and select:

| Status check | Job |
|--------------|-----|
| `lint` | golangci-lint (`ci.yml`) |
| `test` | `go test` with race detector (`ci.yml`) |
| `build` | `make build` (`ci.yml`) |
| `keyless-publish` | publish → inspect → `require-signatures` (`publish-keyless-smoke.yml`) |

Also enable **Require branches to be up to date before merging** so the latest commit is always validated.

Checks appear in the picker after at least one workflow run on a pull request (or after this workflow has run on `main`).

`keyless-publish` satisfies **AC-OV-004** (Phase 1 acceptance). See [docs/sdlc/phase1-must-test-mapping.md](../docs/sdlc/phase1-must-test-mapping.md).

## Not in required CI

- `make smoke` / `make smoke-registry` — contributor convenience; builds the API from the Dockerfile. The **`operator-stack`** job in `ci.yml` is the automated AC-ARCH-001 check and uses workflow-built binaries only (see [operator-validation.md](../docs/guides/operator-validation.md)).
