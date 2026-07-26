# Codebase Structure

**Analysis Date:** 2026-07-25

## Directory Layout

```
nerv-ecosystem/
├── main.go                        # Binary entrypoint — calls cmd.Execute()
├── go.mod                         # Module github.com/tazzledazzle/go-cook/nerv-ecosystem, go 1.25.0
├── Makefile                       # test/lint/build/smoke targets
├── .golangci.yml                  # golangci-lint v2 config
├── cmd/                           # Cobra CLI wiring only (no business logic)
│   ├── root.go                    # NewRootCommand(), Execute(), --store-path flag
│   ├── status.go                  # `modular status` subcommand
│   ├── status_test.go             # happy-path status test
│   └── status_failure_test.go     # failure-path + path-cleaning tests
├── internal/
│   └── store/                     # Sole owner of database/sql + SQLite schema
│       ├── store.go               # Store type, Open/Close/JournalMode/HasTable/SchemaObjects
│       ├── migrate.go             # embed.FS migration runner + schema_migrations tracking
│       ├── path.go                # DefaultPath() — $MODULAR_HOME / ~/.modular resolution
│       ├── migrations/
│       │   └── 0001_init.sql      # projects table + projects_fts + 3 sync triggers
│       ├── store_test.go           # black-box Open/JournalMode/HasTable tests (package store_test)
│       ├── store_perms_test.go     # black-box file/dir permission regression tests
│       ├── migrate_test.go         # black-box reopen/no-reapply + WAL multi-reader tests
│       └── fts_internal_test.go    # white-box FTS5 trigger sync tests (package store)
└── .planning/                     # GSD planning artifacts (not application code)
    ├── PROJECT.md, ROADMAP.md, REQUIREMENTS.md, STATE.md, config.json
    ├── research/                  # STACK.md, ARCHITECTURE.md, FEATURES.md, PITFALLS.md, SUMMARY.md
    ├── codebase/                  # THIS directory — generated codebase maps
    └── phases/
        ├── 01-platform-foundation/  # PLAN/SUMMARY/REVIEW/VALIDATION/VERIFICATION for Phase 1 (complete)
        └── 02-generate/             # CONTEXT/DISCUSSION-LOG for Phase 2 (in discussion, no code yet)
```

Also present at the outer `go-cook` repo root (not part of this module but relevant to CI):
- `.github/workflows/nerv-ecosystem-ci.yml` — path-filtered CI running `go test -race` and `golangci-lint` scoped to `nerv-ecosystem/**`

## Directory Purposes

**`cmd/`:**
- Purpose: Cobra command tree construction and flag wiring only
- Contains: One `.go` file per command (currently `root.go` for the root command, `status.go` for the one subcommand), plus their tests
- Key files: `cmd/root.go` (command tree factory), `cmd/status.go` (only subcommand today)
- Hard rule (enforced by comment, not tooling): never import `database/sql` or a SQLite driver in this package

**`internal/store/`:**
- Purpose: Single system-of-record persistence layer — connection lifecycle, schema, migrations
- Contains: Non-test `.go` files own `*sql.DB` and SQL text; `_test.go` files split between external (`package store_test`, black-box) and internal (`package store`, white-box, used only when a test needs the unexported `db` field — see `fts_internal_test.go`)
- Key files: `internal/store/store.go` (public API surface), `internal/store/migrate.go` (schema evolution mechanism)

**`internal/store/migrations/`:**
- Purpose: Numbered, embedded SQL files applied in order by `runMigrations`
- Contains: `.sql` files named `NNNN_description.sql` (4-digit zero-padded)
- Key files: `0001_init.sql` (only migration in Phase 1 — `projects` + `projects_fts` + triggers)

**`.planning/`:**
- Purpose: GSD workflow planning artifacts — not consumed by the Go build, but is the authoritative source for phase scope, architectural decisions, and research
- Contains: Phase plans/summaries/reviews, project-level requirements/roadmap, research documents, this codebase map directory
- Key files: `.planning/phases/01-platform-foundation/01-SKELETON.md` (locked architectural decisions for Phase 1), `.planning/phases/01-platform-foundation/01-REVIEW.md` (open review findings WR-01/WR-02)

## Key File Locations

