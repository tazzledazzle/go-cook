// Command nervctl is the Nerv CLI: generate registered, templated
// projects backed by the registry, template engine, dependency
// resolver, CI hook, and metrics packages.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"tazzledazzle/nerv/nerv/internal/api"
	"tazzledazzle/nerv/nerv/internal/cihook"
	"tazzledazzle/nerv/nerv/internal/creation"
	"tazzledazzle/nerv/nerv/internal/depgraph"
	"tazzledazzle/nerv/nerv/internal/directory"
	"tazzledazzle/nerv/nerv/internal/manifest"
	"tazzledazzle/nerv/nerv/internal/metrics"
	"tazzledazzle/nerv/nerv/internal/registries"
	"tazzledazzle/nerv/nerv/internal/registry"
	"tazzledazzle/nerv/nerv/internal/template"
)

// app bundles every wired-up component nervctl commands operate on.
type app struct {
	creation  *creation.Service
	directory *directory.Service
	graph     *depgraph.Graph
	metrics   *metrics.Metrics
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	dataDir := os.Getenv("NERV_DATA_DIR")
	if dataDir == "" {
		dataDir = "./.nerv-data"
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatalf("nervctl: creating data dir %q: %v", dataDir, err)
	}

	a, closeFn, err := buildApp(dataDir)
	if err != nil {
		log.Fatalf("nervctl: %v", err)
	}
	defer closeFn()

	switch os.Args[1] {
	case "search":
		a.cmdSearch(os.Args[2:])
	case "lint":
		a.cmdLint(os.Args[2:])
	case "serve":
		a.cmdServe(os.Args[2:])
	case "new":
		a.cmdNew(os.Args[2:])
	case "list":
		a.cmdList(os.Args[2:])
	case "serve-metrics":
		a.cmdServeMetrics(os.Args[2:])
	default:
		usage()
		os.Exit(1)
	}
}

func usage() {
	fmt.Println(`nervctl — Nerv project generator CLI

Usage:
  nervctl new --lang=<language> --name=<project-name> [--dest=<dir>]
  nervctl list [--lang=<language>]
  nervctl serve [--addr=:8080] [--dest=<dir>]
  nervctl lint --file=<manifest-path> [--check-vulnerabilities=true] [--check-existence=false]
  nervctl serve-metrics [--addr=:9090]`)
}

// buildRegistry builds a Registrar backed by Postgres if NERV_POSTGRES_DSN
// is set, otherwise falls back to the embedded BoltDB store. This is a
// deliberate demonstration of the Registrar interface making storage
// swappable without touching any caller.
func buildRegistry(dataDir string) (registry.Registrar, func(), error) {
	if dsn := os.Getenv("NERV_POSTGRES_DSN"); dsn != "" {
		reg, err := registry.NewPostgresStore(context.Background(), dsn)
		if err != nil {
			return nil, nil, fmt.Errorf("building postgres registry: %w", err)
		}
		log.Println("nervctl: using Postgres-backed registry")
		return reg, func() {
			if err := reg.Close(); err != nil {
				log.Printf("nervctl: closing postgres registry: %v", err)
			}
		}, nil
	}

	reg, err := registry.NewBoltStore(filepath.Join(dataDir, "registry.db"))
	if err != nil {
		return nil, nil, fmt.Errorf("building bolt registry: %w", err)
	}
	return reg, func() {
		if err := reg.Close(); err != nil {
			log.Printf("nervctl: closing bolt registry: %v", err)
		}
	}, nil
}

