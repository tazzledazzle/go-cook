package directory

import (
	"path/filepath"
	"testing"

	"tazzledazzle/nerv/nerv/internal/registry"
)

func newTestService(t *testing.T, providers ...CodeSearchProvider) *Service {
	t.Helper()

	reg, err := registry.NewBoltStore(filepath.Join(t.TempDir(), "registry.db"))
	if err != nil {
		t.Fatalf("NewBoltStore() error = %v", err)
	}
	t.Cleanup(func() { _ = reg.Close() })

	svc, err := New(reg, providers...)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	return svc
}

func TestRegisterPersistsAndIndexes(t *testing.T) {
	svc := newTestService(t)

	p := registry.Project{ID: "proj-1", Name: "widget-api", Language: "go", TemplateName: "service"}
	if err := svc.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, err := svc.Get("proj-1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Name != "widget-api" {
		t.Errorf("Get().Name = %q, want %q", got.Name, "widget-api")
	}

	results, err := svc.Search("widget")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "proj-1" {
		t.Errorf("Search(widget) = %v, want exactly proj-1", results)
	}
}

func TestSearchAcrossMultipleModules(t *testing.T) {
	svc := newTestService(t)

	modules := []registry.Project{
		{ID: "p1", Name: "widget-api", Language: "go", TemplateName: "service"},
		{ID: "p2", Name: "auth-lib", Language: "go", TemplateName: "library"},
		{ID: "p3", Name: "data-pipeline", Language: "python", TemplateName: "service"},
	}
	for _, m := range modules {
		if err := svc.Register(m); err != nil {
			t.Fatalf("Register(%q) error = %v", m.ID, err)
		}
	}

	results, err := svc.Search("go service")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) == 0 || results[0].ID != "p1" {
		t.Errorf("Search(go service) top result = %v, want p1 (matches both go and service)", results)
	}
}

func TestNewRebuildsIndexFromExistingRegistrar(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "registry.db")

	// First "process": register a module, then close.
	reg1, err := registry.NewBoltStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltStore() error = %v", err)
	}
	svc1, err := New(reg1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := svc1.Register(registry.Project{ID: "durable-1", Name: "widget-api", Language: "go"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := svc1.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// "Restart": open a fresh Service over the same DB file. Its
	// in-memory index starts empty and must be rebuilt from what's
	// already persisted.
	reg2, err := registry.NewBoltStore(dbPath)
	if err != nil {
		t.Fatalf("NewBoltStore() (second open) error = %v", err)
	}
	defer reg2.Close()

	svc2, err := New(reg2)
	if err != nil {
		t.Fatalf("New() (second instance) error = %v", err)
	}

	results, err := svc2.Search("widget")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 1 || results[0].ID != "durable-1" {
		t.Errorf("Search() after restart = %v, want durable-1 found via rebuilt index", results)
	}
}

func TestCodeSearchLinksBuildsURLsForEachProvider(t *testing.T) {
	svc := newTestService(t,
		OpenGrokProvider{BaseURL: "https://opengrok.internal.example.com"},
		SourcegraphProvider{BaseURL: "https://sourcegraph.internal.example.com"},
	)

	links := svc.CodeSearchLinks("widget-api")

	if len(links) != 2 {
		t.Fatalf("CodeSearchLinks() returned %d entries, want 2", len(links))
	}
	if links["opengrok"] != "https://opengrok.internal.example.com/source/search?full=widget-api" {
		t.Errorf("opengrok link = %q", links["opengrok"])
	}
	if links["sourcegraph"] != "https://sourcegraph.internal.example.com/search?q=context%3Aglobal+widget-api" {
		t.Errorf("sourcegraph link = %q", links["sourcegraph"])
	}
}

func TestCodeSearchLinksEmptyWhenNoProvidersConfigured(t *testing.T) {
	svc := newTestService(t) // no providers

	links := svc.CodeSearchLinks("anything")
	if len(links) != 0 {
		t.Errorf("CodeSearchLinks() = %v, want empty map when no providers configured", links)
	}
}

func TestSearchReturnsEmptyForNoMatches(t *testing.T) {
	svc := newTestService(t)
	if err := svc.Register(registry.Project{ID: "p1", Name: "widget-api", Language: "go"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	results, err := svc.Search("nonexistent")
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Search(nonexistent) = %v, want empty", results)
	}
}
