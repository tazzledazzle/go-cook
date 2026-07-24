# Phase 1: Platform Foundation - Research

**Researched:** 2026-07-24
**Domain:** Go single-binary CLI skeleton (Cobra) + embedded pure-Go SQLite store (WAL + FTS5) + TDD scaffolding
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

No CONTEXT.md exists for this phase (`/gsd-discuss-phase` has not been run) — all decisions below are at Claude's discretion, constrained only by ROADMAP.md's stated success criteria, PROJECT.md's constraints, and the project-level research already synthesized in `.planning/research/{SUMMARY,STACK,ARCHITECTURE}.md`.

### Locked (from project-level research / ROADMAP, treat as non-negotiable for this phase)
- Go 1.25+ module floor, built with the latest available 1.26.x toolchain
- `spf13/cobra` for the CLI command tree (no urfave/cli, no Kong)
- `modernc.org/sqlite` (pure Go, no CGO) as the sole SQLite driver — never `mattn/go-sqlite3`
- One embedded SQLite file is the single system of record; WAL mode + FTS5 virtual table must exist from the first `go run`/binary invocation
- `cmd/` contains Cobra wiring only; `internal/store` is the *only* package that imports `database/sql` / knows the schema
- Ports-and-adapters (narrow repository interfaces) for domain packages — required starting in Phase 2 (`generate`/`publish`/`deps`/`search`); Phase 1 has no feature-domain package yet, only `internal/store`, so this pattern applies to `internal/store`'s own exposed interface, not to a second Phase-1 domain package
- TDD mandatory (`workflow.tdd_mode: true`): a failing test must exist before the corresponding production code, verified under `go test -race`
- No required cloud accounts, no always-on daemons for the core path

### Claude's Discretion (this research makes recommendations)
- Exact `internal/store` schema shipped in Phase 1 (which tables beyond the FTS5 table)
- Migration mechanism (hand-rolled `embed.FS` vs. `golang-migrate`)
- Default store file location (project-local vs. user-home-scoped)
- Whether Viper is wired in Phase 1 or deferred until real config exists
- Which concrete CLI command(s) ship in Phase 1 to demonstrate the store end-to-end
- Whether a CI workflow is added in Phase 1 or deferred

