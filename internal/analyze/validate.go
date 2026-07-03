package analyze

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrendenWalker/lineagis/internal/arch"
	"github.com/BrendenWalker/lineagis/internal/core/graph"
	goingest "github.com/BrendenWalker/lineagis/internal/ingest/go"
)

// ValidateArchitecture loads lineagis.arch.yaml from moduleRoot and checks imports.
func ValidateArchitecture(g *graph.Graph, moduleRoot string) ([]arch.Violation, error) {
	rulesPath := filepath.Join(moduleRoot, arch.DefaultPath)
	if _, err := os.Stat(rulesPath); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	rules, err := arch.LoadRules(rulesPath)
	if err != nil {
		return nil, err
	}
	modPath, err := goingest.ModulePath(moduleRoot)
	if err != nil {
		return nil, err
	}
	return arch.ValidateImports(g, modPath, rules), nil
}

// ValidateArchitectureStrict returns error when violations exist.
func ValidateArchitectureStrict(g *graph.Graph, moduleRoot string) error {
	violations, err := ValidateArchitecture(g, moduleRoot)
	if err != nil {
		return err
	}
	if len(violations) > 0 {
		return fmt.Errorf("%s", arch.FormatViolations(violations))
	}
	return nil
}
