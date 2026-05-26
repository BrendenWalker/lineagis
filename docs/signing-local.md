# Local signing for `verity publish`

`verity publish` signs the OCI artifact manifest digest with Sigstore keyless signing by default (FR-SIGN-004), then calls `AttachSignature` on the Verity API.

## Without Fulcio (typical local stack)

The default `docker compose` setup does not run Fulcio or Rekor. Use one of:

```bash
verity publish ./dist --namespace gh/you/repo --artifact myapp --skip-sign
```

or:

```bash
export VERITY_SKIP_SIGN=1
verity publish ./dist --namespace gh/you/repo --artifact myapp
```

Push-time `require-signature` policy (when enabled on the namespace) will reject unsigned digests on `SetTag`; use `--skip-sign` only on namespaces that allow unsigned artifacts.

## Keyless signing in CI (GitHub Actions)

1. Grant `id-token: write` to the workflow job.
2. Ensure the OIDC token audience includes `sigstore` (Verity docs / your Fulcio configuration).
3. Run `verity publish` without `--skip-sign`. Cosign uses ambient GitHub OIDC when `SIGSTORE_ID_TOKEN` is unset.

Alternatively set `SIGSTORE_ID_TOKEN` to a pre-minted identity token.

## Environment variables

Verity-prefixed variables take precedence over cosign-standard `SIGSTORE_*` names. Endpoints default to **Sigstore public-good** (`fulcio.sigstore.dev`, `rekor.sigstore.dev`) when unset (NFR-SIGN-001, OQ-SIGN-001).

| Variable | Purpose |
|----------|---------|
| `VERITY_SKIP_SIGN` | `1` / `true` / `yes` — skip signing (local dev) |
| `VERITY_SIGSTORE_ID_TOKEN` | OIDC identity token for Fulcio (falls back to `SIGSTORE_ID_TOKEN`) |
| `VERITY_SIGSTORE_FULCIO_URL` | Fulcio endpoint (falls back to `SIGSTORE_FULCIO_URL`, then public-good) |
| `VERITY_SIGSTORE_REKOR_URL` | Rekor endpoint (falls back to `SIGSTORE_REKOR_URL`, then public-good) |

### Trust roots (verification / self-hosted)

Use these when verifying keyless bundles against non–public-good Fulcio/Rekor or pinned trust material:

| Variable | Purpose |
|----------|---------|
| `VERITY_SIGSTORE_TRUSTED_ROOT` | Path to Sigstore **trusted root** JSON (`application/vnd.dev.sigstore.trustedroot+json`) for v0.3 bundles |
| `VERITY_SIGSTORE_CA_ROOTS` | PEM path for Fulcio root CAs (legacy bundles) |
| `VERITY_SIGSTORE_CA_INTERMEDIATES` | PEM path for Fulcio intermediate CAs |
| `VERITY_SIGSTORE_ROOT_FILE` | Overrides Fulcio root CA (exported to `SIGSTORE_ROOT_FILE` for cosign) |
| `VERITY_SIGSTORE_REKOR_PUBLIC_KEY` | Rekor out-of-band public key PEM |
| `VERITY_SIGSTORE_CT_LOG_PUBLIC_KEY_FILE` | CT log public key for Fulcio SCT validation |

Each `VERITY_SIGSTORE_*` trust variable falls back to the matching `SIGSTORE_*` name if set. When trust paths are unset, cosign uses its default TUF public-good roots.

Signing failures abort publish (NFR-SIGN-002) unless signing is skipped.
