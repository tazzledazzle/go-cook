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

func TestRegisterDuplicateIDFails(t *testing.T) {
	store := newTestStore(t)

	p := Project{ID: "proj-dup", Name: "svc", Language: "go"}

	if err := store.Register(p); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err := store.Register(p)
	if err != ErrAlreadyExists {
		t.Errorf("second Register() error = %v", err)
	}
}

func TestGetMissingReturnsNotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.Get("does-not-exist")
	if err != ErrNotFound {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestListFiltersByLanguage(t *testing.T) {
	store := newTestStore(t)

	projects := []Project{
		{ID: "go-1", Name: "svc-a", Language: "go"},
		{ID: "go-2", Name: "svc-b", Language: "go"},
		{ID: "py-1", Name: "svc-c", Language: "python"},
	}

	for _, p := range projects {
		if err := store.Register(p); err != nil {
			t.Fatalf("Register(%q) error = %v", p.ID, err)
		}
	}

	goProjects, err := store.List("go")
	if err != nil {
		t.Errorf("List(\"go\") error = %v", err)
	}

	all, err := store.List("")
	if err != nil {
		t.Fatalf("List(\"\") error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List(\"\") = %+v, want 3", len(all))
	}
}
