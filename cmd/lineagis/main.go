package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/BrendenWalker/lineagis/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	switch {
	case len(args) > 0 && args[0] == "ingest":
		return runIngest(args[1:])
	case len(args) > 0 && args[0] == "trace":
		return runTrace(args[1:])
	case len(args) > 0 && args[0] == "why":
		return runWhy(args[1:])
	case len(args) > 0 && args[0] == "visualize":
		return runVisualize(args[1:])
	case len(args) > 0 && args[0] == "analyze":
		return runAnalyze(args[1:])
	}

	fs := flag.NewFlagSet("lineagis", flag.ContinueOnError)
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
		fmt.Fprintf(os.Stderr, "lineagis: unknown command %q\n", fs.Arg(0))
		return 1
	}
	printUsage()
	return 1
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis ingest <files...> | lineagis analyze [path] | lineagis trace <ref> | lineagis why <ref> | lineagis visualize <ref> --format dot | lineagis [--version]\n")
}
