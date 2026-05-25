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
	var path, namespace, artifact, tag string
	for i := 0; i < len(args); i++ {
		switch args[i] {
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
		Namespace: namespace,
		Artifact:  artifact,
		Tag:       tag,
		Path:      path,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "publish: %v\n", err)
		return 1
	}

	fmt.Println(digest)
	return 0
}

func printPublishUsage() {
	fmt.Fprintf(os.Stderr, "Usage: verity publish <path> --namespace <ns> --artifact <name> [--tag <tag>]\n")
}
