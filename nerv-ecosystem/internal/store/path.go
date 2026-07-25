package store

import (
	"os"
	"path/filepath"
)

const defaultDirName = ".modular"
const defaultFileName = "registry.db"

// DefaultPath returns $MODULAR_HOME/registry.db if MODULAR_HOME is set,
// otherwise ~/.modular/registry.db. A shared, home-scoped default (rather
// than a project-local one) is required so Phase 4/5's deps/search commands
// see the same registry regardless of which generated project's directory
// the operator is standing in when they run modular.
func DefaultPath() string {
	if dir := os.Getenv("MODULAR_HOME"); dir != "" {
		return filepath.Clean(filepath.Join(dir, defaultFileName))
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Clean(filepath.Join(".", defaultDirName, defaultFileName))
	}
	return filepath.Clean(filepath.Join(home, defaultDirName, defaultFileName))
}
