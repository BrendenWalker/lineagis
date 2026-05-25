package publish

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BrendenWalker/verity/internal/registry"
)

// CollectFiles walks publishPath and returns file layers per ADR-0001 (files only, no symlinks).
func CollectFiles(publishPath string) ([]registry.FileLayer, string, error) {
	publishPath = strings.TrimSpace(publishPath)
	if publishPath == "" {
		return nil, "", fmt.Errorf("publish path is required")
	}

	info, err := os.Stat(publishPath)
	if err != nil {
		return nil, "", fmt.Errorf("stat publish path: %w", err)
	}

	var root string
	if info.IsDir() {
		root, err = filepath.Abs(publishPath)
		if err != nil {
			return nil, "", err
		}
	} else {
		root = filepath.Dir(publishPath)
		if root == "" {
			root = "."
		}
		root, err = filepath.Abs(root)
		if err != nil {
			return nil, "", err
		}
	}

	var layers []registry.FileLayer
	err = filepath.WalkDir(publishPath, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		mode, err := d.Info()
		if err != nil {
			return err
		}
		if mode.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks are not supported: %s", path)
		}

		absPath, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, absPath)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "../") {
			rel = filepath.ToSlash(filepath.Base(absPath))
		}

		data, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}

		t := mode.ModTime()
		layers = append(layers, registry.FileLayer{
			Path:    rel,
			Data:    data,
			Created: &t,
		})
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if len(layers) == 0 {
		return nil, "", fmt.Errorf("no files found under %s", publishPath)
	}
	return layers, publishPath, nil
}
