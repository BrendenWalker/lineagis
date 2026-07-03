package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrendenWalker/lineagis/internal/analyze"
	"github.com/BrendenWalker/lineagis/internal/report"
	"github.com/BrendenWalker/lineagis/internal/storage/memory"
)

func runReport(args []string) int {
	graphIn, _, rest, ok := parseLineageFlags(args, "report")
	if !ok {
		printReportUsage()
		return 2
	}
	outDir := "generated"
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--out" && i+1 < len(rest) {
			outDir = rest[i+1]
			i++
		}
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		return 1
	}
	if err := report.WriteTree(store.Graph(), outDir); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		return 1
	}
	abs, _ := filepath.Abs(outDir)
	fmt.Fprintf(os.Stderr, "report: wrote artifacts under %s\n", abs)
	return 0
}

func runReportWithAnalyze(args []string) int {
	graphIn, graphOut, rest, ok := parseLineageFlags(args, "report")
	if !ok {
		printReportUsage()
		return 2
	}
	path := "."
	outDir := "generated"
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--out":
			if i+1 < len(rest) {
				i++
				outDir = rest[i]
			}
		default:
			if !stringsHasPrefixDash(rest[i]) {
				path = rest[i]
			}
		}
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		return 1
	}
	if err := analyze.Path(store.Graph(), path); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		return 1
	}
	if err := store.Save(graphOut); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		return 1
	}
	if err := report.WriteTree(store.Graph(), outDir); err != nil {
		fmt.Fprintf(os.Stderr, "report: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "report: analyzed %s → %s, artifacts under %s\n", path, graphOut, outDir)
	return 0
}

func stringsHasPrefixDash(s string) bool {
	return len(s) > 0 && s[0] == '-'
}

func printReportUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis report [--out dir] [--graph-in path]\n")
	fmt.Fprintf(os.Stderr, "       lineagis report analyze [path] [--out dir] [--graph-in path] [--graph-out path]\n")
}
