# Developer Experience

## Summary

Developer experience defines the CLI-first workflow, GitHub Actions integration, and human-readable verification output (`verity publish`, `verity inspect`). It ties together publishing, signing, provenance, and policy into flows matching the README examples.

See [00-overview.md](00-overview.md#mvp-delivery-matrix).

## Goals

- Provide a minimal, documented CLI for publish and inspect.
- Integrate GitHub Actions as the primary maintainer path.
- Present clear, actionable verification output for consumers.

## Non-goals

- Web UI or graphical dashboard.
- IDE plugins (VS Code, JetBrains).
- Interactive TUI beyond basic CLI prompts.
- Package manager-specific plugins (pip, npm) in MVP.

## Personas

| Persona | Need |
|---------|------|
| **Maintainer** | One-command publish from CI. |
| **Consumer** | One-command inspect before installing an artifact. |
| **Operator** | Documented configuration for CLI and Actions. |

## User stories

| ID | Priority | Story |
|----|----------|-------|
| US-DX-001 | Must | As a maintainer, I want `verity publish dist/*` to upload, sign, and tag in one step. |
| US-DX-002 | Must | As a consumer, I want `verity inspect package.whl` to print a trust checklist. |
| US-DX-003 | Should | As a maintainer, I want a GitHub Action to run publish on release without custom scripts. |
| US-DX-004 | Must | As a user, I want configuration via environment variables and config file for API/registry endpoints. |
| US-DX-005 | Must | As a consumer, I want non-zero exit code when any configured policy or Must check fails so that CI can gate releases. |
| US-DX-006 | Must | As a maintainer, I want `--skip-sign` and dev tokens documented as local-only so that production releases always use keyless signing. |

## CLI commands (MVP)

| Command | Priority | Behavior |
|---------|----------|----------|
| `verity publish <path>` | Must | Push to OCI, sign (unless skipped), register metadata, set tag, run push policies. |
| `verity inspect <ref>` | Must | Resolve local file, digest, tag, or path; print trust checklist; optional JSON output. |
| `verity login` | Should | Obtain and cache OIDC/device token for API. |
| `verity pull <artifact:tag>` | Should | Pull artifact by tag or digest. |
| `verity policy` | Should | Operator subcommands to view/apply policy (minimal). |

### `verity publish` flags (informative)

| Flag | Priority | Description |
|------|----------|-------------|
| `--tag` | Must | Semver tag to apply (default from env or prompt). |
| `--namespace` | Must | Target namespace. |
| `--artifact` | Must | Logical artifact name. |
| `--skip-sign` | Must (dev-only) | Skip signing for local stack only; SHALL NOT be used in production release workflows; fails if policy requires signature when tagging. |
| `--skip-provenance` | Must (dev-only) | Skip provenance for local stack only; SHALL NOT be used in production release workflows. |
| `--sbom <file>` | Should | Attach SBOM file. |
| `--provenance <file>` | Should | Attach custom provenance (else generate from CI env). |

### `verity inspect` output lines

Map to README example and feature specs:

| Line | Priority | Source spec |
|------|----------|-------------|
| `✓ Signed by GitHub Actions` | Must | [02-signing-verification.md](02-signing-verification.md) |
| `✓ Repository verified` | Should | [03-provenance-metadata.md](03-provenance-metadata.md), [04-policy-enforcement.md](04-policy-enforcement.md) |
| `✓ Publisher allowed` | Must (when rule configured) | [04-policy-enforcement.md](04-policy-enforcement.md) — trusted-publishers |
| `✓ SBOM attached` | Should | [03-provenance-metadata.md](03-provenance-metadata.md) |
| `✓ Provenance verified` | Should | [03-provenance-metadata.md](03-provenance-metadata.md) |
| `✓ No critical vulnerabilities detected` | Deferred | [04-policy-enforcement.md](04-policy-enforcement.md) |

Failed checks use `✗` with reason. Configured policy failures are never downgraded to `⚠`. Unconfigured checks use `—` or `⚠` (missing optional attestation), never `✓`.

## GitHub Actions integration

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-DX-001 | Must | Documentation SHALL describe a GitHub Actions workflow using `id-token: write` for OIDC. |
| FR-DX-002 | Should | A reusable workflow or official action SHALL invoke `verity publish` with repository and ref context. |
| FR-DX-003 | Should | Required secrets and permissions SHALL be listed (registry URL, Verity API URL, namespace). |
| FR-DX-004 | Should | Example workflow SHALL tag releases on git tag push or `workflow_dispatch`. |

**Example workflow shape (informative):**

```yaml
permissions:
  id-token: write
  contents: read
steps:
  - uses: actions/checkout@v4
  - uses: verity-dev/verity-action@v0  # Should; name TBD
    with:
      namespace: gh/${{ github.repository }}
      artifact: my-package
      tag: ${{ github.ref_name }}
      path: dist/*
```

## Functional requirements

| ID | Priority | Requirement |
|----|----------|-------------|
| FR-DX-005 | Must | CLI SHALL exit non-zero on inspect when any Must check or any **configured** policy rule fails. |
| FR-DX-010 | Must | CLI help and guides SHALL state that `--skip-sign`, `--skip-provenance`, and `VERITY_DEV_TOKEN` are for local development only. |
| FR-DX-006 | Must | CLI SHALL support `--output json` for machine-readable trust reports. |
| FR-DX-007 | Must | Error messages SHALL reference failing requirement id when applicable (e.g. policy rule). |
| FR-DX-008 | Should | CLI version SHALL be reported in API client user-agent for support. |
| FR-DX-009 | Should | `verity publish` in GitHub Actions SHALL use `ACTIONS_ID_TOKEN_REQUEST_TOKEN` / OIDC for signing and API auth. |
| FR-DX-011 | Should | `verity login` SHALL obtain an API bearer token (env, GitHub Actions OIDC, or cached config) and persist it for subsequent CLI commands. |
| FR-DX-012 | Should | `verity pull` SHALL resolve a tag or digest via the Verity API, pull manifest and layers from the OCI registry, and write files to a local directory. |

## Non-functional requirements

| ID | Requirement |
|----|-------------|
| NFR-DX-001 | CLI binaries SHALL be distributed for Linux, macOS, and Windows (amd64 minimum). |
| NFR-DX-002 | Help text (`--help`) SHALL document required flags for publish and inspect. |

## Standards and references

- [README example workflow](../../README.md#example-workflow)
- [GitHub Actions OIDC](https://docs.github.com/en/actions/deployment/security-hardening-your-deployments/about-security-hardening-with-openid-connect)
- All feature specs 01–04

## Dependencies

- [01-artifact-publishing.md](01-artifact-publishing.md)
- [02-signing-verification.md](02-signing-verification.md)
- [03-provenance-metadata.md](03-provenance-metadata.md)
- [04-policy-enforcement.md](04-policy-enforcement.md)
- [api.md](api.md)

## Acceptance criteria

| ID | Criterion | Maps to |
|----|-----------|---------|
| AC-DX-001 | Given documented quickstart, when a maintainer follows steps on a clean repo, then `verity publish` completes with digest and tag. | US-DX-001, FR-DX-005 |
| AC-DX-002 | Given published signed artifact, when running `verity inspect` on the file, then Must lines appear per trust state. | US-DX-002 |
| AC-DX-003 | Given inspect with failed Must check, when exit code is checked, then it is non-zero. | FR-DX-005 |
| AC-DX-004 | Given sample GitHub workflow, when run on tag push, then release artifacts are published without manual keys. | FR-DX-001, FR-DX-009 |
| AC-DX-005 | Given `--output json`, when inspect runs, then output validates against documented JSON schema (schema TBD in implementation). | FR-DX-006 |
| AC-DX-PULL-001 | Given digest D and valid auth, when `verity pull ns/artifact@sha256:…`, then bytes match registry manifest for D. | FR-DX-012, US-PUB-003 |
| AC-DX-PULL-002 | Given `require-signatures` and unsigned digest, when pull with `--verify`, then non-zero exit. | FR-DX-012, FR-POL-005 |

## Open questions

| ID | Question |
|----|----------|
| OQ-DX-001 | Monolithic `verity` binary vs separate `verity` and `verity-server`? |
| OQ-DX-002 | Config file path default: `~/.config/verity/config.yaml`? |
