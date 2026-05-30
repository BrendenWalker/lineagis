package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/BrendenWalker/verity/internal/cliauth"
)

func runLogin(args []string) int {
	for _, a := range args {
		if a == "-h" || a == "--help" {
			printLoginUsage()
			return 0
		}
		fmt.Fprintf(os.Stderr, "login: unexpected argument %q\n", a)
		return 1
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	f, err := cliauth.Login(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "login: %v\n", err)
		return 1
	}
	path, _ := cliauth.ConfigPath()
	fmt.Fprintf(os.Stderr, "login: credentials saved to %s (api_url=%s)\n", path, f.APIURL)
	return 0
}

func printLoginUsage() {
	fmt.Fprintf(os.Stderr, "Usage: verity login\n")
	fmt.Fprintf(os.Stderr, "\nSaves API token from VERITY_TOKEN, VERITY_DEV_TOKEN, or GitHub Actions OIDC to ~/.verity/config.\n")
}
