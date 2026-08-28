package api

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"tazzledazzle/nerv/nerv/internal/cihook"
	"tazzledazzle/nerv/nerv/internal/depgraph"
	"tazzledazzle/nerv/nerv/internal/metrics"
	"tazzledazzle/nerv/nerv/internal/registry"
	"tazzledazzle/nerv/nerv/internal/template"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()

	dir := t.TempDir()

	reg, err := registry.NewBoltStore(filepath.Join(dir, "registry.db"))
	if err != nil {
		t.Fatalf("NewBoltStore() error = %v", err)
	}
	t.Cleanup(func() { _ = reg.Close() })

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

	return &Server{
		Reg:      reg,
		Engine:   template.NewEngine(catalog, pointers),
		Resolver: depgraph.NewResolver(depgraph.NewEnforcer(depgraph.Policy{}), depgraph.NewCache()),
		Hook:     cihook.NewStubHook(),
		Graph:    depgraph.NewGraph(),
		Metrics:  metrics.New(),
		DestRoot: filepath.Join(dir, "generated"),
	}
}

func TestCreateProjectSuccess(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	body, _ := json.Marshal(NewProjectRequest{
		Name:     "widget-api",
		Language: "go",
		Deps:     []DepRequest{{Name: "otel-sdk", Version: "1.4.2"}},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}

	var resp NewProjectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Project.Name != "widget-api" {
		t.Errorf("Project.Name = %q, want %q", resp.Project.Name, "widget-api")
	}
	if resp.DepsCacheHit {
		t.Error("first request DepsCacheHit = true, want false")
	}
}

func TestCreateProjectCacheHitAcrossRequests(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	deps := []DepRequest{{Name: "otel-sdk", Version: "1.4.2"}}

	post := func(name string) NewProjectResponse {
		body, _ := json.Marshal(NewProjectRequest{Name: name, Language: "go", Deps: deps})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		mux.ServeHTTP(rec, req)
		if rec.Code != 201 {
			t.Fatalf("POST %q status = %d, body = %s", name, rec.Code, rec.Body.String())
		}
		var resp NewProjectResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshaling response for %q: %v", name, err)
		}
		return resp
	}

	first := post("svc-a")
	if first.DepsCacheHit {
		t.Error("first POST DepsCacheHit = true, want false")
	}

	second := post("svc-b") // different project, same dep set
	if !second.DepsCacheHit {
		t.Error("second POST (same deps, same running server) DepsCacheHit = false, want true")
	}
}

func TestCreateProjectRejectsPolicyViolation(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	body, _ := json.Marshal(NewProjectRequest{
		Name:     "bad-project",
		Language: "go",
		Deps:     []DepRequest{{Name: "some-lib", Version: "^2.0.0"}},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	if rec.Code != 422 {
		t.Fatalf("status = %d, want 422, body = %s", rec.Code, rec.Body.String())
	}

	projects, err := srv.Reg.List("")
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected no registered projects after policy violation, got %d", len(projects))
	}
}

func TestListProjectsFiltersByLanguage(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	for _, p := range []struct{ name, lang string }{
		{"go-svc", "go"}, {"py-svc", "python"},
	} {
		body, _ := json.Marshal(NewProjectRequest{Name: p.name, Language: p.lang})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		mux.ServeHTTP(rec, req)
		if rec.Code != 201 {
			t.Fatalf("seeding %q status = %d, body = %s", p.name, rec.Code, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/projects?lang=go", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}

	var projects []registry.Project
	if err := json.Unmarshal(rec.Body.Bytes(), &projects); err != nil {
		t.Fatalf("unmarshaling: %v", err)
	}
	if len(projects) != 1 || projects[0].Language != "go" {
		t.Errorf("GET /projects?lang=go returned %+v, want exactly one go project", projects)
	}
}

func TestHealthz(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/healthz", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestCreateProjectResponseHasNonZeroCreatedAt(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	body, _ := json.Marshal(NewProjectRequest{Name: "svc-c", Language: "go"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
	mux.ServeHTTP(rec, req)

	if rec.Code != 201 {
		t.Fatalf("status = %d, want 201, body = %s", rec.Code, rec.Body.String())
	}

	var resp NewProjectResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshaling response: %v", err)
	}
	if resp.Project.CreatedAt.IsZero() {
		t.Error("Project.CreatedAt in POST response is zero, want it populated at creation time")
	}
}

func TestAddDependencyAndGetDependents(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	create := func(name string) string {
		body, _ := json.Marshal(NewProjectRequest{Name: name, Language: "go"})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		mux.ServeHTTP(rec, req)
		if rec.Code != 201 {
			t.Fatalf("creating %q: status = %d, body = %s", name, rec.Code, rec.Body.String())
		}
		var resp NewProjectResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("unmarshaling create response for %q: %v", name, err)
		}
		return resp.Project.ID
	}

	apiID := create("api-svc")
	authID := create("auth-lib")

	depBody, _ := json.Marshal(DependsOnRequest{DependsOnID: authID})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/projects/"+apiID+"/depends-on", bytes.NewReader(depBody))
	mux.ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("POST depends-on status = %d, body = %s", rec.Code, rec.Body.String())
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/projects/"+authID+"/dependents", nil)
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("GET dependents status = %d, body = %s", rec2.Code, rec2.Body.String())
	}

	var depResp struct {
		Dependents []string `json:"dependents"`
	}
	if err := json.Unmarshal(rec2.Body.Bytes(), &depResp); err != nil {
		t.Fatalf("unmarshaling dependents response: %v", err)
	}
	if len(depResp.Dependents) != 1 || depResp.Dependents[0] != apiID {
		t.Errorf("Dependents(auth-lib) = %v, want [%q]", depResp.Dependents, apiID)
	}
}

func TestAddDependencyRejectsUnknownProject(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	depBody, _ := json.Marshal(DependsOnRequest{DependsOnID: "does-not-exist"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/projects/also-does-not-exist/depends-on", bytes.NewReader(depBody))
	mux.ServeHTTP(rec, req)

	if rec.Code != 404 {
		t.Errorf("status = %d, want 404, body = %s", rec.Code, rec.Body.String())
	}
}

func TestAddDependencyRejectsCycle(t *testing.T) {
	srv := newTestServer(t)
	mux := srv.Router()

	create := func(name string) string {
		body, _ := json.Marshal(NewProjectRequest{Name: name, Language: "go"})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/projects", bytes.NewReader(body))
		mux.ServeHTTP(rec, req)
		var resp NewProjectResponse
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		return resp.Project.ID
	}

	aID := create("proj-a")
	bID := create("proj-b")

	link := func(from, to string) int {
		body, _ := json.Marshal(DependsOnRequest{DependsOnID: to})
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/projects/"+from+"/depends-on", bytes.NewReader(body))
		mux.ServeHTTP(rec, req)
		return rec.Code
	}

	if code := link(aID, bID); code != 201 {
		t.Fatalf("a->b link status = %d, want 201", code)
	}
	if code := link(bID, aID); code != 409 {
		t.Errorf("b->a link (cycle) status = %d, want 409", code)
	}
}