**Entry Points:**
- `main.go`: process entrypoint, delegates immediately to `cmd.Execute()`

**Configuration:**
- `go.mod`: module path, Go version floor, dependency versions
- `.golangci.yml`: lint rule configuration (v2 schema)
- `Makefile`: canonical dev commands (`test`, `lint`, `build`, `smoke`)

**Core Logic:**
- `cmd/root.go`, `cmd/status.go`: CLI surface
- `internal/store/store.go`, `internal/store/migrate.go`, `internal/store/path.go`: persistence layer

**Testing:**
- `cmd/status_test.go`, `cmd/status_failure_test.go`: CLI-level tests (black-box, `package cmd_test`)
- `internal/store/store_test.go`, `store_perms_test.go`, `migrate_test.go`: store-level black-box tests (`package store_test`)
- `internal/store/fts_internal_test.go`: store-level white-box test (`package store`) — the only test file that reaches the unexported `db` field

## Naming Conventions

**Files:**
- Production file: `<concern>.go` (e.g. `store.go`, `migrate.go`, `path.go`, `status.go`, `root.go`) — one primary concern per file, named after the noun/verb it owns
- Test file: `<same-basename>_test.go` for tests co-located with the production file they exercise, but this project favors splitting by **behavior** over exact 1:1 basename mirroring (e.g. `store_perms_test.go` and `fts_internal_test.go` both test aspects of `store.go`/`migrate.go` but are named after the behavior under test, not the production file)
- Migration files: `NNNN_short_description.sql`, 4-digit zero-padded sequence number (`0001_init.sql`)

**Directories:**
- `internal/<domain>/` — one directory per bounded persistence/domain concern (today: only `store`; future: `generate`, `publish`, `deps`, `search`, `ociregistry`, `templates` per research)
- `cmd/` stays flat (no subdirectories) while there is only one subcommand; research architecture suggests `internal/cli/<verb>/command.go` once more verbs are added, but the current implementation keeps everything directly in `cmd/`

## Where to Add New Code

**New Cobra subcommand (e.g. `modular generate`):**
- Command wiring: new file in `cmd/` (e.g. `cmd/generate.go`) exporting `newGenerateCommand(...)`, registered via `root.AddCommand(...)` in `cmd/root.go`
- Domain logic: new package `internal/generate/` (cobra-free, unit-testable) — the command's `RunE` should call into this package, not implement logic inline (mirrors `cmd/status.go`'s calls into `internal/store`)
- Tests: `cmd/generate_test.go` (black-box CLI test, `package cmd_test`) plus `internal/generate/generate_test.go` (domain logic tests)

**New store capability (e.g. a `Projects.Create` writer):**
- Implementation: new file in `internal/store/` (e.g. `internal/store/projects.go`), exported methods on `*Store`
- Schema change: new migration file `internal/store/migrations/0002_<description>.sql`, following the existing 4-digit zero-padded numbering
- Tests: black-box test in `internal/store/*_test.go` (`package store_test`) unless the test needs the unexported `db` field, in which case add to (or create a new) internal test file (`package store`)

**Utilities:**
- No shared utility/helpers package exists yet. If cross-cutting helpers are needed (e.g. path validation shared by `cmd/root.go` and a future `internal/generate`), place them in a new `internal/pathutil/` (or similar) package rather than duplicating logic — do not put shared helpers in `internal/store` or `cmd/`, both of which have single, narrow responsibilities today.

## Special Directories

**`internal/store/migrations/`:**
- Purpose: SQL migration source of truth, embedded into the binary via `//go:embed migrations/*.sql` (`internal/store/migrate.go:12-13`)
- Generated: No — hand-written SQL files
- Committed: Yes

**`.planning/codebase/`:**
- Purpose: Generated codebase-map documents (this file and its siblings), consumed by GSD planning/execution commands
- Generated: Yes (by the codebase-mapper subagent)
- Committed: Yes (per project convention of committing planning artifacts)

**`.beads/`:**
- Purpose: Beads task-tracking backing store (embedded Dolt database under `.beads/embeddeddolt/nerv_ecosystem/`)
- Generated: Yes
- Committed: Not verified — treat as tooling state, not application source

---

*Structure analysis: 2026-07-25*
