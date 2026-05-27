package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/cliconfig"
	"github.com/BrendenWalker/verity/internal/publish"
	"github.com/BrendenWalker/verity/internal/registry"
)

func runPublish(args []string) int {
	var path, namespace, artifact, tag, sbomPath string
	skipSign := publish.SkipSignFromEnv()
	skipProvenance := publish.SkipProvenanceFromEnv()
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--skip-sign":
			skipSign = true
		case "--skip-provenance":
			skipProvenance = true
		case "--sbom":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "publish: --sbom requires a value\n")
				return 1
			}
			i++
			sbomPath = args[i]
		case "--namespace":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "publish: --namespace requires a value\n")
				return 1
			}
			i++
			namespace = args[i]
		case "--artifact":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "publish: --artifact requires a value\n")
				return 1
			}
			i++
			artifact = args[i]
		case "--tag":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "publish: --tag requires a value\n")
				return 1
			}
			i++
			tag = args[i]
		case "-h", "--help":
			printPublishUsage()
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "publish: unknown flag %q\n", args[i])
				return 1
			}
			if path != "" {
				fmt.Fprintf(os.Stderr, "publish: unexpected argument %q\n", args[i])
				return 1
			}
			path = args[i]
		}
	}

	if path == "" {
		printPublishUsage()
		return 1
	}
	if namespace == "" || artifact == "" {
		fmt.Fprintf(os.Stderr, "publish: --namespace and --artifact are required\n")
		return 1
	}

	cfg, err := cliconfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "publish: %v\n", err)
		return 1
	}

	reg, err := registry.New(cfg.RegistryURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "publish: registry: %v\n", err)
		return 1
	}
	api := apiclient.New(cfg.APIURL, cfg.Token)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	digest, err := publish.Publish(ctx, reg, api, publish.Options{
		Namespace:      namespace,
		Artifact:       artifact,
		Tag:              tag,
		Path:             path,
		SBOMPath:         sbomPath,
		SkipSign:         skipSign,
		SkipProvenance:   skipProvenance,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "publish: %v\n", err)
		return 1
	}

	fmt.Println(digest)
	return 0
}

func printPublishUsage() {
	fmt.Fprintf(os.Stderr, "Usage: verity publish <path> --namespace <ns> --artifact <name> [--tag <tag>] [--sbom <file>] [--skip-sign] [--skip-provenance]\n")
	fmt.Fprintf(os.Stderr, "\nSigning uses Sigstore public-good (Fulcio/Rekor) by default. Local dev without Fulcio: --skip-sign or VERITY_SKIP_SIGN=1.\n")
	fmt.Fprintf(os.Stderr, "CI keyless: VERITY_SIGSTORE_ID_TOKEN / SIGSTORE_ID_TOKEN, or GitHub Actions ambient OIDC (id-token: write).\n")
	fmt.Fprintf(os.Stderr, "Operator trust roots: VERITY_SIGSTORE_* (see docs/signing-local.md).\n")
}
