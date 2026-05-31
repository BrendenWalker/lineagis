package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BrendenWalker/lineagis/internal/apiclient"
	"github.com/BrendenWalker/lineagis/internal/cliconfig"
	"github.com/BrendenWalker/lineagis/internal/inspect"
)

func runVerify(args []string) int {
	// verify is an alias for inspect with digest-first defaults (FR-DX-005).
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		printVerifyUsage()
		return 0
	}
	if !hasFlag(args, "--require-digest") {
		args = append([]string{"--require-digest"}, args...)
	}
	return runInspect(args)
}

func printVerifyUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis verify <sha256:digest> --namespace <ns> --artifact <name> [--local-verify] [--output json]\n")
	fmt.Fprintf(os.Stderr, "\nAlias for inspect with --require-digest enabled. Pin sha256:… digests in CI.\n")
}

func hasFlag(args []string, name string) bool {
	for _, a := range args {
		if a == name {
			return true
		}
	}
	return false
}

func runInspect(args []string) int {
	var namespace, artifact, ref, output string
	localVerify := true
	trustAPIOnly := false
	requireDigest := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--namespace":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "inspect: --namespace requires a value\n")
				return 1
			}
			i++
			namespace = args[i]
		case "--artifact":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "inspect: --artifact requires a value\n")
				return 1
			}
			i++
			artifact = args[i]
		case "--output":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "inspect: --output requires a value\n")
				return 1
			}
			i++
			output = args[i]
		case "--local-verify":
			localVerify = true
		case "--trust-api":
			localVerify = false
			trustAPIOnly = true
		case "--require-digest":
			requireDigest = true
		case "-h", "--help":
			printInspectUsage()
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "inspect: unknown flag %q\n", args[i])
				return 1
			}
			if ref != "" {
				fmt.Fprintf(os.Stderr, "inspect: unexpected argument %q\n", args[i])
				return 1
			}
			ref = args[i]
		}
	}

	if ref == "" {
		printInspectUsage()
		return 1
	}
	if namespace == "" || artifact == "" {
		fmt.Fprintf(os.Stderr, "inspect: --namespace and --artifact are required\n")
		return 1
	}
	if requireDigest && !strings.HasPrefix(ref, "sha256:") {
		fmt.Fprintf(os.Stderr, "inspect: --require-digest requires a sha256:… digest reference (got %q)\n", ref)
		return 1
	}
	switch output {
	case "", "text":
	case "json":
	default:
		fmt.Fprintf(os.Stderr, "inspect: --output must be text or json\n")
		return 1
	}
	_ = trustAPIOnly

	cfg, err := cliconfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
		return 1
	}

	api := apiclient.New(cfg.APIURL, cfg.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := inspect.Run(ctx, api, inspect.Options{
		Namespace:     namespace,
		Artifact:      artifact,
		Ref:           ref,
		LocalVerify:   localVerify,
		RegistryURL:   cfg.RegistryURL,
		RequireDigest: requireDigest,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
		return 1
	}

	if output == "json" {
		if err := inspect.EncodeJSON(os.Stdout, result); err != nil {
			fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
			return 1
		}
	} else {
		if result.TagWarning != "" {
			fmt.Fprintln(os.Stderr, result.TagWarning)
		}
		for _, line := range inspect.HumanLines(result) {
			fmt.Println(line)
		}
	}

	if inspect.MustFailed(result.MustLines) {
		return 1
	}
	return 0
}

func printInspectUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis inspect <ref> --namespace <ns> --artifact <name> [--output text|json] [--local-verify] [--trust-api] [--require-digest]\n")
	fmt.Fprintf(os.Stderr, "\n<ref> is a local file or directory, sha256:… digest, or semver tag.\n")
	fmt.Fprintf(os.Stderr, "Default: local Sigstore verify + API policy checks. Use --trust-api to skip local crypto.\n")
	fmt.Fprintf(os.Stderr, "Exits non-zero when any Must check fails (FR-DX-005).\n")
}
