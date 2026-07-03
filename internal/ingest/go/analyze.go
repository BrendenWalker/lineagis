package goingest

import (
	"fmt"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"

	"github.com/BrendenWalker/lineagis/internal/core/model"
)

// Result holds nodes and edges from Go module analysis.
type Result struct {
	Nodes []model.Node
	Edges []model.Edge
}

// Analyze parses a Go module tree at path and returns code-graph nodes and edges.
func Analyze(path string) (Result, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return Result{}, fmt.Errorf("analyze path: %w", err)
	}
	moduleRoot, err := findModuleRoot(absPath)
	if err != nil {
		return Result{}, err
	}
	modPath, goVersion, err := readModuleMeta(moduleRoot)
	if err != nil {
		return Result{}, err
	}
	scopePrefix, err := scopePrefix(modPath, moduleRoot, absPath)
	if err != nil {
		return Result{}, err
	}

	cfg := &packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedTypes | packages.NeedSyntax | packages.NeedModule,
		Dir:   moduleRoot,
		Tests: false,
	}
	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return Result{}, fmt.Errorf("load packages: %w", err)
	}
	if packages.PrintErrors(pkgs) > 0 {
		return Result{}, fmt.Errorf("load packages: type errors in module (fix compile errors and retry)")
	}

	var res Result
	moduleID := model.ModuleID(modPath)
	modMeta := map[string]string{"path": modPath}
	if goVersion != "" {
		modMeta["go_version"] = goVersion
	}
	res.Nodes = append(res.Nodes, model.Node{ID: moduleID, Type: model.NodeModule, Metadata: modMeta})

	seenPkg := map[string]struct{}{}
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || !inScope(pkg.PkgPath, scopePrefix) {
			continue
		}
		if _, dup := seenPkg[pkg.PkgPath]; dup {
			continue
		}
		seenPkg[pkg.PkgPath] = struct{}{}

		pkgID := model.PackageID(pkg.PkgPath)
		pkgMeta := map[string]string{
			"name": pkg.Name,
			"dir":  relToModule(moduleRoot, pkg.Dir),
		}
		res.Nodes = append(res.Nodes, model.Node{ID: pkgID, Type: model.NodePackage, Metadata: pkgMeta})
		res.Edges = append(res.Edges, model.Edge{From: moduleID, To: pkgID, Type: model.EdgeContains})

		for _, imp := range pkg.Imports {
			if imp == nil || imp.PkgPath == "" {
				continue
			}
			impID := model.PackageID(imp.PkgPath)
			if _, ok := seenPkg[imp.PkgPath]; !ok {
				seenPkg[imp.PkgPath] = struct{}{}
				impMeta := map[string]string{"name": filepath.Base(imp.PkgPath)}
				if imp.PkgPath == modPath {
					impMeta["dir"] = "."
				}
				res.Nodes = append(res.Nodes, model.Node{ID: impID, Type: model.NodePackage, Metadata: impMeta})
			}
			res.Edges = append(res.Edges, model.Edge{From: pkgID, To: impID, Type: model.EdgeImports})
		}

		fileSet := pkg.Fset
		if fileSet == nil {
			fileSet = token.NewFileSet()
		}
		for _, filename := range uniqueFiles(pkg) {
			relFile := relToModule(moduleRoot, filename)
			fileID := model.FileID(relFile)
			res.Nodes = append(res.Nodes, model.Node{
				ID:   fileID,
				Type: model.NodeFile,
				Metadata: map[string]string{
					"language": "go",
				},
			})
			res.Edges = append(res.Edges, model.Edge{From: pkgID, To: fileID, Type: model.EdgeContains})
		}

		if pkg.Types != nil {
			for _, name := range pkg.Types.Scope().Names() {
				obj := pkg.Types.Scope().Lookup(name)
				if obj == nil || !obj.Exported() {
					continue
				}
				symName := name
				kind := symbolKind(obj)
				symID := model.SymbolID(pkg.PkgPath, symName)
				res.Nodes = append(res.Nodes, model.Node{
					ID:   symID,
					Type: model.NodeSymbol,
					Metadata: map[string]string{
						"kind": kind,
					},
				})
				if pos := fileSet.Position(obj.Pos()); pos.Filename != "" {
					fileID := model.FileID(relToModule(moduleRoot, pos.Filename))
					res.Edges = append(res.Edges, model.Edge{From: fileID, To: symID, Type: model.EdgeContains})
				}
			}
		}
	}

	return res, nil
}

func findModuleRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	for d := abs; ; d = filepath.Dir(d) {
		if _, err := os.Stat(filepath.Join(d, "go.mod")); err == nil {
			return d, nil
		}
		if d == filepath.Dir(d) {
			return "", fmt.Errorf("no go.mod found above %s", dir)
		}
	}
}

func readModuleMeta(moduleRoot string) (modulePath, goVersion string, err error) {
	data, err := os.ReadFile(filepath.Join(moduleRoot, "go.mod"))
	if err != nil {
		return "", "", fmt.Errorf("read go.mod: %w", err)
	}
	f, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return "", "", fmt.Errorf("parse go.mod: %w", err)
	}
	if f.Module == nil || f.Module.Mod.Path == "" {
		return "", "", fmt.Errorf("go.mod: missing module directive")
	}
	goVersion = f.Go.Version
	return f.Module.Mod.Path, goVersion, nil
}

func scopePrefix(modPath, moduleRoot, absPath string) (string, error) {
	rel, err := filepath.Rel(moduleRoot, absPath)
	if err != nil {
		return "", err
	}
	rel = filepath.ToSlash(rel)
	if rel == "." {
		return modPath, nil
	}
	return modPath + "/" + strings.TrimSuffix(rel, "/"), nil
}

func inScope(pkgPath, scopePrefix string) bool {
	return pkgPath == scopePrefix || strings.HasPrefix(pkgPath, scopePrefix+"/")
}

func relToModule(moduleRoot, path string) string {
	if path == "" {
		return ""
	}
	rel, err := filepath.Rel(moduleRoot, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func uniqueFiles(pkg *packages.Package) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, f := range pkg.GoFiles {
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

func symbolKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		if obj.(*types.Func).Signature().Recv() != nil {
			return "method"
		}
		return "func"
	case *types.TypeName:
		switch obj.Type().Underlying().(type) {
		case *types.Interface:
			return "interface"
		case *types.Struct:
			return "struct"
		default:
			return "type"
		}
	default:
		return "symbol"
	}
}
