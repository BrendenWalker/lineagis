package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/cliconfig"
	"github.com/BrendenWalker/verity/internal/inspect"
)

func runInspect(args []string) int {
	var namespace, artifact, ref, output string
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
	switch output {
	case "", "text":
	case "json":
	default:
		fmt.Fprintf(os.Stderr, "inspect: --output must be text or json\n")
		return 1
	}

	cfg, err := cliconfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "inspect: %v\n", err)
		return 1
	}

	api := apiclient.New(cfg.APIURL, cfg.Token)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	result, err := inspect.Run(ctx, api, inspect.Options{
		Namespace: namespace,
		Artifact:  artifact,
		Ref:       ref,
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
	fmt.Fprintf(os.Stderr, "Usage: verity inspect <ref> --namespace <ns> --artifact <name> [--output text|json]\n")
	fmt.Fprintf(os.Stderr, "\n<ref> is a local file or directory, sha256:… digest, or semver tag.\n")
	fmt.Fprintf(os.Stderr, "Trust checks use the Verity API (signature verification is server-side).\n")
	fmt.Fprintf(os.Stderr, "Exits non-zero when any Must check fails (FR-DX-005).\n")
}
