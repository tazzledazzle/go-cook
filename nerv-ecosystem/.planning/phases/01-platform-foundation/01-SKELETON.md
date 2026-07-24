# Walking Skeleton — Nerv Ecosystem

**Phase:** 1 (01-platform-foundation)
**Generated:** 2026-07-24

> This is a contract, not a scratchpad. Phases 2–5 add vertical slices *on top of* these decisions
> without renegotiating them. Changing anything in "Architectural Decisions" requires an explicit
> decision entry in PROJECT.md, not a silent edit here.

## Capability Proven End-to-End

An operator runs `./modular status` from a shell and sees the real on-disk store path, `journal_mode: wal`,
and `fts5 ready: true` — proving that a single Go binary opened (creating it if absent) one embedded SQLite
file in WAL mode with the FTS5 search table present, with no cloud account and no running daemon.

## Architectural Decisions

| Decision | Choice | Rationale |
|---|---|---|
| Language / toolchain | Go, `go.mod` floor `go 1.25`, built with toolchain `go1.26.2`+ | Locked project-wide; every downstream dep (oras-go, OTel, prometheus/client_golang) has a Go 1.25 floor and a two-minor support window |
| Module path | `github.com/tazzledazzle/go-cook/nerv-ecosystem` | Resolves RESEARCH Open Question 1. The module lives in a subdirectory of the `tazzledazzle/go-cook` GitHub repo, so this is the honest importable path and keeps `go install …/nerv-ecosystem@latest` possible later. Bare `module nerv-ecosystem` was rejected because it permanently blocks external `go get`. |
| Binary name | `modular` | Matches PLAT-01's "project-chosen name" and the demo flow in PROJECT.md |
| CLI framework | `spf13/cobra` v1.10.2 | Locked. `cmd/` holds Cobra wiring only — no business logic, no `database/sql` import |
| Data layer | One embedded SQLite file via `modernc.org/sqlite` v1.54.0 (pure Go, CGO-free) | Locked. FTS5 compiled in; single-binary cross-compilation and `-race` stay trivial |
| SQLite DSN | `file:<path>?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)` with `SetMaxOpenConns(1)`/`SetMaxIdleConns(1)` | modernc-specific `_pragma=name(value)` syntax. The `mattn/go-sqlite3` form `?_journal_mode=WAL` is silently ignored by this driver (RESEARCH Pitfall 1) — the DSN is asserted in tests, not assumed |
| Store location | `$MODULAR_HOME/registry.db`, else `~/.modular/registry.db`; overridable per-invocation with `--store-path` | One shared registry visible from any working directory. A project-local store would fragment the registry and break Phase 4/5 `deps`/`search` after `cd`-ing into a generated service |
| Migrations | Hand-rolled `embed.FS` runner (numbered `.sql` files + `schema_migrations` tracking table), ~40 lines | One migration exists in Phase 1; `golang-migrate` is a new dependency outside the locked stack. Revisit if Phase 2/3 need `down` migrations |
| Phase 1 schema | `projects` + `projects_fts` (FTS5 external content, `content='projects'`, `content_rowid='id'`) + three AFTER INSERT/UPDATE/DELETE sync triggers. **No `versions`, no `edges`.** | Schema is built incrementally per slice: `versions` arrives with Phase 3 publish, `edges` with Phase 3/4 |
| Search index sync | SQLite triggers, not application code | Sync is atomic with the write transaction and impossible for a future `generate`/`publish` call site to forget |
| Package layout | `main.go` (root) → `cmd/` (Cobra only) → `internal/store` (sole owner of `database/sql` and the schema) | Locked layout. Feature-domain packages (`internal/generate`, …) arrive in Phase 2 and reach the DB only through `internal/store`'s exported methods |
| Test approach | Go stdlib `testing` + `testify` v1.11.1, table-driven, `t.Parallel()`, real SQLite files under `t.TempDir()`, always `-race` | PLAT-03. No mocks for SQLite — the driver's real behavior (WAL pragma, FTS5 grammar) is exactly what must be proven |
| Run target | Local single binary: `go build -o modular . && ./modular status` | No deploy environment exists; this is the documented full-stack run command |

## Stack Touched in Phase 1

- [ ] Project scaffold — `go.mod` with pinned deps, `.golangci.yml` (v2 schema), GitHub Actions CI running `go test -race`
- [ ] Command routing — at least one real Cobra command (`modular status`) reachable from the built binary
- [ ] Database — one real write (migration runner inserts `schema_migrations` rows inside a transaction; FTS5 triggers fire on a `projects` insert under test) **and** one real read (`PRAGMA journal_mode`, `sqlite_master` lookup, FTS5 `MATCH`)
- [ ] Operator-visible output — `status` prints store path / journal mode / fts5 readiness to stdout, exits non-zero on failure
- [ ] Local run — `go build -o modular . && ./modular status` documented and exercised in an automated verify

## Out of Scope (Deferred to Later Slices)

Explicit so later phases do not re-litigate Phase 1's minimalism:

- `versions` and `edges` tables, and any publish/consumer edge writes (Phase 3/4)
- Any `generate`, `publish`, `deps`, or `search` command or domain package (Phases 2–5)
- A production API for writing `projects` rows — Phase 1 only proves the table and its triggers exist and work; `generate` (GEN-08) owns the writer
- Viper / layered config — no real setting exists to layer yet (Phase 2 introduces template root and registry path)
- OCI registry, `apidiff`, semver gate, OTel instrumentation
- Path-traversal validation of untrusted `--name`/`--team` input (GEN-07, Phase 2). Phase 1 only cleans the developer-supplied `--store-path`
- Cross-process concurrency tuning beyond `SetMaxOpenConns(1)` + `busy_timeout`

## Subsequent Slice Plan

Each later phase adds one vertical slice on top of this skeleton without altering the decisions above:

- **Phase 2 — Generate:** `modular generate --lang … --name … --team …` renders a template tree and writes the first real `projects` row through `internal/store`, which fires the FTS5 triggers this skeleton installed.
- **Phase 3 — Publish:** adds `versions` + `edges` migrations (`0002_*.sql` through the same runner) and `internal/ociregistry`; the semver gate blocks breaking changes.
- **Phase 4 — Dependency Graph:** reads persisted edges through `internal/store` and materializes a DAG per invocation.
- **Phase 5 — Search:** queries `projects_fts` with `MATCH`/bm25 — the index is already live and trigger-synced, so no reindex step exists to build.
