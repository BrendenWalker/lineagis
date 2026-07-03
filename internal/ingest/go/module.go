package goingest

import (
	"path/filepath"
)

// ModuleRoot finds the Go module root containing path.
func ModuleRoot(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return findModuleRoot(abs)
}

// ModulePath reads the module path from go.mod at moduleRoot.
func ModulePath(moduleRoot string) (string, error) {
	path, _, err := readModuleMeta(moduleRoot)
	return path, err
}
