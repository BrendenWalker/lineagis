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

| Variable | Purpose |
|----------|---------|
| `VERITY_SKIP_SIGN` | `1` / `true` / `yes` — skip signing (local dev) |
| `SIGSTORE_ID_TOKEN` | OIDC identity token for Fulcio (aud must include `sigstore`) |
| `SIGSTORE_FULCIO_URL` | Fulcio endpoint (default: `https://fulcio.sigstore.dev`) |
| `SIGSTORE_REKOR_URL` | Rekor endpoint (default: `https://rekor.sigstore.dev`) |

Signing failures abort publish (NFR-SIGN-002) unless signing is skipped.