### Deferred Ideas (OUT OF SCOPE for Phase 1)
- All `generate`/`publish`/`deps`/`search` domain logic (Phases 2–5)
- `versions` and `edges` tables (added incrementally in Phase 2/Phase 3 per ARCHITECTURE.md's explicit "built incrementally" guidance — do not create them in Phase 1)
- OCI registry, apidiff, semver gate, OTel — none of this touches Phase 1
</user_constraints>

<architectural_responsibility_map>
## Architectural Responsibility Map

Single-tier application — this is a local CLI, not a client/server system. Every capability below lives in the same OS process on the operator's laptop; there is no browser, no frontend server, no CDN, and no separately-deployed API tier.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| CLI argument/flag parsing, help/usage | CLI process (`cmd/`) | — | Cobra owns command routing; no other tier involved |
| Store lifecycle (open, migrate, WAL pragma) | CLI process (`internal/store`) | Database/Storage (SQLite file on local disk) | The SQLite file is the only persistent "storage tier"; it is embedded in-process, not a separate service |
| FTS5 search table bootstrap | Database/Storage (SQLite) | — | FTS5 is a SQLite virtual table feature, created via the same migration path as structured tables |
| TDD test harness | CLI process (Go test binary) | — | `go test -race` runs in-process against a temp SQLite file; no external test infra needed |

**All capabilities reside in one local Go process** — no tier-misassignment risk exists for this phase (it reappears starting Phase 2, where `internal/ociregistry` introduces the first true external-integration boundary).
</architectural_responsibility_map>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| PLAT-01 | Operator can install/run a single Go binary CLI (`modular` or project-chosen name) with no required cloud accounts or always-on daemons for the core path | Standard Stack (Cobra-only CLI, no network dependency); Recommended Project Structure; Code Example "Root Command + main.go" |
| PLAT-02 | Platform persists projects, versions, dependency edges, and search index in one embedded SQLite store (WAL + FTS5) | Standard Stack (`modernc.org/sqlite`); Code Examples "WAL DSN" and "FTS5 Bootstrap Migration"; Architecture Patterns 3; Common Pitfall 1 |
| PLAT-03 | Every vertical slice is implemented test-first; race detector and table-driven tests cover domain packages | Validation Architecture section; Architecture Pattern 4 (RED-GREEN store bootstrap); Common Pitfall 4 |
</phase_requirements>

<research_summary>
## Summary

Phase 1 is a **walking skeleton**, not a feature phase: it must prove — with one real CLI invocation and one real SQLite read/write — that the four pillars built in Phases 2–5 have a foundation to stand on. Nothing here is architecturally novel; Cobra command-tree wiring, `embed.FS`-backed migrations, and SQLite WAL+FTS5 bootstrapping are all well-documented, conventional Go patterns (confirmed HIGH confidence against official docs and the project's own prior research in `STACK.md`/`ARCHITECTURE.md`). The only genuine risk in this phase is a well-documented **DSN footgun**: `modernc.org/sqlite` silently ignores the `mattn/go-sqlite3`-style `?_journal_mode=WAL` query parameter that most tutorials show, and requires the driver-specific `?_pragma=journal_mode(WAL)` syntax instead — get this wrong and the store silently runs in default rollback-journal mode with no error, which is exactly the kind of assumption-not-verified bug PLAT-03's TDD mandate exists to catch.

The recommended shape: a top-level `main.go` that only calls `cmd.Execute()`, a `cmd/` package holding Cobra command definitions only (per the locked package-layout decision), and one `internal/store` package that owns the SQLite connection, a hand-rolled `embed.FS` migration runner, and a minimal Phase-1 schema — a `projects` table (even though nothing writes to it until Phase 2's `generate`) plus its FTS5-backed search table, wired together with `AFTER INSERT/UPDATE/DELETE` triggers so the index never needs manual re-sync code. One CLI command (recommended: `modular status`) exercises the full path — open store, verify WAL mode, verify the FTS5 table exists — giving the operator (and the test suite) an observable, demoable proof that PLAT-01 and PLAT-02 are both true. `internal/store`'s own test suite, run against `t.TempDir()`-backed real SQLite files under `go test -race`, is what proves PLAT-03: write the failing test (assert `Open()` returns a store, `PRAGMA journal_mode` is `wal`, and `projects`/`projects_fts` exist in `sqlite_master`) before writing `store.Open()`.

**Primary recommendation:** `main.go` → `cmd/` (Cobra-only) → `internal/store` (sole `database/sql` owner) → one hand-rolled `embed.FS` migration creating `projects` + an external-content FTS5 table with sync triggers, opened via the modernc-specific `?_pragma=journal_mode(WAL)` DSN syntax with `SetMaxOpenConns(1)`. Ship a `status` command that proves the whole chain end-to-end, and TDD every line of `internal/store` against real temp-file SQLite databases under `-race`.
</research_summary>

## Project Constraints (from .cursor/rules/)

`.cursor/rules/` does not exist in this repository (checked at `/Users/terenceschumacher/dev/july-portfolio-projects/go-cook/nerv-ecosystem/.cursor/rules` and repo root) — no additional directives beyond what's already captured in `CLAUDE.md` (which itself mirrors `PROJECT.md` + `research/STACK.md`, already reflected in Locked constraints above). `CLAUDE.md`'s `GSD:workflow-start` block requires all file-changing work to go through a GSD entry point (`/gsd-execute-phase` etc.) rather than direct edits — this is a process directive for the *executor*, not a research finding, but the planner should be aware plans will run under `/gsd-execute-phase`.

<standard_stack>
## Standard Stack

All versions below are re-verified in this session against the official Go module proxy (`proxy.golang.org`), not carried over unchecked from the project-level `STACK.md` — dates confirmed current as of 2026-07-24.

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go | module floor `go 1.25`, build toolchain `1.26.2`+ [VERIFIED: local `go version` = go1.26.2 darwin/arm64] | Language/runtime | Matches the project-locked floor; local dev machine already exceeds it |
| github.com/spf13/cobra | v1.10.2 (2025-12-03) [VERIFIED: proxy.golang.org] | CLI command tree | De facto standard for kubectl/docker/gh-style CLIs; locked by project research |
| modernc.org/sqlite | v1.54.0 (2026-07-15) [VERIFIED: proxy.golang.org; source gitlab.com/cznic/sqlite] | Pure-Go, CGO-free SQLite driver with FTS5 compiled in | No C toolchain needed; trivial cross-compilation; safe under `go test -race`; locked by project research |
| github.com/stretchr/testify | v1.11.1 (2025-08-27) [VERIFIED: proxy.golang.org] | Test assertions (`require`/`assert`) | Reduces boilerplate in table-driven tests without fighting Go idioms |
| github.com/google/go-cmp | v0.7.0 (2025-01-14) [VERIFIED: proxy.golang.org] | Deep-equality diffs in tests | Better failure output than `reflect.DeepEqual` for struct/slice comparisons (e.g. migration result rows) |

### Supporting (not required by Phase 1, but locked project-wide — see discretion notes)

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/spf13/viper | v1.21.0 (2025-09-08) [VERIFIED: proxy.golang.org] | Layered CLI config | **Recommendation: defer to Phase 2.** Phase 1 has no real config surface (no registry URL, no template root) to layer — wiring Viper now with nothing to configure adds a dependency with no behavior to test. Introduce it when `generate`/`publish` need their first real setting. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled `embed.FS` migration runner | `golang-migrate/migrate/v4` with its `database/sqlite` driver (confirmed to support `modernc.org/sqlite`, not just `mattn/go-sqlite3`) | golang-migrate is a real, well-tested option and *is* CGO-free-compatible — but it's a new dependency not in the project's locked `STACK.md`, and Phase 1 needs exactly one migration file. A ~40-line hand-rolled runner (numbered `.sql` files in `embed.FS` + a `schema_migrations` table) is simpler to TDD and keeps the walking skeleton's dependency count minimal. Revisit golang-migrate if Phase 2/3 migrations grow complex (e.g. need `down` migrations). |
| `~/.modular/registry.db` (user-home-scoped default path) | Project-local `./.nerv/registry.db` | Project-local fragments the registry per working directory, which breaks the whole point of `deps`/`search` (Phase 4/5) needing one shared store visible regardless of which generated-project directory the operator is standing in. User-home-scoped (overridable via an env var) is what makes `modular deps --graph` work after `cd`-ing into a different generated service. |
| FTS5 external-content table + sync triggers | FTS5 contentless or "in-table" (content stored twice) | External-content (`content='projects', content_rowid='id'`) avoids storing text twice and is the documented SQLite pattern for "structured table + searchable index of the same data" — see Code Examples. |

**Installation:**
```bash
cd nerv-ecosystem
go mod init nerv-ecosystem   # or a github.com/<user>/nerv-ecosystem path — see Open Questions
go get github.com/spf13/cobra@v1.10.2
go get modernc.org/sqlite@v1.54.0
go get github.com/stretchr/testify@v1.11.1
go get github.com/google/go-cmp@v0.7.0
```
</standard_stack>

## Package Legitimacy Audit

| Package | Registry | Age | Downloads | Source Repo | slopcheck | Disposition |
|---------|----------|-----|-----------|-------------|-----------|-------------|
| github.com/spf13/cobra | go | Years (est. 2013+); latest tag 2025-12-03 | Very high (kubectl/docker/gh ecosystem) | github.com/spf13/cobra | [OK] | Approved |
| github.com/spf13/viper | go | Years (est. 2014+); latest tag 2025-09-08 | Very high | github.com/spf13/viper | [OK] | Approved (deferred to Phase 2, see Discretion) |
| github.com/google/go-cmp | go | Years; latest tag 2025-01-14 | Very high (Google-maintained) | github.com/google/go-cmp | [OK] | Approved |
| github.com/stretchr/testify | go | Years; latest tag 2025-08-27 | Very high | github.com/stretchr/testify | [OK] | Approved |
| modernc.org/sqlite | go | **Flagged by slopcheck as "9 days old"** — this is a false positive: it measures the *latest release* date (v1.54.0, 2026-07-15, i.e. 9 days before this research), not package age. Verified via `proxy.golang.org/modernc.org/sqlite/@v/list`: version history goes back to v1.1.0, spanning 50+ tagged releases; source repo is `gitlab.com/cznic/sqlite` (slopcheck's "no source repo" flag is a tooling limitation — it doesn't check GitLab). | High (widely used pure-Go SQLite driver; transitively pulled by golangci-lint's own toolchain research per project STACK.md) | gitlab.com/cznic/sqlite [VERIFIED] | [SUS] (false positive, overridden with registry evidence) | **Approved** — see verification note above; this is the correct choice per project-locked stack, not a hallucinated/suspicious package |

**Packages removed due to slopcheck [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** `modernc.org/sqlite` — flagged only due to slopcheck's release-date heuristic and its GitHub-only source-repo check; independently verified via the official Go module proxy (50+ historical versions since v1.1.0, canonical GitLab source repo, and exact version/date match against the project's own `STACK.md`/`PROJECT.md` research). No `checkpoint:human-verify` needed — this is a documented tooling limitation, not a real risk signal. Planner may proceed without gating this install.

<architecture_patterns>
## Architecture Patterns

### System Architecture Diagram

```
operator's shell
      │
      │  ./modular status   (or any other subcommand)
      ▼
┌─────────────────────────────────────────────────────────┐
│  main.go  →  cmd.Execute()                               │
│  (Cobra root command dispatch — routing only)             │
└───────────────────────┬───────────────────────────────────┘
                         │  cmd/status.go RunE calls straight
                         │  into the store package (no domain
                         │  layer exists yet in Phase 1)
                         ▼
┌─────────────────────────────────────────────────────────┐
│  internal/store                                           │
│                                                             │
│  Open(path)                                                │
│    ├─▶ os.MkdirAll(dir, 0700)         [create store dir]  │
│    ├─▶ sql.Open("sqlite", dsn)        [WAL + busy_timeout]│
│    ├─▶ db.SetMaxOpenConns(1)          [serialize writers] │
│    └─▶ runMigrations(db, embed.FS)    [idempotent, once]  │
│           │                                                │
│           ▼                                                │
│      ┌─────────────────────────────┐                      │
│      │ migrations/0001_init.sql      │                      │
│      │  CREATE TABLE projects        │                      │
│      │  CREATE VIRTUAL TABLE         │                      │
│      │    projects_fts USING fts5    │                      │
│      │  CREATE TRIGGER (AI/AU/AD)    │  ← keeps fts5 synced │
│      └─────────────────────────────┘                      │
└───────────────────────┬───────────────────────────────────┘
                         │  Info(ctx) reads PRAGMA + sqlite_master
                         ▼
              one SQLite file on disk (WAL mode:
              main .db file + -wal + -shm sidecar files)
                         │
                         ▼
              stdout: store path, journal_mode=wal,
              fts5 table present ✓
```

### Recommended Project Structure

```
nerv-ecosystem/
├── main.go                    # package main; only calls cmd.Execute()
├── cmd/                       # Cobra wiring ONLY — no business logic (locked layout)
│   ├── root.go                # rootCmd, Execute(), global --store-path flag
│   └── status.go               # `modular status` — proves store bootstrap end-to-end
├── internal/
│   └── store/                 # sole package that imports database/sql
│       ├── store.go            # Open/Close, DSN construction, *sql.DB lifecycle
│       ├── migrate.go          # embed.FS migration runner (schema_migrations table)
│       ├── migrations/
│       │   ├── 0001_init.sql   # projects table + projects_fts + sync triggers
│       │   └── migrations.go   # //go:embed *.sql
│       ├── store_test.go       # table-driven, -race, uses t.TempDir()
│       └── migrate_test.go     # idempotency + ordering tests
├── go.mod
├── go.sum
└── .golangci.yml               # v2 schema, run.go: '1.25'
```

### Pattern 1: `main.go` stays trivial; `cmd/` is Cobra-only

**What:** `main.go` contains nothing but a call into `cmd.Execute()` and the top-level error-to-exit-code translation. Every Cobra command lives in the `cmd` package; no command's `RunE` contains business logic beyond parsing flags and calling into `internal/store`.

**When to use:** From the first commit — this is the locked package-layout decision, and retrofitting it later (after `RunE` closures accumulate logic) is exactly the kind of technical debt this project's own research explicitly warns against.

**Example:**
```go
// main.go
package main

import (
	"fmt"
	"os"

	"nerv-ecosystem/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```
```go
// cmd/root.go
package cmd

import "github.com/spf13/cobra"

var storePath string

var rootCmd = &cobra.Command{
	Use:   "modular",
	Short: "Nerv Ecosystem platform CLI",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&storePath, "store-path", "", "override the default store location")
}
```
Source: pattern matches `cobra-cli init`'s own default scaffold (`main.go` at module root + `cmd/root.go` package `cmd`) and Cobra's official "Create rootCmd" guide (cobra.dev/docs/how-to-guides/working-with-commands/).

### Pattern 2: WAL mode via the driver-specific pragma DSN, not the CGO-driver query string

**What:** `modernc.org/sqlite` requires `?_pragma=journal_mode(WAL)` (its own extension syntax), not the `mattn/go-sqlite3`-style `?_journal_mode=WAL` that most StackOverflow/tutorial snippets show. Getting this wrong produces **no error** — the driver just silently ignores the unrecognized parameter and runs in the default rollback-journal mode.

**When to use:** Every `sql.Open` call against the store, from the very first commit.

**Example:**
```go
// internal/store/store.go
package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create store dir: %w", err)
	}

	dsn := fmt.Sprintf(
		"file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
		path,
	)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open store: %w", err)
	}
	// SQLite serializes writes; a single pooled connection avoids
	// write-write lock errors that busy_timeout alone can't resolve.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := runMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate store: %w", err)
	}
	return &Store{db: db, path: path}, nil
}
```
Source: modernc.org/sqlite DSN pragma syntax [CITED: multiple independently-corroborated 2026-dated Go projects switching from the CGO-style DSN after silent WAL failures — pgEdge/ai-dba-workbench commit c9e6ee9, MaorBril/clauder PR #24, BorisYaoA/cagent `sqliteutil/sqlite.go`; all three converge on identical `_pragma=name(value)` syntax and the `SetMaxOpenConns(1)` mitigation].

### Pattern 3: FTS5 as an external-content table synced by triggers, not app-code re-indexing

**What:** Create the FTS5 virtual table with `content='projects', content_rowid='id'` so it stores no duplicate text, and let three `AFTER INSERT/UPDATE/DELETE` triggers on `projects` keep it in sync automatically — the application code that inserts a project never has to remember to also update the search index.

**When to use:** The moment any table needs to be both queryable structurally and full-text searchable — true for `projects` from Phase 1 onward.

**Example:**
```sql
-- internal/store/migrations/0001_init.sql
CREATE TABLE IF NOT EXISTS projects (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    team       TEXT NOT NULL,
    language   TEXT NOT NULL,
    path       TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- It is an error to add column types/constraints/PRIMARY KEY inside
-- a CREATE VIRTUAL TABLE ... USING fts5(...) statement (see fts5.html §4).
CREATE VIRTUAL TABLE IF NOT EXISTS projects_fts USING fts5(
    name, team, language,
    content='projects',
    content_rowid='id'
);

CREATE TRIGGER IF NOT EXISTS projects_ai AFTER INSERT ON projects BEGIN
    INSERT INTO projects_fts(rowid, name, team, language)
    VALUES (new.id, new.name, new.team, new.language);
END;

CREATE TRIGGER IF NOT EXISTS projects_ad AFTER DELETE ON projects BEGIN
    INSERT INTO projects_fts(projects_fts, rowid, name, team, language)
    VALUES ('delete', old.id, old.name, old.team, old.language);
END;

CREATE TRIGGER IF NOT EXISTS projects_au AFTER UPDATE ON projects BEGIN
    INSERT INTO projects_fts(projects_fts, rowid, name, team, language)
    VALUES ('delete', old.id, old.name, old.team, old.language);
    INSERT INTO projects_fts(rowid, name, team, language)
    VALUES (new.id, new.name, new.team, new.language);
END;
```
Source: SQLite official FTS5 documentation, "External Content Tables" (sqlite.org/fts5.html §4/§6) [CITED — full page fetched this session]; syntax cross-checked against `pkg.go.dev/gosqlite.org/fts` and `sqlite.org/fts5.html`'s own `CREATE VIRTUAL TABLE email USING fts5(sender, title, body)` canonical example.

**Note for the planner:** Only `projects` + `projects_fts` ship in Phase 1's migration. `versions` and `edges` tables are explicitly deferred to Phase 2/3 per the project's own `ARCHITECTURE.md` ("`internal/store` is not a separate phase — built incrementally as each vertical slice needs them"). Do not create them now.

### Pattern 4: RED before GREEN for the store package itself

**What:** Because `internal/store` *is* Phase 1's only domain package, PLAT-03's "failing test before production code" mandate applies directly to it: write `store_test.go`'s assertions (temp-dir `Open()` succeeds, `PRAGMA journal_mode` returns `wal`, `sqlite_master` contains `projects` and `projects_fts`) against a `store.Open` function that doesn't exist yet (or returns a stub error), watch it fail, then implement `Open`/`runMigrations` until it's green.

**When to use:** Every task in this phase's plan — this is the mechanism, not a one-time setup step.

**Example:**
```go
// internal/store/store_test.go
package store_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"nerv-ecosystem/internal/store"
)

func TestOpen_CreatesWALModeStoreWithFTS5Table(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "fresh database file"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dbPath := filepath.Join(t.TempDir(), "registry.db")
			st, err := store.Open(dbPath)
			require.NoError(t, err)
			defer st.Close()

			mode, err := st.JournalMode(context.Background())
			require.NoError(t, err)
			require.Equal(t, "wal", mode)

			tables, err := st.SchemaObjects(context.Background())
			require.NoError(t, err)
			require.Contains(t, tables, "projects")
			require.Contains(t, tables, "projects_fts")
		})
	}
}
```

### Anti-Patterns to Avoid
- **Opening the store with the `mattn/go-sqlite3`-style DSN (`?_journal_mode=WAL`) copy-pasted from a generic tutorial:** silently no-ops under `modernc.org/sqlite`; always use `?_pragma=journal_mode(WAL)`.
- **Business logic inside a `cmd/*.go` `RunE` closure:** breaks the locked `cmd/`-is-routing-only layout and makes the logic untestable without a Cobra harness.
- **A second package reaching into `database/sql` directly:** violates "`internal/store` is the sole DB package" — every read/write from a future domain package must go through `store`'s exported methods.
- **Creating `versions`/`edges` tables "while we're at it":** explicitly out of scope for Phase 1 per the project's own incremental-schema architecture decision.
</architecture_patterns>

<dont_hand_roll>
## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|--------------|-----|
| WAL/busy-timeout/foreign-key pragma configuration | A custom query-string builder or post-`Open` `Exec("PRAGMA ...")` calls | The documented `_pragma=name(value)` DSN syntax, applied once at `sql.Open` time | Per-connection pragmas (like `foreign_keys`) must apply to every pooled connection; setting them in the DSN guarantees that even if the pool ever grows beyond 1 connection later |
| Keeping a search index in sync with its source table | Manual "insert into projects, then remember to also insert into the fts table" call sites scattered across `generate`/`publish` | SQLite `AFTER INSERT/UPDATE/DELETE` triggers on the FTS5 external-content table | Triggers make the sync atomic with the write transaction and impossible to forget in a future call site |
| Schema versioning / migration ordering | A bespoke `if !tableExists("projects") { ... }` conditional sprinkled through `store.go` | A tiny embed.FS-based numbered-migration runner with a `schema_migrations` tracking table (or `golang-migrate` if it grows past ~3-4 migrations) | Ad hoc existence checks don't compose once Phase 2/3 add `versions`/`edges` migrations; a single ordered runner does |
| CLI flag parsing, help text, shell completion | Hand-rolled `os.Args` parsing | Cobra + pflag (already locked) | Solved problem; also gives free shell-completion generation later |

**Key insight:** everything hand-rollable in this phase (pragma strings, index sync, migration ordering) has a single documented "correct" SQLite/Cobra idiom — the risk in Phase 1 isn't missing library coverage, it's copy-pasting the *wrong dialect's* idiom (CGO-driver DSN syntax, or a plain `CREATE TABLE`-shaped FTS5 statement) and getting no error to signal the mistake.
</dont_hand_roll>

<common_pitfalls>
## Common Pitfalls

### Pitfall 1: modernc.org/sqlite silently ignores the wrong DSN pragma syntax
**What goes wrong:** Code opens the store with `?_journal_mode=WAL&_busy_timeout=5000` (the `mattn/go-sqlite3` convention). No error is raised at `sql.Open` or at runtime — the database simply stays in default `delete` (rollback-journal) mode with `busy_timeout=0`.
**Why it happens:** Nearly every SQLite-in-Go tutorial found via search assumes the CGO driver's query-parameter convention; `modernc.org/sqlite` deliberately uses a different, driver-specific `_pragma=name(value)` syntax and does not validate/reject unrecognized parameters.
**How to avoid:** Use `?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)`. Assert the effective mode in a test: `SELECT` the result of `PRAGMA journal_mode` and require it equals `"wal"` — don't just assert `Open()` returned no error.
**Warning signs:** `-wal`/`-shm` sidecar files never appear next to the `.db` file on disk; concurrent reads block during a write when they shouldn't.

### Pitfall 2: Multiple pooled connections defeat SQLite's single-writer model even in WAL mode
**What goes wrong:** `database/sql`'s default connection pool opens more than one connection to the SQLite file; two connections both attempt to upgrade from a read to a write transaction and deadlock or return `SQLITE_BUSY`, even with a `busy_timeout` pragma set.
**Why it happens:** SQLite allows many concurrent readers in WAL mode but still serializes writers at the file level; `busy_timeout` alone only delays the error, it doesn't prevent pool-level write/write contention.
**How to avoid:** Call `db.SetMaxOpenConns(1)` (and `SetMaxIdleConns(1)`) right after `sql.Open`, matching the pattern in every corroborating source found this session.
**Warning signs:** Intermittent `"database is locked"` errors that only appear under `-race` or when tests run in parallel (`t.Parallel()`), not in single-threaded runs.

### Pitfall 3: Writing an FTS5 `CREATE VIRTUAL TABLE` statement like a normal `CREATE TABLE`
**What goes wrong:** Adding a column type, `NOT NULL`, or a `PRIMARY KEY` clause inside the `USING fts5(...)` column list raises a SQL error at migration time.
**Why it happens:** FTS5's virtual-table column list looks like ordinary SQL column syntax but is a different grammar — "It is an error to add types, constraints or PRIMARY KEY declarations to a CREATE VIRTUAL TABLE statement" (sqlite.org/fts5.html §4).
**How to avoid:** Keep FTS5 column lists to bare column names (optionally `UNINDEXED`) plus `content=`/`content_rowid=`/`tokenize=` configuration options only — see Code Examples for the exact working syntax.
**Warning signs:** Migration fails immediately on first run with a SQLite syntax error referencing the virtual table statement.

### Pitfall 4: Running `go test` without `-race` locally and only discovering data races in CI (or never)
**What goes wrong:** PLAT-03 explicitly requires the race detector to pass, but it's easy to run `go test ./...` during day-to-day TDD and only remember `-race` right before a phase-completion check — by which point several tasks' tests were "passing" against a build that was never actually race-checked.
**Why it happens:** `-race` is opt-in per invocation and roughly 2-10x slower, so it's tempting to skip it during the fast RED-GREEN loop.
**How to avoid:** Make `-race` part of the phase's one canonical test command from the first task (see Validation Architecture below), not an afterthought added at phase-verification time.
**Warning signs:** A plan or CI config that has a "regular" `go test` step and a separate, later "`-race` check" step — that split is itself the anti-pattern.

### Pitfall 5: `main.go` exits 0 even when `cmd.Execute()` returns an error
**What goes wrong:** A common Cobra starter-template mistake is calling `rootCmd.Execute()` and ignoring its returned error (Cobra already prints the error to stderr by default, which masks the missing `os.Exit(1)`), so scripts/CI treat a failed command as successful.
**Why it happens:** Cobra's default error printing makes the failure *look* handled in the terminal, hiding the missing exit-code propagation.
**How to avoid:** Always check `cmd.Execute()`'s return value in `main.go` and `os.Exit(1)` on non-nil error (see Pattern 1's example).
**Warning signs:** `echo $?` after a deliberately-broken invocation (e.g. `modular status --store-path /root/no-permission`) reports `0`.
</common_pitfalls>

<code_examples>
## Code Examples

### Hand-rolled embed.FS migration runner
```go
// internal/store/migrate.go
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func runMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
		)`); err != nil {
		return fmt.Errorf("ensure schema_migrations: %w", err)
	}

	entries, err := fs.Glob(migrationsFS, "migrations/*.sql")
	if err != nil {
		return fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(entries) // "0001_init.sql" < "0002_....sql" lexically

	for _, name := range entries {
		var version int
		if _, err := fmt.Sscanf(name, "migrations/%04d_", &version); err != nil {
			return fmt.Errorf("parse migration version from %q: %w", name, err)
		}

		var applied bool
		if err := db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)`, version,
		).Scan(&applied); err != nil {
			return fmt.Errorf("check migration %d: %w", version, err)
		}
		if applied {
			continue
		}

		sqlBytes, err := migrationsFS.ReadFile(name)
		if err != nil {
			return fmt.Errorf("read migration %q: %w", name, err)
		}
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx, string(sqlBytes)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %d: %w", version, err)
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schema_migrations (version) VALUES (?)`, version,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", version, err)
		}
	}
	return nil
}
```
Source: pattern synthesized from SQLite/Go community "numbered SQL files + tracking table" convention; no single canonical upstream doc (this exact runner is intentionally minimal/hand-rolled per the Alternatives Considered decision above) — [ASSUMED low-risk: standard, widely-used shape, but this specific code is original to this research, not copied from an authoritative source].

### `modular status` command wiring the whole chain
```go
// cmd/status.go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"nerv-ecosystem/internal/store"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the local store's path and health",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := storePath
			if path == "" {
				path = store.DefaultPath()
			}
			st, err := store.Open(path)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			defer st.Close()

			mode, err := st.JournalMode(cmd.Context())
			if err != nil {
				return fmt.Errorf("read journal mode: %w", err)
			}
			hasFTS, err := st.HasTable(cmd.Context(), "projects_fts")
			if err != nil {
				return fmt.Errorf("check fts5 table: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "store:        %s\njournal_mode: %s\nfts5 ready:   %v\n", path, mode, hasFTS)
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(newStatusCmd())
}
```

### `store.DefaultPath()` — user-home-scoped, env-overridable
```go
// internal/store/path.go
package store

import (
	"os"
	"path/filepath"
)

const defaultDirName = ".modular"
const defaultFileName = "registry.db"

// DefaultPath returns $MODULAR_HOME/registry.db if MODULAR_HOME is set,
// otherwise ~/.modular/registry.db. A shared, home-scoped default (rather
// than a project-local one) is required so Phase 4/5's deps/search commands
// see the same registry regardless of which generated project's directory
// the operator is standing in when they run modular.
func DefaultPath() string {
	if dir := os.Getenv("MODULAR_HOME"); dir != "" {
		return filepath.Join(dir, defaultFileName)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", defaultDirName, defaultFileName)
	}
	return filepath.Join(home, defaultDirName, defaultFileName)
}
```
</code_examples>

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| `mattn/go-sqlite3` (CGO) as the default Go SQLite driver | `modernc.org/sqlite` (pure Go) for CGO-free/cross-platform builds | Ongoing shift since ~2022, now the documented default for laptop-first single-binary CLIs per this project's own `STACK.md` | No C toolchain required to build/cross-compile `modular`; the tradeoff is the DSN-syntax pitfall documented above |
| `go.mod` with only a `go` line | `go` line (minimum) + `toolchain` line (preferred build toolchain) | Introduced Go 1.21 (2023), now standard practice | Lets Phase 1's `go.mod` declare `go 1.25` (compatibility floor, matches locked dependency requirements) while pinning `toolchain go1.26.2`+ (or newer) for reproducible builds — [CITED: go.dev/doc/modules/gomod-ref, go.dev/ref/mod] |
| golangci-lint v1 flat config | golangci-lint v2 versioned config (`version: "2"`, linters under `linters:` block) | v2.0 release, 2025 | Any `.golangci.yml` written for this phase must use v2 schema from the start — copying a v1-era example online will silently no-op or error |

**Deprecated/outdated:**
- `?_journal_mode=`/`?_busy_timeout=` query parameters: only valid for `mattn/go-sqlite3`; meaningless (silently ignored) with `modernc.org/sqlite`.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The exact hand-rolled migration-runner code in Code Examples is original to this research (not copied from an authoritative upstream source) | Code Examples | Low — it's a thin wrapper around standard `database/sql` transaction calls; if a bug exists, `internal/store`'s own TDD suite (Pattern 4) will catch it before it ships |
| A2 | Go module path `nerv-ecosystem` (bare, non-URL) is acceptable for `go.mod` rather than a `github.com/<user>/...` path | Standard Stack installation snippet, Open Questions | Low functional risk (bare module paths build and run fine locally) but blocks `go get`-ing this module from outside the repo later; flagged as an open question for the user/planner to confirm before `go mod init` |
| A3 | `~/.modular/registry.db` (vs. a project-local path) is the correct default store location | Alternatives Considered, Code Examples | Medium — if wrong, Phase 4/5's `deps`/`search` commands would need a path-discovery redesign; the reasoning (shared registry must be visible across `cd`-ed-into generated-project directories) is directly implied by ROADMAP.md's demo flow, so risk is low, but it is still a discretion call, not a locked decision |

## Open Questions

1. **What Go module path should `go.mod` declare?**
   - What we know: the binary/CLI name is `modular` (or "project-chosen name" per PLAT-01's own wording); the repo lives nested inside `go-cook` without its own `.git`.
   - What's unclear: whether a real `github.com/<username>/nerv-ecosystem` path is intended (enabling `go install github.com/.../nerv-ecosystem@latest` later) or a bare local module name is fine for a laptop-only portfolio project.
   - Recommendation: default to a bare `module nerv-ecosystem` (or `module github.com/terenceschumacher/nerv-ecosystem` if a public repo is planned) — either works identically for local `go build`/`go run`/`go test`; the planner should pick one in the first task and treat it as a one-line decision, not a blocker.

2. **Should Phase 1 ship a `.github/workflows/ci.yml` running `go test -race` + `golangci-lint`, or is that deferred?**
   - What we know: PLAT-03's stated success criterion is about `go test -race` passing locally/in the repo, not about CI existing; GEN-04 (Phase 2) is where CI *templates for generated projects* is a hard requirement, which is a different concern (CI templates the CLI generates for others, not CI for the CLI's own repo).
   - What's unclear: whether a CI workflow for the CLI's own repository is implicitly expected now (good practice, cheap to add) or explicitly out of this phase's scope.
   - Recommendation: treat as Claude's discretion; adding a minimal `ci.yml` (checkout → setup-go → `go test -race ./...` → `golangci-lint run`) is low-cost and directly enforces PLAT-03 automatically on every future commit, but is not required to satisfy the phase's literal success criteria.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Everything in this phase | ✓ | go1.26.2 darwin/arm64 [VERIFIED: `go version`] | — |
| git | Commits, `commit_docs` workflow | ✓ | 2.53.0 [VERIFIED: `git --version`] | — |
| golangci-lint | Optional lint step (Open Question 2) | ✓ | v2.12.2, built with go1.26.2 [VERIFIED: `golangci-lint --version`] — exactly matches the project-locked pin | — |
| gsd-sdk | Phase workflow tooling (init/commit) | ✓ | present at `~/.local/bin/gsd-sdk` [VERIFIED] | — |
| slopcheck | Package legitimacy audit | ✓ | present, ran successfully this session [VERIFIED] | — |
| Docker | Not needed until Phase 2+ (OCI registry, testcontainers) | ✓ (present, unused) | present at `/opt/homebrew/bin/docker` [VERIFIED] | N/A for this phase |

**Missing dependencies with no fallback:** none.
**Missing dependencies with fallback:** none — every tool this phase touches is already installed on the target machine.

## Validation Architecture

### Test Framework

| Property | Value |
|----------|-------|
| Framework | Go stdlib `testing` + `testify` (require/assert) v1.11.1 |
| Config file | none — `go.mod` alone; no separate test-framework config needed |
| Quick run command | `go test ./internal/... -race` |
| Full suite command | `go test ./... -race -count=1` |

### Phase Requirements → Test Map

| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|---------------------|--------------|
| PLAT-01 | `modular status` (or equivalent) runs and exits 0 with no network/daemon dependency | integration (in-process, no network) | `go test ./cmd/... -run TestStatusCommand -race` | ❌ Wave 0 |
| PLAT-02 | `store.Open` creates a WAL-mode file with `projects`/`projects_fts` present | unit | `go test ./internal/store/... -run TestOpen -race` | ❌ Wave 0 |
| PLAT-03 | All `internal/store` tests are table-driven and pass under `-race`; failing test precedes each production change | unit (self-referential — the test suite itself is the proof) | `go test ./... -race -count=1` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/... -race`
- **Per wave merge:** `go test ./... -race -count=1`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/store/store_test.go` — covers PLAT-02 (WAL mode + FTS5 table existence)
- [ ] `internal/store/migrate_test.go` — covers PLAT-03 (migration idempotency: calling `Open` twice must not error or double-apply)
- [ ] `cmd/status_test.go` — covers PLAT-01 (command runs end-to-end against a temp store, using Cobra's `SetOut`/`SetArgs`/`Execute()` test pattern, no live network)
- [ ] `go.mod`/`go.sum` — no test framework installed yet; `go get github.com/stretchr/testify@v1.11.1` is itself a Wave 0 task

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | No | Single-operator local CLI; no auth surface in this phase |
| V3 Session Management | No | No sessions — one-shot CLI invocations |
| V4 Access Control | No | No multi-user access boundary in this phase |
| V5 Input Validation | Minimal | `--store-path` is a developer-supplied flag, not untrusted network input; still pass it through `filepath.Clean` before use. Real untrusted-input validation (path traversal on `--name`/`--team`) is explicitly GEN-07 in Phase 2, not this phase. |
| V6 Cryptography | No | No secrets/crypto touched by store bootstrap |
| V12 Files and Resources | Yes | Create the store directory with restrictive permissions (`os.MkdirAll(dir, 0o700)`) since it's a fixed, non-shared location that will hold the project's full registry in later phases; never construct SQL by string-concatenating table/column names from anything outside the static migration files (all runtime queries use `?` placeholders) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| SQL injection via string-built queries | Tampering | Always use parameterized queries (`db.QueryContext(ctx, "... WHERE id = ?", id)`); never `fmt.Sprintf` user-controlled values into SQL. Migration SQL itself is static/embedded, not runtime-constructed, so it is not an injection surface. |
| World-readable store directory/file exposing registry contents | Information Disclosure | `0o700` directory permission (see V12 above); SQLite driver defaults new files to `0o600` — verify this isn't overridden |
| Silent WAL-mode failure leaving the store vulnerable to reader/writer corruption under concurrent access | Tampering / Denial of Service | Common Pitfall 1's test-the-actual-pragma-value discipline doubles as a security control here — an un-verified fallback to rollback-journal mode is also a correctness/availability risk under any future concurrent access (e.g. a background watcher process) |

<sources>
## Sources

### Primary (HIGH confidence)
- proxy.golang.org/{github.com/spf13/cobra,modernc.org/sqlite,github.com/stretchr/testify,github.com/google/go-cmp}/@latest and modernc.org/sqlite/@v/list — live version/date verification, 2026-07-24 session
- sqlite.org/fts5.html — official FTS5 documentation (full page fetched this session): external-content tables, trigger-sync pattern, virtual-table column-list grammar constraints
- go.dev/doc/modules/gomod-ref, go.dev/ref/mod, go.dev/doc/toolchain — `go`/`toolchain` directive syntax, official
- Local environment checks this session: `go version` (go1.26.2), `git --version` (2.53.0), `golangci-lint --version` (v2.12.2), `slopcheck --version`/run, `gsd-sdk` presence
- `.planning/research/{SUMMARY,STACK,ARCHITECTURE}.md` and `.planning/{PROJECT,ROADMAP,REQUIREMENTS,STATE}.md` — this project's own prior, already-verified research and locked decisions

### Secondary (MEDIUM confidence)
- modernc.org/sqlite DSN pragma syntax — corroborated across three independent 2026-dated Go repositories (pgEdge/ai-dba-workbench commit c9e6ee9; MaorBril/clauder PR #24; BorisYaoA/cagent `pkg/sqliteutil/sqlite.go`), all converging on identical `_pragma=name(value)` syntax and `SetMaxOpenConns(1)` mitigation — not a single canonical modernc.org README quote, but consistent, credible, and independently arrived-at across unrelated projects
- golang-migrate's `modernc.org/sqlite`-compatible driver (`database/sqlite`, distinct from `database/sqlite3`) — confirmed via the driver's own merged PR (golang-migrate/migrate#555) and pkg.go.dev listing; used only to inform the Alternatives Considered tradeoff, not adopted for Phase 1
- Cobra `main.go` + `cmd/root.go` scaffold convention — matches `cobra-cli init`'s documented default output and cobra.dev's official command-organization guide (already cited HIGH in the project's own `ARCHITECTURE.md`; not independently re-fetched this session, carried forward as consistent with official docs)

### Tertiary (LOW confidence)
- None — every load-bearing claim in this document was either freshly verified against an official registry/doc this session, or carried forward from the project's own already-verified `STACK.md`/`ARCHITECTURE.md` research.
</sources>

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version re-verified against `proxy.golang.org` live this session, all match project-locked `STACK.md`
- Architecture: HIGH — Cobra/`internal/store` layout is a direct, minor refinement of the project's own already-HIGH-confidence `ARCHITECTURE.md`, adjusted only to honor the "cmd/ for Cobra only" locked layout note
- Pitfalls: HIGH for the WAL-DSN and FTS5-grammar pitfalls (multiple independent corroborating sources + official SQLite docs); MEDIUM for the migration-runner code shape (original synthesis, not copied from an authoritative source, but mitigated by TDD)

**Research date:** 2026-07-24
**Valid until:** 2026-08-23 (30 days — Go/Cobra/SQLite tooling in this phase is stable; re-verify package versions if planning is delayed past this window)

---

*Phase: 01-platform-foundation*
*Research completed: 2026-07-24*
*Ready for planning: yes*
