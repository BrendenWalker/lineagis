package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/report"
	"github.com/BrendenWalker/lineagis/internal/storage/memory"
)

func runImpact(args []string) int {
	if len(args) < 2 || args[0] != "package" {
		printImpactUsage()
		return 2
	}
	graphIn, _, rest, ok := parseLineageFlags(args[1:], "impact")
	if !ok {
		printImpactUsage()
		return 2
	}
	format := "text"
	importPath := ""
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--format" && i+1 < len(rest) {
			format = rest[i+1]
			i++
			continue
		}
		if importPath == "" {
			importPath = rest[i]
		}
	}
	if importPath == "" {
		printImpactUsage()
		return 2
	}
	pkgID, err := model.ParsePackageRef(importPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "impact: %v\n", err)
		return 1
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "impact: %v\n", err)
		return 1
	}
	res, err := report.ImpactPackage(store.Graph(), pkgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "impact: %v\n", err)
		return 1
	}
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	} else {
		fmt.Print(report.ImpactMarkdown(res))
	}
	return 0
}

func runExplain(args []string) int {
	if len(args) < 2 || args[0] != "dependency" {
		printExplainUsage()
		return 2
	}
	graphIn, _, rest, ok := parseLineageFlags(args[1:], "explain")
	if !ok {
		printExplainUsage()
		return 2
	}
	format := "text"
	modulePath := ""
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--format" && i+1 < len(rest) {
			format = rest[i+1]
			i++
			continue
		}
		if modulePath == "" {
			modulePath = rest[i]
		}
	}
	if modulePath == "" {
		printExplainUsage()
		return 2
	}
	modPath, err := model.ParseModuleRef(modulePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explain: %v\n", err)
		return 1
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "explain: %v\n", err)
		return 1
	}
	res, err := report.ExplainDependency(store.Graph(), modPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "explain: %v\n", err)
		return 1
	}
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	} else {
		fmt.Print(report.ExplainMarkdown(res))
	}
	return 0
}

func runWhyPackage(args []string) int {
	graphIn, _, rest, ok := parseLineageFlags(args, "why")
	if !ok {
		printWhyPackageUsage()
		return 2
	}
	format := "text"
	importPath := ""
	for i := 0; i < len(rest); i++ {
		if rest[i] == "--format" && i+1 < len(rest) {
			format = rest[i+1]
			i++
			continue
		}
		if importPath == "" {
			importPath = rest[i]
		}
	}
	if importPath == "" {
		printWhyPackageUsage()
		return 2
	}
	pkgID, err := model.ParsePackageRef(importPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "why: %v\n", err)
		return 1
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "why: %v\n", err)
		return 1
	}
	res, err := report.WhyPackage(store.Graph(), pkgID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "why: %v\n", err)
		return 1
	}
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	} else {
		fmt.Print(report.SummaryPackageWhy(res))
	}
	return 0
}

func printImpactUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis impact package <importPath> [--format json|text] [--graph-in path]\n")
}

func printExplainUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis explain dependency <modulePath> [--format json|text] [--graph-in path]\n")
}

func printWhyPackageUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis why package <importPath> [--format json|text] [--graph-in path]\n")
}
