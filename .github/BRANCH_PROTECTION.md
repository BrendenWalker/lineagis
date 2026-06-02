# Branch protection — required status checks

CI runs on every pull request, on pushes to `main`, and on merge queue entries.

## Required checks

In **Settings → Branches → Branch protection rules → `main`**, enable **Require status checks to pass before merging** and select:

| Status check | Job |
|--------------|-----|
| `lint` | golangci-lint (`ci.yml`) |
| `test` | `make test-lineage` (`ci.yml`) |
| `build` | `make build` (`ci.yml`) |
| `smoke-lineage` | ingest → trace → why smoke (`ci.yml`) |

Also enable **Require branches to be up to date before merging**.

Checks appear in the picker after at least one workflow run on a pull request.
