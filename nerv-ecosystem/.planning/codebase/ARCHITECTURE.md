<!-- refreshed: 2026-07-25 -->
# Architecture

**Analysis Date:** 2026-07-25

## System Overview

```text
┌─────────────────────────────────────────────────────────────┐
│                  main.go (binary entrypoint)                 │
│                       `main.go`                               │
└───────────────────────────┬───────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              cmd/ — Cobra CLI surface (routing only)          │
│  root command (--store-path)   │   status subcommand           │
│  `cmd/root.go`                 │   `cmd/status.go`              │
└───────────────────────────┬───────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│         internal/store — sole owner of database/sql          │
│  Open/Close/JournalMode/HasTable/SchemaObjects                │
│  `internal/store/store.go`                                    │
│  migration runner (embed.FS numbered .sql files)               │
│  `internal/store/migrate.go`                                   │
│  DefaultPath resolution ($MODULAR_HOME / ~/.modular)            │
│  `internal/store/path.go`                                      │
└───────────────────────────┬───────────────────────────────────┘
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  One embedded SQLite file (WAL mode, FTS5 compiled in)        │
│  `projects` table + `projects_fts` virtual table + 3 triggers  │
│  `internal/store/migrations/0001_init.sql`                     │
│  Location: $MODULAR_HOME/registry.db or ~/.modular/registry.db │
└─────────────────────────────────────────────────────────────┘
```

**Planned (not yet present):** `internal/generate`, `internal/publish`, `internal/deps`, `internal/search`, `internal/ociregistry`, `internal/templates` domain packages sit between `cmd/` and `internal/store` per `.planning/research/ARCHITECTURE.md`. None of these directories exist yet — see "Planned Architecture" below.

## Component Responsibilities

| Component | Responsibility | File |
|-----------|----------------|------|
| Binary entrypoint | Calls `cmd.Execute()`, prints top-level errors, sets exit code | `main.go` |
| Root command | Builds a fresh Cobra command tree per invocation, owns the persistent `--store-path` flag and its cleaning (`filepath.Clean`) | `cmd/root.go` |
| Status command | Opens (creating/migrating) the store, reads back journal mode and FTS5 table presence, writes plain-text status to stdout | `cmd/status.go` |
| Store lifecycle | Owns the single `*sql.DB`, DSN construction, WAL pragmas, connection-pool limits (`SetMaxOpenConns(1)`), post-open permission tightening | `internal/store/store.go` |
| Migration runner | Applies numbered `.sql` files from an `embed.FS`, tracks applied versions in `schema_migrations`, each migration in its own transaction | `internal/store/migrate.go` |
| Path resolution | Resolves the default store file location from `$MODULAR_HOME` or `~/.modular` | `internal/store/path.go` |
| Schema + FTS5 sync | Defines `projects` table, `projects_fts` FTS5 virtual table, and 3 AFTER INSERT/UPDATE/DELETE triggers that keep the index in sync with `projects` | `internal/store/migrations/0001_init.sql` |

## Pattern Overview

**Overall:** Single-binary modular monolith — one Cobra command tree over a small set of `internal/` packages sharing one embedded SQLite datastore. No network boundary between components; all calls are direct in-process Go function calls.

**Key Characteristics:**
- Strict layering enforced by convention and comment (`cmd/root.go:1-4` explicitly states `cmd` never imports `database/sql` or a SQLite driver)
- `internal/store` is the sole owner of `database/sql` and the schema — every other package reaches persistence only through `Store`'s exported methods
- No business logic in the CLI layer — `RunE` closures in `cmd/status.go` call straight into `store.Open`/`store.JournalMode`/`store.HasTable` and format output; no validation/business rules live in `cmd/`

## Layers

**CLI layer (`cmd/`):**
- Purpose: Parse flags/args, invoke exactly one domain package (currently only `internal/store` directly, since no `generate`/`publish`/`deps`/`search` domain packages exist yet), format output to `cmd.OutOrStdout()`
- Location: `cmd/root.go`, `cmd/status.go`
- Contains: Cobra command construction (`NewRootCommand`, `newStatusCommand`), flag wiring, `RunE` closures
- Depends on: `internal/store` (only), `spf13/cobra`
- Used by: `main.go`