// buildApp wires every package into a working app instance, backed by
// two BoltDB files under dataDir. Returns a close function that shuts
// down every store cleanly.
func buildApp(dataDir string) (*app, func(), error) {
	reg, closeReg, err := buildRegistry(dataDir)
	if err != nil {
		return nil, nil, err
	}

	pointers, err := template.NewBoltPointerStore(filepath.Join(dataDir, "pointers.db"))
	if err != nil {
		closeReg()
		return nil, nil, fmt.Errorf("building pointer store: %w", err)
	}

	catalog := template.NewMemCatalog()
	seedBuiltinTemplates(catalog)
	if err := pointers.SetCurrent("service", "v1"); err != nil {
		_ = reg.Close()
		_ = pointers.Close()
		return nil, nil, fmt.Errorf("seeding template pointer: %w", err)
	}

	engine := template.NewEngine(catalog, pointers)
	resolver := depgraph.NewResolver(depgraph.NewEnforcer(depgraph.Policy{}), depgraph.NewCache())
	hook := cihook.NewStubHook()
	m := metrics.New()

	dirSvc, err := directory.New(reg)
	if err != nil {
		closeReg()
		_ = pointers.Close()
		return nil, nil, fmt.Errorf("building directory service: %w", err)
	}

	creationSvc := &creation.Service{
		Engine:    engine,
		Resolver:  resolver,
		Hook:      hook,
		Directory: dirSvc,
		Metrics:   m,
		DestRoot:  "./generated",
	}

	a := &app{creation: creationSvc, directory: dirSvc, graph: depgraph.NewGraph(), metrics: m}

	closeFn := func() {
		closeReg()
		if err := pointers.Close(); err != nil {
			log.Printf("nervctl: closing pointer store: %v", err)
		}
	}

	return a, closeFn, nil
}

// seedBuiltinTemplates registers a minimal "service" template version so
// `nervctl new` works out of the box.
func seedBuiltinTemplates(catalog *template.MemCatalog) {
	files := template.Files{
		"main.go":   mainGoTemplate,
		"README.md": readmeTemplate,
	}
	if err := catalog.RegisterVersion("service", "v1", files); err != nil {
		log.Printf("nervctl: seeding template: %v", err)
	}
}

// Declared as package-level backtick strings (column 0) rather than
// indented inline literals, so the leading whitespace of each source
// line doesn't leak into the rendered output file.
const mainGoTemplate = `package main

func main() {
	println("{{.ServiceName}} starting up")
}
`

const readmeTemplate = `# {{.ServiceName}}

Generated by Nerv.
`

func (a *app) cmdServe(args []string) {
	fs := newFlagSet("serve")
	addr := fs.String("addr", ":8080", "listen address")
	mustParse(fs, args)

	srv := &api.Server{
		Creation:  a.creation,
		Directory: a.directory,
		Graph:     a.graph,
	}

	fmt.Printf("Serving Nerv API on %s (POST/GET /projects, GET /search, GET /metrics not yet re-wired, GET /healthz)\n", *addr)
	if err := http.ListenAndServe(*addr, srv.Router()); err != nil {
		log.Fatalf("nervctl serve: %v", err)
	}
}

func (a *app) cmdNew(args []string) {
	fs := newFlagSet("new")
	lang := fs.String("lang", "", "project language (required)")
	name := fs.String("name", "", "project name (required)")
	depsFlag := fs.String("deps", "", `comma-separated deps as name@version`)
	mustParse(fs, args)

	if *lang == "" || *name == "" {
		log.Fatal("nervctl new: --lang and --name are required")
	}

	deps, err := parseDeps(*depsFlag)
	if err != nil {
		log.Fatalf("nervctl new: parsing --deps: %v", err)
	}

	result, err := a.creation.Create(creation.Request{Name: *name, Language: *lang, Deps: deps})
	if err != nil {
		log.Fatalf("nervctl new: %v", err)
	}

	if len(result.ResolvedDeps) > 0 {
		fmt.Printf("Resolved %d dependencies (cache hit: %v)\n", len(result.ResolvedDeps), result.DepsCacheHit)
	}
	fmt.Printf("Generated project %q (id=%s)\n  path:     %s\n  pipeline: %s\n",
		*name, result.Project.ID, result.Project.Path, result.PipelineConfig)
}

