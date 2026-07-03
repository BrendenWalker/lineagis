package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/report"
	"github.com/BrendenWalker/lineagis/internal/storage/memory"
	goingest "github.com/BrendenWalker/lineagis/internal/ingest/go"
)

func runAnalyze(args []string) int {
	graphIn, graphOut, rest, ok := parseLineageFlags(args, "analyze")
	if !ok {
		printAnalyzeUsage()
		return 2
	}
	format := "text"
	path := "."
	outDir := ""
	validateArch := false
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--format":
			if i+1 < len(rest) {
				i++
				format = rest[i]
			}
		case "--out":
			if i+1 < len(rest) {
				i++
				outDir = rest[i]
			}
		case "--validate-arch":
			validateArch = true
		default:
			if !strings.HasPrefix(rest[i], "-") {
				path = rest[i]
			}
		}
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "analyze: %v\n", err)
		return 1
	}
	if err := analyze.Path(store.Graph(), path); err != nil {
		fmt.Fprintf(os.Stderr, "analyze: %v\n", err)
		return 1
	}
	if validateArch {
		moduleRoot, err := goingest.ModuleRoot(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "analyze: %v\n", err)
			return 1
		}
		if err := analyze.ValidateArchitectureStrict(store.Graph(), moduleRoot); err != nil {
			fmt.Fprintf(os.Stderr, "analyze: %v\n", err)
			return 1
		}
	}
	if err := store.Save(graphOut); err != nil {
		fmt.Fprintf(os.Stderr, "analyze: %v\n", err)
		return 1
	}
	if outDir != "" {
		if err := report.WriteTree(store.Graph(), outDir); err != nil {
			fmt.Fprintf(os.Stderr, "analyze: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "analyze: wrote reports under %s\n", outDir)
	}
	switch format {
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(store.Graph().Export()); err != nil {
			fmt.Fprintf(os.Stderr, "analyze: %v\n", err)
			return 1
		}
	case "dot":
		fmt.Print(analyze.PackageImportDOT(store.Graph()))
	default:
		snap := store.Graph().Export()
		fmt.Fprintf(os.Stderr, "analyzed %s → %s (%d nodes, %d edges, %s)\n",
			path, graphOut, store.Graph().NodeCount(), store.Graph().EdgeCount(), snap.SchemaVersion)
	}
	return 0
}

func printAnalyzeUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis analyze [path] [--format json|dot|text] [--out dir] [--validate-arch] [--graph-in path] [--graph-out path]\n")
}