**Store layer (`internal/store/`):**
- Purpose: Own the SQLite connection, DSN/pragma configuration, migration execution, and read-only introspection queries (`JournalMode`, `HasTable`, `SchemaObjects`, `AppliedMigrations`)
- Location: `internal/store/store.go`, `internal/store/migrate.go`, `internal/store/path.go`
- Contains: `*sql.DB` lifecycle, embedded migration SQL, repository-style exported methods
- Depends on: `modernc.org/sqlite` (driver, imported for side effects only: `_ "modernc.org/sqlite"`)
- Used by: `cmd/status.go` (today); will be used by all future domain packages

## Data Flow

### Primary Request Path (`modular status`)

1. `main.go:11` calls `cmd.Execute()`
2. `cmd.Execute()` (`cmd/root.go:47-49`) builds a fresh root command and calls `ExecuteContext`
3. `PersistentPreRunE` (`cmd/root.go:31-36`) cleans `--store-path` via `filepath.Clean` if set
4. `newStatusCommand`'s `RunE` (`cmd/status.go:19-47`) resolves the path (flag value or `store.DefaultPath()`), calls `store.Open(path)`
5. `store.Open` (`internal/store/store.go:31-67`) creates the parent dir, builds the DSN, opens the SQLite connection, runs migrations (`runMigrations` in `internal/store/migrate.go:19-60`), then `os.Chmod`s the file to `0600`
6. `RunE` calls `st.JournalMode(ctx)` and `st.HasTable(ctx, "projects_fts")`, both of which issue read-only queries against the live SQLite handle
7. Result is written via `fmt.Fprintf(cmd.OutOrStdout(), ...)` — store path, journal mode, FTS5 readiness

### Migration Path (inside `store.Open`)

1. `runMigrations` ensures `schema_migrations` exists (`internal/store/migrate.go:20-26`)
2. Globs `migrations/*.sql` from the embedded `embed.FS` (`internal/store/migrate.go:12-13,28`), lexically sorts filenames (relies on 4-digit zero-padding — see `CONCERNS.md` IN-02)
3. For each migration file, checks `schema_migrations` for the parsed version; skips if already applied
4. Otherwise reads the SQL text and calls `applyMigration` (`internal/store/migrate.go:97-117`), which runs the SQL and inserts the `schema_migrations` row in one transaction (commit-or-rollback together)

**State Management:**
- All state lives in one on-disk SQLite file. No in-memory application state persists across invocations — each CLI run opens the store fresh, runs (no-op if already applied) migrations, and closes it via `defer st.Close()`.

## Key Abstractions

**`Store` (internal/store/store.go:17-21):**
- Purpose: Wraps a single `*sql.DB` connection to one embedded SQLite file; the only type that touches SQL directly
- Examples: `internal/store/store.go`, `internal/store/migrate.go` (methods on `*Store`)
- Pattern: Thin wrapper struct with connection + path; exported methods return plain values/errors, never `*sql.Rows` or `*sql.DB` to callers

**`NewRootCommand()` factory (cmd/root.go:18-44):**
- Purpose: Builds a fresh Cobra command tree per call so tests can run in parallel without shared flag state (explicitly forbids package-level `init()` registration — see comment at `cmd/root.go:14-17`)
- Examples: `cmd/root.go`, consumed by `cmd/status_test.go`, `cmd/status_failure_test.go`
- Pattern: Constructor function returning `*cobra.Command`, closures capturing local flag variables (`storePath`) rather than package-level globals

## Entry Points

**`main.go`:**
- Location: `main.go`
- Triggers: `go build -o modular . && ./modular <subcommand>`, or `go run .`
- Responsibilities: Call `cmd.Execute()`; on error, print to stderr and `os.Exit(1)`

**`modular status` (Cobra subcommand):**
- Location: `cmd/status.go`
- Triggers: `./modular status [--store-path PATH]`
- Responsibilities: Open/migrate the store, print path/journal_mode/fts5-readiness, return non-zero exit via `RunE` error on failure

## Architectural Constraints

- **Threading:** Single-threaded CLI execution per invocation; no goroutines spawned in application code. SQLite connection pool is deliberately capped at 1 (`db.SetMaxOpenConns(1)` / `SetMaxIdleConns(1)`, `internal/store/store.go:48-49`) because "SQLite serializes writers at the file level even in WAL mode."
- **Global state:** None at the Go level — `NewRootCommand()` explicitly avoids package-level `init()`/globals so parallel tests don't share flag state. The only durable global state is the on-disk SQLite file itself.
- **Circular imports:** None possible today — `cmd` → `internal/store` is a one-way dependency; `internal/store` imports no other project package.
- **Package boundary rule:** `cmd/` and any future domain package (`internal/generate`, etc.) must never import `database/sql` or a SQLite driver directly — this is enforced by comment/convention today (`cmd/root.go:1-4`, `internal/store/store.go:1-4`), not by a build-time linter rule. There is no automated enforcement (e.g., a golangci-lint `depguard` rule) — see `CONCERNS.md`.

