package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/lineagis/internal/core/engine"
	"github.com/BrendenWalker/lineagis/internal/core/model"
	"github.com/BrendenWalker/lineagis/internal/core/query"
	"github.com/BrendenWalker/lineagis/internal/lineage"
	"github.com/BrendenWalker/lineagis/internal/storage/memory"
)

func runIngest(args []string) int {
	graphIn, graphOut, paths, ok := parseLineageFlags(args, "ingest")
	if !ok {
		printIngestUsage()
		return 2
	}
	if len(paths) == 0 {
		fmt.Fprintf(os.Stderr, "ingest: at least one file required\n")
		return 2
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "ingest: %v\n", err)
		return 1
	}
	if err := lineage.IngestFiles(store.Graph(), paths...); err != nil {
		fmt.Fprintf(os.Stderr, "ingest: %v\n", err)
		return 1
	}
	if err := store.Save(graphOut); err != nil {
		fmt.Fprintf(os.Stderr, "ingest: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "ingested %d file(s) → %s (%d nodes, %d edges)\n",
		len(paths), graphOut, store.Graph().NodeCount(), store.Graph().EdgeCount())
	return 0
}

func runTrace(args []string) int {
	graphIn, _, rest, ok := parseLineageFlags(args, "trace")
	if !ok {
		printTraceUsage()
		return 2
	}
	format := "text"
	ref, flags := splitRefAndFlags(rest)
	for i := 0; i < len(flags); i++ {
		if flags[i] == "--format" && i+1 < len(flags) {
			format = flags[i+1]
			i++
		}
	}
	if ref == "" {
		printTraceUsage()
		return 2
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "trace: %v\n", err)
		return 1
	}
	res, err := query.Trace(store.Graph(), ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "trace: %v\n", err)
		return 1
	}
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	} else {
		fmt.Println(query.SummaryTrace(res))
		if len(res.Verification.Findings) > 0 {
			for _, f := range res.Verification.Findings {
				fmt.Printf("  ⚠ %s\n", f)
			}
		}
	}
	return 0
}

func runWhy(args []string) int {
	graphIn, _, rest, ok := parseLineageFlags(args, "why")
	if !ok {
		printWhyUsage()
		return 2
	}
	format := "text"
	ref, flags := splitRefAndFlags(rest)
	for i := 0; i < len(flags); i++ {
		if flags[i] == "--format" && i+1 < len(flags) {
			format = flags[i+1]
			i++
		}
	}
	if ref == "" {
		printWhyUsage()
		return 2
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "why: %v\n", err)
		return 1
	}
	res, err := query.Why(store.Graph(), ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "why: %v\n", err)
		return 1
	}
	if format == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(res)
	} else {
		fmt.Println(query.SummaryWhy(res))
	}
	if !res.Complete {
		return 1
	}
	return 0
}

func runVisualize(args []string) int {
	graphIn, _, rest, ok := parseLineageFlags(args, "visualize")
	if !ok {
		printVisualizeUsage()
		return 2
	}
	format := ""
	ref, flags := splitRefAndFlags(rest)
	for i := 0; i < len(flags); i++ {
		if flags[i] == "--format" && i+1 < len(flags) {
			format = flags[i+1]
			i++
		}
	}
	if ref == "" || format != "dot" {
		printVisualizeUsage()
		return 2
	}
	store := memory.NewStore()
	if err := store.Load(graphIn); err != nil {
		fmt.Fprintf(os.Stderr, "visualize: %v\n", err)
		return 1
	}
	id, err := model.ParseRef(ref)
	if err != nil {
		fmt.Fprintf(os.Stderr, "visualize: %v\n", err)
		return 1
	}
	fmt.Print(engine.ToDOT(store.Graph(), id))
	return 0
}

func parseLineageFlags(args []string, cmd string) (graphIn, graphOut string, paths []string, ok bool) {
	graphIn = memory.ResolveGraphPath("")
	graphOut = graphIn
	var positional []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-h", "--help":
			return "", "", nil, false
		case "--graph-in":
			if i+1 >= len(args) {
				return "", "", nil, false
			}
			i++
			graphIn = args[i]
			if graphOut == memory.ResolveGraphPath("") {
				graphOut = graphIn
			}
		case "--graph-out":
			if i+1 >= len(args) {
				return "", "", nil, false
			}
			i++
			graphOut = args[i]
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "%s: unknown flag %q\n", cmd, args[i])
				return "", "", nil, false
			}
			positional = append(positional, args[i])
		}
	}
	if cmd == "ingest" {
		return graphIn, graphOut, positional, true
	}
	return graphIn, graphOut, positional, true
}

func splitRefAndFlags(args []string) (ref string, flags []string) {
	if len(args) == 0 {
		return "", nil
	}
	ref = args[0]
	if len(args) > 1 {
		flags = args[1:]
	}
	return ref, flags
}

func printIngestUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis ingest <file> [file...] [--graph-in path] [--graph-out path]\n")
}

func printTraceUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis trace <ref> [--format json] [--graph-in path]\n")
}

func printWhyUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis why <ref> [--format json] [--graph-in path]\n")
}

func printVisualizeUsage() {
	fmt.Fprintf(os.Stderr, "Usage: lineagis visualize <ref> --format dot [--graph-in path]\n")
}
