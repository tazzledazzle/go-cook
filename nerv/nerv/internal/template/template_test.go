package template

import (
	"os"
	"path/filepath"
	"testing"
)

func newTestEngine(t *testing.T) (*Engine, *MemCatalog, *BoltPointerStore) {
	t.Helper()

	catalog := NewMemCatalog()

	dbPath := filepath.Join(t.TempDir(), "pointers.db")
	pointers, err := NewBoltPointerStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltPointerStore() error = %v", err)
	}
	t.Cleanup(func() {
		if err := pointers.Close(); err != nil {
			t.Errorf("pointers.Close() error = %v", err)
		}
	})

	return NewEngine(catalog, pointers), catalog, pointers
}

func TestRenderCurrentVersion(t *testing.T) {
	engine, catalog, pointers := newTestEngine(t)

	v1Files := Files{
		"main.go": `package main

func main() {
	println("{{.ServiceName}} v1")
}
`,
	}
	if err := catalog.RegisterVersion("service", "v1", v1Files); err != nil {
		t.Fatalf("RegisterVersion(v1) error = %v", err)
	}
	if err := pointers.SetCurrent("service", "v1"); err != nil {
		t.Fatalf("SetCurrent(v1) error = %v", err)
	}

	dest := t.TempDir()
	vars := map[string]interface{}{"ServiceName": "widget-api"}

	if err := engine.Render("service", dest, vars); err != nil {
		t.Fatalf("Render() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dest, "main.go"))
	if err != nil {
		t.Fatalf("reading rendered file: %v", err)
	}
	if !contains(string(got), "widget-api v1") {
		t.Errorf("rendered file = %q, want it to contain %q", got, "widget-api v1")
	}
}

func TestRollbackToOlderVersion(t *testing.T) {
	engine, catalog, pointers := newTestEngine(t)

	v1 := Files{"main.go": `println("{{.ServiceName}} v1 stable")`}
	v2 := Files{"main.go": `println("{{.ServiceName}} v2 BROKEN")`}

	if err := catalog.RegisterVersion("service", "v1", v1); err != nil {
		t.Fatalf("RegisterVersion(v1) error = %v", err)
	}
	if err := catalog.RegisterVersion("service", "v2", v2); err != nil {
		t.Fatalf("RegisterVersion(v2) error = %v", err)
	}

	vars := map[string]interface{}{"ServiceName": "widget-api"}

	// Roll forward to v2.
	if err := pointers.SetCurrent("service", "v2"); err != nil {
		t.Fatalf("SetCurrent(v2) error = %v", err)
	}
	destV2 := t.TempDir()
	if err := engine.Render("service", destV2, vars); err != nil {
		t.Fatalf("Render() at v2 error = %v", err)
	}
	gotV2, _ := os.ReadFile(filepath.Join(destV2, "main.go"))
	if !contains(string(gotV2), "BROKEN") {
		t.Fatalf("expected v2 render to contain BROKEN marker, got %q", gotV2)
	}

	// v2 turned out bad — roll the pointer back to v1 centrally.
	if err := pointers.SetCurrent("service", "v1"); err != nil {
		t.Fatalf("SetCurrent(v1) rollback error = %v", err)
	}
	destV1 := t.TempDir()
	if err := engine.Render("service", destV1, vars); err != nil {
		t.Fatalf("Render() after rollback error = %v", err)
	}
	gotV1, _ := os.ReadFile(filepath.Join(destV1, "main.go"))
	if contains(string(gotV1), "BROKEN") {
		t.Errorf("expected rollback render to avoid BROKEN, got %q", gotV1)
	}
	if !contains(string(gotV1), "stable") {
		t.Errorf("expected rollback render to contain 'stable', got %q", gotV1)
	}
}

func TestRenderWithNoCurrentVersionFails(t *testing.T) {
	engine, catalog, _ := newTestEngine(t)

	if err := catalog.RegisterVersion("service", "v1", Files{"a.go": "x"}); err != nil {
		t.Fatalf("RegisterVersion() error = %v", err)
	}

	err := engine.Render("service", t.TempDir(), nil)
	if err != ErrNoCurrentVersion {
		t.Errorf("Render() error = %v, want ErrNoCurrentVersion", err)
	}
}

func TestRenderUnknownTemplateFails(t *testing.T) {
	engine, _, pointers := newTestEngine(t)

	// Set a pointer for a template that was never registered in the catalog —
	// simulates a pointer/catalog drift bug.
	if err := pointers.SetCurrent("ghost", "v1"); err != nil {
		t.Fatalf("SetCurrent() error = %v", err)
	}

	err := engine.Render("ghost", t.TempDir(), nil)
	if err != ErrTemplateNotFound {
		t.Errorf("Render() error = %v, want ErrTemplateNotFound", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
