package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BrendenWalker/lineagis/internal/cliconfig"
	"github.com/BrendenWalker/lineagis/internal/pull"
)

func runPull(args []string) int {
	var output string
	verify := false
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "pull: --output requires a value\n")
				return 1
			}
			i++
			output = args[i]
		case "--verify":
			verify = true
		case "-h", "--help":
			printPullUsage()
			return 0
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "pull: unknown flag %q\n", args[i])
				return 1
			}
			positional = append(positional, args[i])
		}
	}
	if len(positional) != 1 {
		printPullUsage()
		return 1
	}

	cfg, err := cliconfig.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pull: %v\n", err)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	digest, err := pull.Pull(ctx, pull.Options{
		Ref:         positional[0],
		OutputDir:   output,
		Verify:      verify,
		APIURL:      cfg.APIURL,
		RegistryURL: cfg.RegistryURL,
		Token:       cfg.Token,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "pull: %v\n", err)
		return 1
	}
	fmt.Printf("%s\n", digest)
	return 0
}

func printPullUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis pull <namespace/.../artifact@sha256:…|artifact:tag> [-o dir] [--verify]\n")
	fmt.Fprintf(os.Stderr, "\nResolves tag via Lineagis API, pulls OCI manifest and layers from LINEAGIS_REGISTRY_URL.\n")
}
