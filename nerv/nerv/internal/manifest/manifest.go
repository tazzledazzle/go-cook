// Package manifest defines Nerv's on-disk dependency manifest format —
// the file a developer maintains locally and that `nervctl lint` reads
// to check policy compliance before ever generating or registering
// anything. This is the "linter" half of the linter-plus-CI-gate
// pattern the real Nerv system used for semver enforcement.
package manifest

import (
	"encoding/json"
	"fmt"
	"os"

	"tazzledazzle/nerv/nerv/internal/depgraph"
)

// DependencyEntry is one dependency as written in the manifest file.
type DependencyEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// ProjectManifest is the full parsed contents of a manifest file.
type ProjectManifest struct {
	Name         string            `json:"name"`
	Language     string            `json:"language"`
	Dependencies []DependencyEntry `json:"dependencies"`
}

// Load reads and parses a manifest file at path.
func Load(path string) (*ProjectManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("manifest: reading %q: %w", path, err)
	}

	var m ProjectManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("manifest: parsing %q: %w", path, err)
	}

	if m.Name == "" {
		return nil, fmt.Errorf("manifest: %q: \"name\" is required", path)
	}

	return &m, nil
}

// ToDependencies converts the manifest's dependency entries into the
// depgraph package's Dependency type, ready to pass to an Enforcer or
// Resolver.
func (m *ProjectManifest) ToDependencies() []depgraph.Dependency {
	deps := make([]depgraph.Dependency, len(m.Dependencies))
	for i, d := range m.Dependencies {
		deps[i] = depgraph.Dependency{Name: d.Name, Constraint: d.Version}
	}
	return deps
}
