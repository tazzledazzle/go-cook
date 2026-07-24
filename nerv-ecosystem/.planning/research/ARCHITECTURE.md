# Architecture Research

**Domain:** Modular-style developer-platform CLI (generator + semver-gated publish + dependency graph + project search), Go, local-first
**Researched:** 2026-07-24
**Confidence:** HIGH (Go ecosystem/tooling claims verified against official docs/pkg.go.dev/GitHub) / MEDIUM (architectural judgment calls specific to this project's scale)

## Standard Architecture

### System Overview

The right shape for this project is a **single Go binary, modular monolith** — one `cobra` command tree over a small set of `internal/` domain packages, all sharing one embedded datastore. There is no network boundary between "generate," "publish," "deps," and "search" — those are Go packages, not services. The only real I/O boundaries are the filesystem (templates, generated projects), an in-process OCI registry (package artifacts), and a single SQLite file (structured metadata + full-text index).

```
┌──────────────────────────────────────────────────────────────────────┐
│                         modular (single binary)                      │
├──────────────────────────────────────────────────────────────────────┤
│  cmd/ — Cobra CLI surface (routing only, zero business logic)        │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐             │
│  │ generate  │ │ publish   │ │ deps      │ │ search    │             │
│  └─────┬─────┘ └─────┬─────┘ └─────┬─────┘ └─────┬─────┘             │
├────────┼─────────────┼─────────────┼─────────────┼───────────────────┤
│        ▼             ▼             ▼             ▼                   │
│  internal/domain packages (cobra-free, unit-testable, TDD'd)         │
│  ┌───────────┐ ┌────────────┐ ┌──────────┐ ┌──────────┐              │
│  │ generate  │ │ publish    │ │ deps     │ │ search   │              │
│  │ (template │ │ (apidiff + │ │ (DAG     │ │ (FTS5    │              │
│  │  render)  │ │  semver    │ │  build + │ │  query)  │              │
│  │           │ │  gate)     │ │  walk)   │ │          │              │
│  └─────┬─────┘ └──────┬─────┘ └────┬─────┘ └────┬─────┘              │
├────────┼──────────────┼────────────┼────────────┼────────────────────┤
│        ▼              ▼            │            │                    │
│  ┌───────────┐  ┌─────────────┐    │            │                    │
│  │ templates │  │ ociregistry │    │            │                    │
│  │  (FS/     │  │ (in-process │    │            │                    │
│  │  embed.FS)│  │  OCI, local │    │            │                    │
│  │           │  │  layout dir)│    │            │                    │
│  └───────────┘  └──────┬──────┘    │            │                    │
│                        └───────────┴────────────┘                    │
│                                    ▼                                 │
│                        ┌─────────────────────────┐                   │
│                        │  store (SQLite, single   │                   │
│                        │  file, FTS5 virtual      │                   │
│                        │  table, WAL mode)        │                   │
│                        │  = system of record for  │                   │
│                        │  projects/packages/      │                   │
│                        │  versions/deps edges      │                   │
│                        └─────────────────────────┘                   │
└──────────────────────────────────────────────────────────────────────┘
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| `cmd/` (CLI layer) | Parse flags/args, call one domain package, format output. No business logic, no direct storage/registry access. | `spf13/cobra`, one subpackage per verb (`cmd/generate`, `cmd/publish`, `cmd/deps`, `cmd/search`), each exporting `NewCommand()`. |
| `internal/generate` | Load a template manifest for a language, parameterize (name/team/language), render to a target directory, register the new project in the store. | `embed.FS` (or on-disk dir for dev iteration) + `text/template`; one manifest file per language (`go.tmpl.yaml`, `java.tmpl.yaml`, `python.tmpl.yaml`) listing files + required params. |
| `internal/publish` | Diff the current package's public API against the last published version, enforce the semver rule (breaking ⇒ major required), push the artifact, and record the new version + publisher/consumer edges. | `golang.org/x/exp/cmd/apidiff`-style diff (invoked as a library, not a subprocess, when possible) for Go targets; a simpler declared-interface diff (e.g. exported function/class signatures) for Java/Python targets in v1. |
| `internal/deps` | Load dependency edges for a package from the store, build an in-memory DAG, walk it to compute consumer blast radius (direct + transitive). | `dominikbraun/graph` (`graph.Directed()`, `graph.PreventCycles()`, `graph.TopologicalSort`/BFS) built fresh per invocation — the store is the persistent graph, this is a query-time materialization. |
| `internal/search` | Query the FTS5 index for project/package metadata, join to structured tables, return build-config references. | `modernc.org/sqlite` (pure Go, no cgo) `CREATE VIRTUAL TABLE ... USING fts5(...)`, or the typed `gosqlite.org/fts` wrapper on the same driver. |
| `internal/store` | Own the SQLite schema, migrations, and a narrow repository interface (`Projects`, `Packages`, `Versions`, `Edges`, `Search`) used by every other domain package. This is the single system of record. | One `*sql.DB` (WAL mode), embedded migrations (`golang-migrate` or hand-rolled numbered `.sql` files via `embed.FS`), repository methods returning plain structs — never `*sql.Rows` leaking past this package. |
| `internal/ociregistry` | Give `publish` (and later `deps`/`search`) a place to push/pull package artifacts without a running daemon or network dependency. | `google/go-containerregistry` `pkg/registry` (`registry.New()`) served over `httptest`-style loopback, **or** `docker/oci`'s `ocilayout` (filesystem-backed OCI Image Layout, zero network) — prefer `ocilayout` for a laptop-first CLI since it needs no listener at all. |
| `internal/templates` | Own template discovery/versioning for the generator; decoupled from `generate` so template format can evolve independently. | Local FS tree (`templates/go/`, `templates/java/`, `templates/python/`) checked into the repo (or an `embed.FS` build), each with a small manifest describing files, params, and post-render hooks. |
| Generated-project observability/CI stubs | Not a runtime component of the CLI at all — they are **template content** (OTel SDK init snippet, Dockerfile, GitHub Actions workflow, Helm chart) emitted by `generate`. The CLI never talks to Prometheus/Grafana/GHA itself. | Static template files rendered like any other scaffolded file. |

## Recommended Project Structure

```
nerv-ecosystem/
├── cmd/
│   └── modular/
│       └── main.go              # imports internal/cli, calls Execute()
├── internal/
│   ├── cli/                     # Cobra routing layer only
│   │   ├── root.go               # rootCmd, global flags, wires subcommands
│   │   ├── generate/command.go   # NewCommand() -> calls internal/generate
│   │   ├── publish/command.go    # NewCommand() -> calls internal/publish
│   │   ├── deps/command.go       # NewCommand() -> calls internal/deps
│   │   └── search/command.go     # NewCommand() -> calls internal/search
│   ├── generate/                 # domain logic, cobra-free, TDD'd
│   │   ├── generate.go
│   │   ├── manifest.go           # per-language template manifest schema
│   │   └── generate_test.go
│   ├── publish/
│   │   ├── publish.go
│   │   ├── semver_gate.go        # breaking-change → major-bump enforcement
│   │   ├── apidiff_go.go         # Go API diff adapter
│   │   └── publish_test.go
│   ├── deps/
│   │   ├── graph.go               # builds DAG from store edges
│   │   ├── blastradius.go
│   │   └── graph_test.go
│   ├── search/
│   │   ├── search.go
│   │   └── search_test.go
│   ├── store/                    # single embedded datastore, owns schema
│   │   ├── store.go               # *sql.DB lifecycle, WAL pragmas
│   │   ├── migrations/            # embed.FS of numbered .sql files
│   │   ├── projects.go
│   │   ├── packages.go
│   │   ├── versions.go
│   │   ├── edges.go
│   │   └── fts.go                 # FTS5 virtual table + sync triggers
│   ├── ociregistry/               # local artifact store, no daemon
│   │   ├── ociregistry.go
│   │   └── ociregistry_test.go
│   └── templates/                 # template discovery, not rendering
│       ├── templates.go
│       └── fs/                    # go/, java/, python/ template trees
├── templates/                     # (or under internal/templates/fs)
│   ├── go/
│   ├── java/
│   └── python/
├── go.mod
└── go.sum
```

### Structure Rationale

- **`cmd/modular/main.go` stays tiny.** It only calls `cli.Execute()`. This keeps the binary entrypoint trivial to test-skip and matches the Cobra-recommended layout for apps that outgrow a single `cmd/` package.
- **`internal/cli/*` is routing-only.** Each verb's Cobra command lives in its own package and calls exactly one domain package function with a typed config struct — never the reverse. This is the load-bearing boundary that keeps `generate`/`publish`/`deps`/`search` testable without spinning up Cobra, and keeps the CLI framework swappable later without touching domain logic.
- **One `internal/store` package, not four.** `generate`, `publish`, `deps`, and `search` all read/write through `store`'s repository interfaces rather than opening their own `*sql.DB` or reaching into SQL directly. This is the single most important boundary for avoiding "unrunnable sprawl" — it is the modern, laptop-scale substitute for the real Nerv system's staged multi-datastore growth (DynamoDB/S3 → RDS → Postgres/SQLite/Mongo), which happened for *organizational* reasons (many independent teams standardizing on different infra) that don't apply to a single-operator playground.
- **`internal/ociregistry` isolates the only place that touches OCI/registry APIs.** `publish` depends on it; `deps`/`search` only depend on `store` (which records refs, not blobs). If a future milestone wants a real remote registry, only this package changes.
- **`internal/templates` is separate from `internal/generate`.** Template *discovery/versioning* (what templates exist, what params they need) is a distinct concern from *rendering* (parameterizing and writing files). Splitting them lets template format evolve (e.g., add a manifest field) without touching the generator's control flow, and mirrors the original design's separate "template registry" data model.
- **Package-per-slice, not package-per-layer.** `internal/generate`, `internal/publish`, `internal/deps`, `internal/search` are vertical, each owning its own logic end-to-end (minus shared storage/registry access). This matches the "vertical MVP slices" build philosophy directly — each slice can be planned, TDD'd, and demoed independently.

## Architectural Patterns

### Pattern 1: Modular monolith with package-level ports-and-adapters

**What:** Domain packages (`generate`, `publish`, `deps`, `search`) depend on small interfaces (`store.ProjectRepo`, `store.PackageRepo`, `ociregistry.Pusher`) rather than concrete types. The concrete SQLite/OCI implementations live in `internal/store` and `internal/ociregistry` and are wired at `main.go`/`cli` composition time.

**When to use:** Any time a component talks to "infrastructure" (DB, registry, filesystem) — which for this project is every domain package. This is what makes each vertical slice TDD-able against fakes/in-memory implementations before the real SQLite/OCI adapters exist.

**Trade-offs:** Slightly more boilerplate (interface + struct) than calling `*sql.DB` directly everywhere; pays for itself immediately once you need `_test.go` files with an in-memory repo, and again later if you ever swap SQLite for Postgres.

**Example:**
```go
// internal/deps/graph.go
type EdgeReader interface {
    ConsumerEdges(ctx context.Context, pkg string) ([]store.Edge, error)
}

type Service struct {
    edges EdgeReader
}

func (s *Service) BlastRadius(ctx context.Context, pkg string) (*Result, error) {
    edges, err := s.edges.ConsumerEdges(ctx, pkg)
    if err != nil {
        return nil, fmt.Errorf("load edges: %w", err)
    }
    g := graph.New(graph.StringHash, graph.Directed(), graph.PreventCycles())
    // ... add vertices/edges from `edges`, then BFS/topo-walk from pkg
}
```

### Pattern 2: Cobra command-registry with cobra-free core

**What:** `cmd/*` packages each export `NewCommand() *cobra.Command`; the root wires them via `AddCommand`. `RunE` closures parse flags into a typed config struct and call straight into `internal/<domain>`, which has zero Cobra/Viper imports.

**When to use:** As soon as there's more than one subcommand (there are four here from day one) — verified as the community-recommended pattern once a CLI grows past a handful of files (Cobra's own docs and the `cobra-viper` skill both converge on this).

**Trade-offs:** One extra indirection per command vs. inlining logic in `RunE`; in exchange, `internal/generate`/`publish`/`deps`/`search` can be unit-tested with plain Go structs and no CLI harness — essential for the TDD-first mandate.

**Example:**
```go
// internal/cli/generate/command.go
func NewCommand() *cobra.Command {
    var cfg generate.Config
    cmd := &cobra.Command{
        Use:   "generate",
        Short: "Scaffold a new service from a language template",
        RunE: func(cmd *cobra.Command, args []string) error {
            return generate.New(store.Default(), templates.Default()).Run(cmd.Context(), cfg)
        },
    }
    cmd.Flags().StringVar(&cfg.Language, "lang", "", "go|java|python")
    cmd.Flags().StringVar(&cfg.Name, "name", "", "service name")
    cmd.Flags().StringVar(&cfg.Team, "team", "", "owning team")
    return cmd
}
```

### Pattern 3: Single embedded datastore as shared system of record

**What:** One SQLite file (`~/.modular/registry.db` or project-local) holds projects, packages, versions, dependency edges, and an FTS5 virtual table kept in sync via triggers or explicit writes on the same transaction as the structured tables. `generate` inserts on scaffold; `publish` inserts versions/edges and updates the index; `deps`/`search` only read.

**When to use:** Whenever multiple "components" need to share structured state but don't need independent scaling/availability — true for all four slices here running on one laptop, sequentially, single-user.

**Trade-offs:** A shared schema means slices are coupled through table contracts (a `publish` schema change can break `search`) — mitigate by keeping schema ownership inside `internal/store` and only exposing typed repository methods, never raw SQL, to the other packages. In exchange you get one file to back up, no cross-service consistency problems, and trivial local demos.

## Data Flow

### Request Flow — `generate`

```
modular generate --lang go --name svc-foo --team platform
    ↓
cli/generate.NewCommand → generate.Service.Run(ctx, cfg)
    ↓
templates.Lookup(cfg.Language) → manifest (files + params)
    ↓
render each template file with (Name, Team, Language) → write to ./svc-foo/
    ↓
store.Projects.Create(ctx, Project{ID, Name, Team, Language, Path})
    ↓
store.Search.Index(ctx, project)   # FTS5 row insert, same transaction
    ↓
stdout: path + summary of files written
```

### Request Flow — `publish`

```
modular publish   (run inside a package directory)
    ↓
cli/publish.NewCommand → publish.Service.Run(ctx, cfg)
    ↓
store.Versions.Latest(ctx, pkgName)  → previous version (or "none")
    ↓
apidiff(previous artifact, current source) → {breaking bool, changes []Change}
    ↓
semver_gate.Check(previous.SemVer, declared.SemVer, breaking) → pass/fail
    ↓ pass                                              ↓ fail
ociregistry.Push(artifact) → digest              return ErrSemverViolation
    ↓                                                    ↓
store.Versions.Create(ctx, Version{pkg, semver, digest})   (no writes; CLI exits non-zero)
store.Edges.Sync(ctx, pkg, declaredDeps)
store.Search.Index(ctx, pkg version)
```

### Request Flow — `deps --graph`

```
modular deps --graph pkg-foo
    ↓
cli/deps.NewCommand → deps.Service.BlastRadius(ctx, "pkg-foo")
    ↓
store.Edges.ConsumerEdges(ctx, "pkg-foo")  →  []Edge{publisher, consumer}
    ↓
build in-memory DAG (dominikbraun/graph) from edges
    ↓
BFS/topo-walk from "pkg-foo" over consumer direction → []ConsumerImpact
    ↓
render table/JSON to stdout
```

### Request Flow — `search`

```
modular search "svc-foo"
    ↓
cli/search.NewCommand → search.Service.Query(ctx, "svc-foo")
    ↓
store.Search.Query(ctx, term)  → FTS5 MATCH query, ranked (bm25)
    ↓
join to Projects/Versions for build_config_ref, language, owning team
    ↓
render result rows to stdout
```

### Key Data Flows

1. **Registration write path (generate → store):** `generate` is the *only* writer of new `Project` rows. Everything downstream (`search`) that needs "does this project exist" reads through `store`, never re-derives it from the filesystem.
2. **Publish write path (publish → ociregistry + store, two writes that must agree):** the artifact push and the metadata/edge write should be sequenced so a failed push never leaves a dangling `Version` row (push first, then write metadata referencing the returned digest; on push failure, no store write happens at all).
3. **Read-only fan-out (store → deps, search):** `deps` and `search` never write; they are pure query layers over `store`, which keeps them trivially safe to add/change without risking `generate`/`publish` correctness.

## Scaling Considerations

| Scale | Architecture Adjustments |
|-------|--------------------------|
| Single laptop, single operator (this project's actual target) | Exactly the design above: one binary, one SQLite file, in-process/`ocilayout` OCI store, FS templates. No adjustments needed. |
| A few operators sharing a demo host / CI runner | SQLite in WAL mode already supports multiple readers + one writer; keep the OCI layout directory and DB file on a shared path. No component split needed yet. |
| Hypothetical "team" scale (out of scope, but worth naming for interview defensibility) | Swap `internal/store`'s SQLite adapter for Postgres behind the *same* repository interfaces; swap `internal/ociregistry`'s `ocilayout` adapter for a real registry (e.g. `zot` or a hosted registry) behind the *same* push/pull interface. `generate`/`publish`/`deps`/`search` domain code does not change — this is the payoff of the ports-and-adapters boundary chosen above. |

### Scaling Priorities

1. **First (theoretical) bottleneck:** SQLite single-writer contention if `publish` is called concurrently from many processes. Not a real concern for a single-operator CLI; if it ever mattered, the fix is WAL mode (already recommended) before reaching for a client-server DB.
2. **Second (theoretical) bottleneck:** FTS5 index growth/query latency at large project counts. SQLite FTS5 comfortably handles tens of thousands of rows on a laptop; not a concern at this project's scale, but worth noting during `search` implementation.

## Anti-Patterns

### Anti-Pattern 1: Splitting generate/publish/deps/search into separate services/binaries with HTTP between them

**What people do:** Give each "component" its own process and expose it over localhost REST/gRPC because the original Nerv/Modular system is described as a "platform."

**Why it's wrong:** For a single-binary, single-operator, laptop-run CLI, this adds serialization, port management, process lifecycle, and partial-failure modes with zero corresponding benefit — the opposite of "modular monolith over microservices sprawl" that this project explicitly wants to avoid.

**Do this instead:** Keep `generate`/`publish`/`deps`/`search` as `internal/` Go packages called in-process, with clean interface boundaries (Pattern 1) so they *could* be split later if a real reason emerges — but don't pay that cost up front.

### Anti-Pattern 2: One datastore per domain concern (mimicking Nerv's real 7-year multi-datastore history)

**What people do:** Stand up separate SQLite files (or Postgres/Mongo/etc.) per component because the source blueprint describes DynamoDB+S3→RDS→Postgres/SQLite/MongoDB evolution.

**Why it's wrong:** That heterogeneity existed because dozens of independent teams at Tableau had already standardized on different infrastructure — an organizational constraint, not a technical one. Reproducing it here just fragments a single-operator project's state across files that all need to stay consistent by hand.

**Do this instead:** One `internal/store` SQLite file as the shared system of record (Pattern 3), with `internal/ociregistry` as the only other stateful component (artifact bytes, not metadata).

### Anti-Pattern 3: Requiring a running registry daemon (Docker/`zot`) for the "local OCI registry"

**What people do:** Shell out to `docker run registry:2` or require a `zot` binary running in the background so `publish`/`deps` have "a registry to talk to."

**Why it's wrong:** Adds an external process dependency and startup-ordering problems to what should be a single `go run`/single-binary demo; contradicts the "runs fully local on a laptop" requirement.

**Do this instead:** Use `google/go-containerregistry`'s `pkg/registry` package in-process (it implements the Docker V2/OCI distribution protocol as an `http.Handler` you can serve over `httptest` or a local Unix socket with zero external processes), or better, `docker/oci`'s `ocilayout` package, which is a filesystem-backed `oci.Interface` with **no network listener at all** — push/pull are just calls to a Go interface backed by a directory.

### Anti-Pattern 4: A plugin/language-adapter framework for 3 languages

**What people do:** Build a generic "language plugin" interface with dynamic loading, config-driven adapters, etc., anticipating the original system's staged growth to a dozen+ languages.

**Why it's wrong:** v1 scope is exactly Go, Java, Python (per project constraints) — a plugin system is speculative generality that adds indirection with no near-term payoff.

**Do this instead:** A simple `Language` enum + one template manifest directory per language. If a 4th language is ever added, it's a new directory and a new enum value, not a new abstraction layer.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Local OCI registry (package artifacts) | In-process, via `google/go-containerregistry` `pkg/registry.New()` served over loopback, or `docker/oci` `ocilayout` (filesystem-backed, no listener) | Prefer `ocilayout` for zero-process, zero-port operation; only reach for the in-process HTTP server if you specifically want to exercise real `docker push`/`crane` compatibility for the demo. |
| Embedded SQLite (structured store + FTS5 search index) | `database/sql` + `modernc.org/sqlite` (pure Go, no cgo, no C compiler required) driver, registered as `"sqlite"` | FTS5 is compiled in by default on this driver — no separate extension load step. Use WAL mode (`?_pragma=journal_mode(WAL)`) for safe concurrent reads. |
| Local filesystem templates | Plain `os`/`io/fs`, optionally `embed.FS` for a single-binary distributable | Start with an on-disk `templates/` tree for fast iteration during TDD; switch to `embed.FS` once template content stabilizes, so `go install` produces a self-contained binary. |
| Go API compatibility check (semver gate, Go targets) | `golang.org/x/exp/apidiff` package (library, not just CLI) — `apidiff.Changes(oldPkg, newPkg)` returns classified compatible/incompatible changes | This is an *approximation* by design (no tool detects behavioral changes) — treat its incompatible-change output as the enforcement signal, matching the original two-layer linter+CI-gate design. For Java/Python in v1, a simpler exported-signature diff is acceptable; flag as a phase needing its own research pass. |
| Dependency graph traversal | `dominikbraun/graph` — `graph.New(graph.StringHash, graph.Directed(), graph.PreventCycles())`, `graph.TopologicalSort`/BFS from the target vertex | Build the in-memory graph fresh from `store` edges on each `deps` invocation; the store, not the graph object, is the durable representation. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `cli/*` ↔ `internal/generate|publish|deps|search` | Direct Go function calls, typed config structs in, typed result/error out | No Cobra/Viper types cross this boundary — keeps domain packages unit-testable without a CLI harness. |
| `internal/generate|publish|deps|search` ↔ `internal/store` | Narrow repository interfaces (`ProjectRepo`, `PackageRepo`, `VersionRepo`, `EdgeRepo`, `SearchIndex`) | Only `internal/store` imports the `database/sql` driver and knows the schema; other packages depend on interfaces they could fake in tests. |
| `internal/publish` ↔ `internal/ociregistry` | A small `Pusher`/`Puller` interface (`Push(ctx, ref, blob) (digest, error)`) | Keeps `publish`'s domain logic (semver gate, apidiff) decoupled from *how* artifacts are stored — swappable later without touching the gate logic. |
| `internal/generate` ↔ `internal/templates` | `templates.Lookup(language) (Manifest, error)` returning a manifest + `io/fs.FS` handle to render from | Template *format* changes stay inside `internal/templates`; `generate`'s render loop only depends on the manifest shape. |

## Sources

- Cobra command organization (modular layout, `NewCommand()` per feature) — https://cobra.dev/docs/how-to-guides/working-with-commands/ (official docs, HIGH confidence)
- Cobra/Viper CLI-vs-core-logic separation skill — https://github.com/spf13/go-skills/blob/main/cobra-viper/SKILL.md (community skill referencing official Cobra maintainer patterns, MEDIUM-HIGH confidence)
- `google/go-containerregistry` `pkg/registry` (in-process OCI/Docker V2 registry server) — https://pkg.go.dev/github.com/google/go-containerregistry/pkg/registry (official pkg.go.dev, HIGH confidence)
- `docker/oci` (`ocilayout`, `ocimem`, `ociserver` — filesystem-backed and in-memory OCI interfaces, no network required) — https://github.com/docker/oci (official GitHub, HIGH confidence)
- Zot registry (single-binary OCI-native registry, referenced as a real-registry escape hatch for future scale) — https://zotregistry.dev/ (official project site, HIGH confidence)
- `golang.org/x/exp/apidiff` (Go API compatibility diff for semver gate) — https://pkg.go.dev/golang.org/x/exp/apidiff and https://go.googlesource.com/exp/+/refs/heads/master/apidiff/README.md (official Go team docs, HIGH confidence)
- `modernc.org/sqlite` (pure-Go, cgo-free SQLite driver with FTS5 compiled in) — https://pkg.go.dev/modernc.org/sqlite (official pkg.go.dev, HIGH confidence); usage pattern verified at https://practicalgobook.net/posts/go-sqlite-no-cgo/ (MEDIUM confidence, community tutorial)
- `dominikbraun/graph` (generic DAG library, topological sort/traversal for dependency blast radius) — https://github.com/dominikbraun/graph and https://pkg.go.dev/github.com/dominikbraun/graph (official repo/pkg.go.dev, HIGH confidence)
- Source blueprint for the original Modular/Nerv system's real component/data-model history (used to identify which historical complexity to *deliberately not* reproduce) — `nerv-ecosystem/README.md` (project-internal document, HIGH confidence as a description of intent, not an external source)

---
*Architecture research for: Modular-style generate/publish/deps/search developer-platform CLI (Go, local-first)*
*Researched: 2026-07-24*
