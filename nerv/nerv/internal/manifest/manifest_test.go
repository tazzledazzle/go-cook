package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestManifest(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nerv-deps.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test manifest: %v", err)
	}
	return path
}

func TestLoadValidManifest(t *testing.T) {
	path := writeTestManifest(t, `{
		"name": "widget-api",
		"language": "go",
		"dependencies": [
			{"name": "otel-sdk", "version": "1.4.2"},
			{"name": "grpc-go", "version": "1.60.1"}
		]
	}`)

	m, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if m.Name != "widget-api" {
		t.Errorf("Name = %q, want %q", m.Name, "widget-api")
	}
	if len(m.Dependencies) != 2 {
		t.Fatalf("Dependencies has %d entries, want 2", len(m.Dependencies))
	}
	if m.Dependencies[0].Name != "otel-sdk" || m.Dependencies[0].Version != "1.4.2" {
		t.Errorf("Dependencies[0] = %+v, want otel-sdk@1.4.2", m.Dependencies[0])
	}
}

func TestLoadRejectsMissingName(t *testing.T) {
	path := writeTestManifest(t, `{"language": "go", "dependencies": []}`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want an error for missing \"name\"")
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	path := writeTestManifest(t, `{ this is not valid json `)

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() error = nil, want an error for malformed JSON")
	}
}

func TestLoadRejectsMissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/nerv-deps.json")
	if err == nil {
		t.Fatal("Load() error = nil, want an error for a missing file")
	}
}

func TestToDependenciesConvertsEntries(t *testing.T) {
	m := &ProjectManifest{
		Name: "widget-api",
		Dependencies: []DependencyEntry{
			{Name: "otel-sdk", Version: "1.4.2"},
			{Name: "grpc-go", Version: "1.60.1"},
		},
	}

	deps := m.ToDependencies()
	if len(deps) != 2 {
		t.Fatalf("ToDependencies() returned %d entries, want 2", len(deps))
	}
	if deps[0].Name != "otel-sdk" || deps[0].Constraint != "1.4.2" {
		t.Errorf("deps[0] = %+v, want {Name: otel-sdk, Constraint: 1.4.2}", deps[0])
	}
}

func TestToDependenciesHandlesEmptyManifest(t *testing.T) {
	m := &ProjectManifest{Name: "no-deps-svc"}

	deps := m.ToDependencies()
	if len(deps) != 0 {
		t.Errorf("ToDependencies() returned %d entries, want 0", len(deps))
	}
}
