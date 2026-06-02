package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BrendenWalker/lineagis/internal/core/graph"
	"github.com/BrendenWalker/lineagis/internal/core/model"
)

const DefaultGraphPath = ".lineagis/graph.json"

// Store wraps an in-memory lineage graph with optional snapshot I/O.
type Store struct {
	g *graph.Graph
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{g: graph.New()}
}

// Graph returns the underlying graph (do not replace).
func (s *Store) Graph() *graph.Graph {
	return s.g
}

// Load reads a graph snapshot from path. Missing file yields empty graph.
func (s *Store) Load(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.g = graph.New()
			return nil
		}
		return fmt.Errorf("load graph %s: %w", path, err)
	}
	var snap model.GraphSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return fmt.Errorf("parse graph %s: %w", path, err)
	}
	s.g = graph.New()
	return s.g.LoadSnapshot(snap)
}

// Save writes a deterministic graph snapshot to path.
func (s *Store) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create graph dir: %w", err)
	}
	snap := s.g.Export()
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal graph: %w", err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write graph %s: %w", path, err)
	}
	return nil
}

// ResolveGraphPath returns flag path, env LINEAGIS_GRAPH_FILE, or default.
func ResolveGraphPath(flagPath string) string {
	if flagPath != "" {
		return flagPath
	}
	if env := os.Getenv("LINEAGIS_GRAPH_FILE"); env != "" {
		return env
	}
	return DefaultGraphPath
}
