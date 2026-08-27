package registry

import (
	"path/filepath"
	"testing"
)

func newTestStore(t *testing.T) *BoltStore {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "nerv-test.db")
	store, err := NewBoltStore(dbPath)
	if err != nil {
		t.Fatal("NewBoltStore() error = %w", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Error("store.Close() error = %w", err)
		}
	})

	return store
}

func TestRegisterAndGet(t *testing.T) {
	store := newTestStore(t)

	p := Project{
		ID:              "proj-001",
		Name:            "example-service",
		Language:        "go",
		TemplateName:    "service",
		TemplateVersion: "v1.0.0",
		Path:            "/repo/example-service",
	}

	if err := store.Register(p); err != nil {
		t.Fatal("Register() error = %w", err)
	}

	got, err := store.Get("proj-001")
	if err != nil {
		t.Fatal("Get() error = %w", err)
	}
	if got.ID != p.ID || got.Name != p.Name || got.Language != p.Language {
		t.Errorf("Get() = %+v, want fields matching %+v", got, p)
	}
	if got.CreatedAt.IsZero() {
		t.Error("Get().CreatedAt should be auto-populated, got zero value")
	}
}
