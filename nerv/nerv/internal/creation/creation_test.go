package creation

import (
	"path/filepath"
	"testing"

	"tazzledazzle/nerv/nerv/internal/cihook"
	"tazzledazzle/nerv/nerv/internal/depgraph"
	"tazzledazzle/nerv/nerv/internal/directory"
	"tazzledazzle/nerv/nerv/internal/metrics"
	"tazzledazzle/nerv/nerv/internal/registry"
	"tazzledazzle/nerv/nerv/internal/template"
)

func newTestService(t *testing.T) (*Service, *directory.Service) {
	t.Helper()
	dir := t.TempDir()

	reg, err := registry.NewBoltStore(filepath.Join(dir, "registry.db"))
	if err != nil {
		t.Fatalf("NewBoltStore() error = %v", err)
	}
	t.Cleanup(func() { _ = reg.Close() })

	dirSvc, err := directory.New(reg)
	if err != nil {
		t.Fatalf("directory.New() error = %v", err)
	}

	pointers, err := template.NewBoltPointerStore(filepath.Join(dir, "pointers.db"))
	if err != nil {
		t.Fatalf("NewBoltPointerStore() error = %v", err)
	}
	t.Cleanup(func() { _ = pointers.Close() })

	catalog := template.NewMemCatalog()
	if err := catalog.RegisterVersion("service", "v1", template.Files{
		"main.go": `println("{{.ServiceName}}")`,
	}); err != nil {
		t.Fatalf("RegisterVersion() error = %v", err)
	}
	if err := pointers.SetCurrent("service", "v1"); err != nil {
		t.Fatalf("SetCurrent() error = %v", err)
	}

	svc := &Service{
		Engine:    template.NewEngine(catalog, pointers),
		Resolver:  depgraph.NewResolver(depgraph.NewEnforcer(depgraph.Policy{}), depgraph.NewCache()),
		Hook:      cihook.NewStubHook(),
		Directory: dirSvc,
		Metrics:   metrics.New(),
		DestRoot:  filepath.Join(dir, "generated"),
	}

	return svc, dirSvc
}

func TestCreateSuccessRegistersWithDirectory(t *testing.T) {
	svc, dirSvc := newTestService(t)

	result, err := svc.Create(Request{Name: "widget-api", Language: "go"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.Project.Name != "widget-api" {
		t.Errorf("Project.Name = %q, want %q", result.Project.Name, "widget-api")
	}

	// Confirm the Directory Service actually has it — proves creation
	// really did hand off metadata rather than registering it itself.
	got, err := dirSvc.Get(result.Project.ID)
	if err != nil {
		t.Fatalf("dirSvc.Get() error = %v", err)
	}
	if got.ID != result.Project.ID {
		t.Errorf("directory Get() = %+v, want ID %q", got, result.Project.ID)
	}

	searchResults, err := dirSvc.Search("widget")
	if err != nil {
		t.Fatalf("dirSvc.Search() error = %v", err)
	}
	if len(searchResults) != 1 {
		t.Errorf("dirSvc.Search(widget) = %v, want the newly created module indexed and findable", searchResults)
	}
}

func TestCreateWithDependencies(t *testing.T) {
	svc, _ := newTestService(t)

	result, err := svc.Create(Request{
		Name:     "widget-api",
		Language: "go",
		Deps:     []depgraph.Dependency{{Name: "otel-sdk", Constraint: "1.4.2"}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if result.ResolvedDeps["otel-sdk"] != "1.4.2" {
		t.Errorf("ResolvedDeps = %v, want otel-sdk: 1.4.2", result.ResolvedDeps)
	}
	if result.DepsCacheHit {
		t.Error("first Create() DepsCacheHit = true, want false")
	}
}

func TestCreateFailsPolicyViolationBeforeRegistering(t *testing.T) {
	svc, dirSvc := newTestService(t)

	_, err := svc.Create(Request{
		Name:     "bad-project",
		Language: "go",
		Deps:     []depgraph.Dependency{{Name: "some-lib", Constraint: "^2.0.0"}},
	})
	if err == nil {
		t.Fatal("Create() error = nil, want a policy-violation error")
	}

	// Nothing should have reached the Directory Service.
	all, listErr := dirSvc.List("")
	if listErr != nil {
		t.Fatalf("dirSvc.List() error = %v", listErr)
	}
	if len(all) != 0 {
		t.Errorf("dirSvc.List() = %v, want empty — rejected creation should never reach the directory", all)
	}
}

func TestCreateRequiresNameAndLanguage(t *testing.T) {
	svc, _ := newTestService(t)

	_, err := svc.Create(Request{Name: "", Language: "go"})
	if err == nil {
		t.Error("Create() with empty Name: error = nil, want error")
	}

	_, err = svc.Create(Request{Name: "widget", Language: ""})
	if err == nil {
		t.Error("Create() with empty Language: error = nil, want error")
	}
}