## Anti-Patterns

### Splitting the CLI into multiple binaries/services

**What happens:** Not observed in this codebase — flagged proactively because `.planning/research/ARCHITECTURE.md` explicitly warns against it.
**Why it's wrong:** Would add serialization, port management, and partial-failure modes with zero benefit for a single-binary, single-operator CLI.
**Do this instead:** Keep `generate`/`publish`/`deps`/`search` (when added) as `internal/` Go packages called in-process from `cmd/`, exactly like the current `cmd/status.go` → `internal/store` call pattern.

### One SQLite file per domain concern

**What happens:** Not observed — flagged proactively per research guidance, since Phase 2+ will be tempted to add a second store for template metadata, etc.
**Why it's wrong:** Fragments a single-operator project's state across files that must be kept consistent by hand; the original Nerv system's multi-datastore history existed for organizational reasons that don't apply here.
**Do this instead:** Extend the one `internal/store` SQLite file/schema (new migration files) as the shared system of record, exactly as `01-SKELETON.md` documents for Phase 3 (`versions`+`edges` arrive as `0002_*.sql` through the same runner).

## Error Handling

**Strategy:** Every fallible operation returns a wrapped `error` (`fmt.Errorf("...: %w", err)`); no panics in application code. `cmd/status.go`'s `RunE` returns errors up through Cobra, which (with `SilenceUsage: true` set on the root command, `cmd/root.go:24`) prints just the error message to stderr and causes `main.go` to `os.Exit(1)`.

**Patterns:**
- Wrap every error with a short present-tense description of the failed operation (`"open store: %w"`, `"create store dir: %w"`, `"read journal mode: %w"`) so error chains are diagnosable without a debugger
- Best-effort cleanup calls (`Close()`, `rows.Close()`) are explicitly discarded with `_ = ...` and a comment, never silently ignored without acknowledgment
- `SilenceUsage: true` on the root command prevents Cobra from dumping usage text on every runtime error (only genuine flag-parsing errors show usage) — regression-locked by `cmd/status_failure_test.go`'s `require.NotContains(..., "Usage:")` assertions

## Cross-Cutting Concerns

**Logging:** None — no logging framework or `log`/`slog` usage anywhere in the codebase. All operator-facing output goes through Cobra's `cmd.OutOrStdout()`/`cmd.OutOrStderr()` (testable) rather than global `fmt.Println`/`os.Stdout`.

**Validation:** Minimal and narrowly scoped — `cmd/root.go`'s `PersistentPreRunE` only `filepath.Clean`s `--store-path` because it is "developer-supplied input, not untrusted network input" (comment, `cmd/root.go:26-30`). No other input validation exists yet; path-traversal validation of untrusted `--name`/`--team` is explicitly deferred to Phase 2 (GEN-07) per `01-SKELETON.md`.

**Authentication:** Not applicable — this is a local single-user CLI with no auth boundary in Phase 1 or in the planned architecture.

## Planned Architecture (from research, not yet implemented)

`.planning/research/ARCHITECTURE.md` specifies a target shape that Phase 1 is the foundation for. None of the following exist as code yet:

```
nerv-ecosystem/
├── internal/
│   ├── cli/            # (research suggests renaming cmd/ → internal/cli/; current code uses cmd/ directly)
│   ├── generate/        # Phase 2 — template render + first `projects` row writer
│   ├── publish/         # Phase 3 — apidiff + semver gate + version/edge writes
│   ├── deps/            # Phase 4 — DAG build + blast-radius walk over store edges
│   ├── search/          # Phase 5 — FTS5 MATCH queries
│   ├── ociregistry/      # Phase 3 — local OCI artifact push/pull, no daemon
│   ├── templates/        # Phase 2 — template discovery/versioning (separate from rendering)
│   └── store/            # EXISTS TODAY — will gain packages.go/versions.go/edges.go/fts.go as schema grows
```

Planned pattern: each future domain package depends on narrow interfaces (e.g. `store.ProjectRepo`, `ociregistry.Pusher`) rather than concrete types, wired at `cmd`/composition time — "ports and adapters." Today's codebase has no interfaces yet because there is only one caller (`cmd/status.go`) and one implementation (`*store.Store`); this pattern should be introduced when the second domain package (Phase 2 `generate`) is added, not before.

---

*Architecture analysis: 2026-07-25*
