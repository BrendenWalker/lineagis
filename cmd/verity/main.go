package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/BrendenWalker/verity/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	switch {
	case len(args) > 0 && args[0] == "publish":
		return runPublish(args[1:])
	case len(args) > 0 && args[0] == "inspect":
		return runInspect(args[1:])
	}

	fs := flag.NewFlagSet("verity", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *showVersion {
		fmt.Println(version.Version)
		return 0
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "verity: unknown command %q\n", fs.Arg(0))
		return 1
	}
	fmt.Fprintf(os.Stderr, "Usage: verity publish <path> [flags] | verity inspect <ref> [flags] | verity [--version]\n")
	return 1
}