func (a *app) cmdLint(args []string) {
	fs := newFlagSet("lint")
	file := fs.String("file", "nerv-deps.json", "path to the dependency manifest")
	checkVuln := fs.Bool("check-vulnerabilities", true, "check pinned versions against the known-vulnerable list (no network required)")
	checkExistence := fs.Bool("check-existence", false, "verify pinned versions actually exist in PyPI/npm (requires network)")
	mustParse(fs, args)

	m, err := manifest.Load(*file)
	if err != nil {
		log.Fatalf("nervctl lint: %v", err)
	}

	policy := depgraph.Policy{}
	if *checkVuln {
		policy.Checker = depgraph.NewSTARIncidentVulnerabilityList()
	}
	if *checkExistence {
		policy.ExistenceChecker = registries.NewMultiIndex(
			registries.NamedIndex{Name: "pypi", Index: registries.NewPyPIClient()},
			registries.NamedIndex{Name: "npm", Index: registries.NewNpmClient()},
		)
	}

	enforcer := depgraph.NewEnforcer(policy)
	deps := m.ToDependencies()

	fmt.Printf("Linting %q (%d dependencies)\n", m.Name, len(deps))

	var failed bool
	for _, dep := range deps {
		if err := enforcer.Validate(dep); err != nil {
			fmt.Printf("  FAIL  %s@%s: %v\n", dep.Name, dep.Constraint, err)
			failed = true
			continue
		}
		fmt.Printf("  OK    %s@%s\n", dep.Name, dep.Constraint)
	}

	if failed {
		fmt.Println("\nlint failed: one or more dependencies violate policy")
		os.Exit(1)
	}
	fmt.Println("\nlint passed: all dependencies comply with policy")
}

// parseDeps parses a comma-separated "name@version,name@version" string
// into depgraph.Dependency values. Empty input yields an empty (not nil)
// slice-safe result.
func parseDeps(raw string) ([]depgraph.Dependency, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	deps := make([]depgraph.Dependency, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		nameVer := strings.SplitN(p, "@", 2)
		if len(nameVer) != 2 || nameVer[0] == "" || nameVer[1] == "" {
			return nil, fmt.Errorf("invalid dep %q, want format name@version", p)
		}
		deps = append(deps, depgraph.Dependency{Name: nameVer[0], Constraint: nameVer[1]})
	}
	return deps, nil
}

func (a *app) cmdList(args []string) {
	fs := newFlagSet("list")
	lang := fs.String("lang", "", "filter by language (optional)")
	mustParse(fs, args)

	projects, err := a.directory.List(*lang)
	if err != nil {
		log.Fatalf("nervctl list: %v", err)
	}

	out, err := json.MarshalIndent(projects, "", "  ")
	if err != nil {
		log.Fatalf("nervctl list: marshaling: %v", err)
	}
	fmt.Println(string(out))
}

func (a *app) cmdServeMetrics(args []string) {
	fs := newFlagSet("serve-metrics")
	addr := fs.String("addr", ":9090", "listen address")
	mustParse(fs, args)

	http.Handle("/metrics", a.metrics.Handler())
	fmt.Printf("Serving metrics on %s/metrics\n", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatalf("nervctl serve-metrics: %v", err)
	}
}

func (a *app) cmdSearch(args []string) {
	fs := newFlagSet("search")
	query := fs.String("q", "", "search query (required)")
	mustParse(fs, args)

	if *query == "" {
		log.Fatal("nervctl search: --q is required")
	}

	results, err := a.directory.Search(*query)
	if err != nil {
		log.Fatalf("nervctl search: %v", err)
	}

	out, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(out))

	links := a.directory.CodeSearchLinks(*query)
	if len(links) > 0 {
		fmt.Println("\nCode search:")
		for name, url := range links {
			fmt.Printf("  %s: %s\n", name, url)
		}
	}
}
