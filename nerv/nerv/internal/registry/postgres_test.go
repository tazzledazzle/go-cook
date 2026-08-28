package registry

import (
	"context"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// newTestPostgresStore starts a real, disposable Postgres container via
// testcontainers-go, connects a PostgresStore to it, and registers
// cleanup to terminate the container when the test finishes. Requires
// Docker to be running.
func newTestPostgresStore(t *testing.T) *PostgresStore {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("nerv_test"),
		tcpostgres.WithUsername("nerv"),
		tcpostgres.WithPassword("nerv"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("starting postgres testcontainer: %v (is Docker running?)", err)
	}
	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Errorf("terminating postgres testcontainer: %v", err)
		}
	})

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting postgres connection string: %v", err)
	}

	store, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore() error = %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("store.Close() error = %v", err)
		}
	})

	return store
}

func TestPostgresRegisterAndGet(t *testing.T) {
	store := newTestPostgresStore(t)

	p := Project{
		ID:              "pg-proj-001",
		Name:            "example-service",
		Language:        "go",
		TemplateName:    "service",
		TemplateVersion: "v1.0.0",
		Path:            "/repo/example-service",
	}

	if err := store.Register(p); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	got, err := store.Get("pg-proj-001")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.ID != p.ID || got.Name != p.Name || got.Language != p.Language {
		t.Errorf("Get() = %+v, want fields matching %+v", got, p)
	}
	if got.CreatedAt.IsZero() {
		t.Error("Get().CreatedAt should be auto-populated, got zero value")
	}
}

func TestPostgresRegisterDuplicateIDFails(t *testing.T) {
	store := newTestPostgresStore(t)

	p := Project{ID: "pg-proj-dup", Name: "svc", Language: "go"}

	if err := store.Register(p); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}

	err := store.Register(p)
	if err != ErrAlreadyExists {
		t.Errorf("second Register() error = %v, want ErrAlreadyExists", err)
	}
}

func TestPostgresGetMissingReturnsNotFound(t *testing.T) {
	store := newTestPostgresStore(t)

	_, err := store.Get("does-not-exist")
	if err != ErrNotFound {
		t.Errorf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestPostgresListFiltersByLanguage(t *testing.T) {
	store := newTestPostgresStore(t)

	projects := []Project{
		{ID: "pg-go-1", Name: "svc-a", Language: "go"},
		{ID: "pg-go-2", Name: "svc-b", Language: "go"},
		{ID: "pg-py-1", Name: "svc-c", Language: "python"},
	}
	for _, p := range projects {
		if err := store.Register(p); err != nil {
			t.Fatalf("Register(%q) error = %v", p.ID, err)
		}
	}

	goProjects, err := store.List("go")
	if err != nil {
		t.Fatalf("List(\"go\") error = %v", err)
	}
	if len(goProjects) != 2 {
		t.Errorf("List(\"go\") returned %d projects, want 2", len(goProjects))
	}

	all, err := store.List("")
	if err != nil {
		t.Fatalf("List(\"\") error = %v", err)
	}
	if len(all) != 3 {
		t.Errorf("List(\"\") returned %d projects, want 3", len(all))
	}
}

// TestPostgresDataSurvivesReconnect is the one test that has no bbolt
// analog: it proves data written by one PostgresStore connection is
// visible to a brand-new connection against the same database — the
// actual point of using a real external database instead of an
// embedded, single-process file store.
func TestPostgresDataSurvivesReconnect(t *testing.T) {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("nerv_test"),
		tcpostgres.WithUsername("nerv"),
		tcpostgres.WithPassword("nerv"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("starting postgres testcontainer: %v", err)
	}
	t.Cleanup(func() { _ = pgContainer.Terminate(ctx) })

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("getting connection string: %v", err)
	}

	first, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore() (first connection) error = %v", err)
	}
	if err := first.Register(Project{ID: "durable-proj", Name: "svc", Language: "go"}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("first.Close() error = %v", err)
	}

	second, err := NewPostgresStore(ctx, dsn)
	if err != nil {
		t.Fatalf("NewPostgresStore() (second connection) error = %v", err)
	}
	defer second.Close()

	got, err := second.Get("durable-proj")
	if err != nil {
		t.Fatalf("Get() on second connection error = %v", err)
	}
	if got.ID != "durable-proj" {
		t.Errorf("Get() = %+v, want ID = durable-proj", got)
	}
}
