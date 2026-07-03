package arch

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Rules defines declarative architecture constraints (FR-SA-041).
type Rules struct {
	Layers    map[string]string `yaml:"layers"`
	Forbidden []ForbiddenImport `yaml:"forbidden"`
}

// ForbiddenImport blocks imports between layers.
type ForbiddenImport struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// Violation describes a forbidden import edge.
type Violation struct {
	FromPkg   string
	ToPkg     string
	FromLayer string
	ToLayer   string
	Rule      ForbiddenImport
}

func (v Violation) Error() string {
	return fmt.Sprintf("forbidden import: %s (%s) → %s (%s)",
		v.FromPkg, v.FromLayer, v.ToPkg, v.ToLayer)
}

// DefaultPath is the conventional rules file name.
const DefaultPath = "lineagis.arch.yaml"

// LoadRules reads architecture rules from path.
func LoadRules(path string) (Rules, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Rules{}, err
	}
	var r Rules
	if err := yaml.Unmarshal(data, &r); err != nil {
		return Rules{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(r.Layers) == 0 {
		return Rules{}, fmt.Errorf("parse %s: no layers defined", path)
	}
	return r, nil
}

// PackageLayer maps a package import path to a layer name using rules.
func PackageLayer(modPath, importPath string, rules Rules) string {
	rel := strings.TrimPrefix(importPath, modPath)
	rel = strings.TrimPrefix(rel, "/")
	best := ""
	bestLen := -1
	for layer, prefix := range rules.Layers {
		prefix = strings.TrimSpace(prefix)
		if prefix == "" {
			continue
		}
		if rel == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(rel, prefix) {
			if len(prefix) > bestLen {
				best = layer
				bestLen = len(prefix)
			}
		}
	}
	return best
}

// ValidateImports checks import edges against forbidden layer pairs (FR-SA-042).
func ValidateImports(g *graph.Graph, modPath string, rules Rules) []Violation {
	var out []Violation
	for _, e := range g.Edges() {
		if e.Type != model.EdgeImports {
			continue
		}
		fromPkg := strings.TrimPrefix(e.From, "package:")
		toPkg := strings.TrimPrefix(e.To, "package:")
		if !strings.HasPrefix(fromPkg, modPath) || !strings.HasPrefix(toPkg, modPath) {
			continue
		}
		fromLayer := PackageLayer(modPath, fromPkg, rules)
		toLayer := PackageLayer(modPath, toPkg, rules)
		if fromLayer == "" || toLayer == "" {
			continue
		}
		for _, rule := range rules.Forbidden {
			if fromLayer == rule.From && toLayer == rule.To {
				out = append(out, Violation{
					FromPkg:   fromPkg,
					ToPkg:     toPkg,
					FromLayer: fromLayer,
					ToLayer:   toLayer,
					Rule:      rule,
				})
			}
		}
	}
	return out
}

// FormatViolations returns actionable CI messages.
func FormatViolations(violations []Violation) string {
	if len(violations) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("architecture rule violations:\n")
	for _, v := range violations {
		fmt.Fprintf(&b, "  - %s\n", v.Error())
	}
	b.WriteString("\nRemediation: remove forbidden imports or update lineagis.arch.yaml layers.")
	return b.String()
}
