# Branch protection — required status checks

CI runs on every pull request, on pushes to `main`, and on [merge queue](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/configuring-pull-request-merges/managing-a-merge-queue) entries. Each job reports a separate GitHub status check you can require before merging to `main`.

## Required checks

In **Settings → Branches → Branch protection rules → `main`**, enable **Require status checks to pass before merging** and select:

| Status check | Job |
|--------------|-----|
| `lint` | golangci-lint |
| `test` | `go test` with race detector |
| `build` | `make build` |

Also enable **Require branches to be up to date before merging** so the latest commit is always validated.

Checks appear in the picker after at least one workflow run on a pull request (or after this workflow has run on `main`).

## Not in required CI

- `make smoke` / `make smoke-registry` — need Docker and a full stack; run locally or in release/integration pipelines, not on every PR.
