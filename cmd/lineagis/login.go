package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/BrendenWalker/lineagis/internal/cliauth"
)

func runLogin(args []string) int {
	if len(args) > 0 {
		if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
			printLoginUsage()
			return 0
		}
		fmt.Fprintf(os.Stderr, "login: unexpected argument %q\n", args[0])
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
	fmt.Fprintf(os.Stderr, "Usage: lineagis login\n")
	fmt.Fprintf(os.Stderr, "\nSaves API token from LINEAGIS_TOKEN, LINEAGIS_DEV_TOKEN, or GitHub Actions OIDC to ~/.lineagis/config.\n")
}
