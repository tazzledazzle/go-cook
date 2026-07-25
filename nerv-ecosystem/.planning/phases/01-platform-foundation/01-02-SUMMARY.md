---
phase: 01-platform-foundation
plan: 02
subsystem: database
tags: [go, sqlite, fts5, wal, tdd, ci, golangci-lint]

# Dependency graph
requires: ["01-01"]
provides:
  - "internal/store: AppliedMigrations(ctx) exposing schema_migrations rows without reaching into the unexported *sql.DB"
  - "internal/store: reopen idempotency, FTS5 insert/update/delete trigger sync, 0700 dir + 0600 file permissions, and open-failure error wrapping, all regression-locked by tests"
  - "cmd/root.go: PersistentPreRunE cleans --store-path via filepath.Clean before any subcommand runs"
  - ".golangci.yml + Makefile + ../.github/workflows/nerv-ecosystem-ci.yml: -race and lint enforced locally (make test/lint/smoke) and in CI on every push touching nerv-ecosystem/"
affects: ["Phase 2 generate (first real projects writer; permissions/reopen/lint guarantees now provable)", "Phase 5 search (SRCH-03 no-manual-reindex precondition now proven by fts_internal_test.go)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "In-package `store` test file (fts_internal_test.go) for raw-SQL trigger round-trips; external `store_test` package for reopen/permission/WAL-visibility tests"
    - "WAL multi-reader proof via two raw sql.DB connections opened directly onto a store-bootstrapped file, since no production `projects` writer exists yet (Phase 2 owns that API)"
    - "Explicit os.Chmod(path, 0o600) on the store file after Open, rather than relying on the driver's process-umask-derived creation mode"
    - "cmd root PersistentPreRunE as the single point where developer-supplied --store-path is normalized before any subcommand consumes it"

key-files:
  created:
    - internal/store/migrate_test.go
    - internal/store/fts_internal_test.go
    - internal/store/store_perms_test.go
    - cmd/status_failure_test.go
    - .golangci.yml
    - Makefile
    - ../.github/workflows/nerv-ecosystem-ci.yml
    - .gitignore
  modified:
    - internal/store/migrate.go
    - internal/store/store.go
    - internal/store/path.go
    - cmd/root.go
    - cmd/status.go
    - internal/store/store_test.go

key-decisions:
  - "Deviated from the plan's stated implementation preference (\"do not add a chmod... rely on the driver's 0600 creation mode\"): the driver actually creates the file under the process umask (observed group/other-readable), so an explicit os.Chmod(path, 0o600) was added after Open to satisfy T-1-02 — the plan's behavioral requirement (no group/other bits) is unchanged, only the *how* changed based on what the test revealed"
  - "WAL multi-reader case in migrate_test.go opens two raw sql.DB connections directly onto the store-bootstrapped file rather than two *store.Store values, because Store exposes no projects writer (Phase 2's generate owns that API) and the external store_test package cannot reach the unexported *sql.DB"
  - "AppliedMigrations returns []MigrationRecord{Version, AppliedAt} (richer than the plan's suggested ([]int, error)) so both the reopen-idempotency count check and the applied_at-unchanged check go through one exported method instead of mixing an exported call with a raw-SQL escape hatch"

requirements-completed: [PLAT-01, PLAT-02, PLAT-03]

# Metrics
duration: 45min
completed: 2026-07-24
---

# Phase 1 Plan 02: Hardening — Reopen, FTS5 Sync, Permissions, CI Summary

**Regression-locks wave 1's store guarantees (reopen idempotency, FTS5 trigger sync, 0700/0600 permissions, wrapped failure exit codes) with new TDD coverage, fixes a real permission gap the tests caught, adds `--store-path` cleaning, and wires `golangci-lint` v2 + `go test -race` into both a `Makefile` and a repo-root GitHub Actions workflow so PLAT-03 cannot silently lapse.**

## Performance

- **Duration:** ~45 min
- **Tasks:** 3 (reopen/FTS5 TDD, permissions/path/failure TDD, lint+CI harness)
- **Files created:** 8 (4 test files, `.golangci.yml`, `Makefile`, CI workflow, `.gitignore`)
- **Files modified:** 6

## Accomplishments

- Proved reopen safety: two sequential `store.Open` calls on the same file leave exactly one `schema_migrations` row with an unchanged `applied_at`, verified through a new exported `AppliedMigrations` method rather than reaching into the unexported `*sql.DB`
- Proved the FTS5 triggers wave 1 shipped actually work: table-driven insert/update/delete round-trips against `projects_fts` via parameterized `MATCH ?` queries, run in-package to reach raw SQL without adding a production `projects` writer (that's Phase 2's `generate`)
- Proved WAL multi-reader visibility with two independently-opened raw connections onto the same bootstrapped file
- Found and fixed a real gap: the SQLite driver creates the database file under the process umask (group/other-readable in practice), not the fixed `0600` the plan assumed — added an explicit `os.Chmod` to close it
- Added `filepath.Clean` on `--store-path` in a `PersistentPreRunE`, so redundant traversal segments never reach `store.Open` or the reported status output
- Wired `golangci-lint` v2 (`gosec` + `errcheck` on top of the default set) and fixed all 9 findings it surfaced across wave-1 and this plan's own new code — zero `nolint` suppressions
- Added a `Makefile` (`test`/`lint`/`build`/`smoke`) and a GitHub Actions workflow at the repository root (`../.github/workflows/nerv-ecosystem-ci.yml`, since Actions only reads workflows from there) running `go test ./... -race -count=1` and `golangci-lint-action@v7` pinned to `v2.12.2` on every push touching `nerv-ecosystem/`

## TDD Gates

| Task | Gate | Commit | Message |
|------|------|--------|---------|
| 1 | RED | `19e92b6` | `test(01-02): add failing reopen, WAL, and FTS5 trigger tests` |
| 1 | GREEN | `15d8568` | `feat(01-02): add AppliedMigrations to prove reopen idempotency` |
| 2 | RED | `31e5347` | `test(01-02): add failing permission, path-cleaning, and failure tests` |
| 2 | GREEN | `19edad8` | `feat(01-02): clean --store-path and tighten db file permissions` |
| 3 | — | `7979a74` | `chore(01-02): enforce -race and lint automatically (PLAT-03)` (config/CI task, no `<behavior>`/tdd gate per plan) |

Gate sequence validated in git log: `test(01-02):` precedes both `feat(01-02):` commits.

### Coverage-only (no RED) — assertions that passed on first run

Per the plan's `tdd_gate_note`, these are regression locks on behavior wave 1 already implemented correctly, not TDD gate violations:

- Reopen idempotency itself: `migrate.go`'s single-transaction version recording (SQL exec + `schema_migrations` insert commit together) already prevented duplicate application — only the missing `AppliedMigrations` read surface caused the initial compile-time RED
- WAL multi-reader visibility: the WAL pragma DSN from wave 1 already gave a second connection visibility into the first's committed write
- FTS5 trigger correctness: `0001_init.sql`'s `AFTER DELETE`/`AFTER UPDATE` triggers already used the `('delete', ...)` special-insert form; the only real bug found was in the *test data* (an unescaped hyphen in a MATCH term, e.g. `"alpha-team"`, which FTS5's query-string grammar interprets as an exclusion operator — fixed by using hyphen-free test terms, not a production change)
- Store directory `0700` permission and unopenable-parent wrapped error: both already correct in wave 1's `store.go`

### Genuine RED → GREEN (production changes required)

- `AppliedMigrations` method did not exist (Task 1)
- `--store-path` was not cleaned before reaching `store.Open` or the status output (Task 2)
- The database file's permission bits included group/other read access — the driver does not default to `0600` under the observed umask (Task 2)

## Files Created/Modified

- `internal/store/migrate_test.go` — reopen idempotency + WAL multi-reader (external `store_test` package)
- `internal/store/fts_internal_test.go` — in-package FTS5 insert/update/delete round-trips via parameterized `MATCH ?`
- `internal/store/store_perms_test.go` — `0700` dir, no group/other file bits, unopenable-parent error
- `cmd/status_failure_test.go` — non-zero exit / no usage dump on failure; `--store-path` cleaning
- `internal/store/migrate.go` — added `MigrationRecord` + `AppliedMigrations`; fixed an `errcheck` finding on `rows.Close()`
- `internal/store/store.go` — added `os.Chmod(path, 0o600)` after `Open`; fixed an `errcheck` finding on `rows.Close()`
- `internal/store/path.go` — `filepath.Clean` on both `DefaultPath` resolution branches
- `cmd/root.go` — `PersistentPreRunE` cleans `--store-path`
- `cmd/status.go` — fixed `errcheck` findings (`st.Close()`, `fmt.Fprintf`)
- `internal/store/store_test.go` (wave 1) — fixed `errcheck` findings on `st.Close()` (Task 3 lint sweep)
- `.golangci.yml` — v2 schema, `run.go: '1.25'`, default set + `gosec` + `errcheck`
- `Makefile` — `test`/`lint`/`build`/`smoke` targets
- `../.github/workflows/nerv-ecosystem-ci.yml` — repo-root CI: `go test ./... -race -count=1` + `golangci-lint-action@v7` (`version: v2.12.2`), triggered on `nerv-ecosystem/**` and the workflow file itself, `go-version-file: nerv-ecosystem/go.mod`, one job, no second non-race test step
- `.gitignore` (new) — keeps the built `modular` binary and `*.db*` artifacts untracked

## Decisions Made

- Added `AppliedMigrations(ctx) ([]MigrationRecord, error)` rather than the plan's literal `([]int, error)` suggestion, so one exported call covers both the row-count and `applied_at`-unchanged assertions without an external test reaching into the unexported `*sql.DB`
- Kept the WAL multi-reader proof in `migrate_test.go` (as directed) but implemented it via two raw `sql.DB` connections opened directly onto the store-bootstrapped file, since no production `projects` writer exists to drive two real `*store.Store` handles — documented inline in the test
- Overrode the plan's "do not chmod, rely on the driver's 0600 default" guidance after the test proved that assumption false; the behavioral requirement (no group/other bits) is what the threat model actually needs, so the implementation detail changed while the security guarantee did not

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Database file was group/other-readable; the plan's "rely on driver default" assumption was wrong**
- **Found during:** Task 2, `TestOpen_StoreDirectoryAndFilePermissions`
- **Issue:** `store_perms_test.go` asserted the database file carries no group/other bits. It failed: `os.Stat` reported `----r--r--` on the group/other bits (mode ~0644), because `modernc.org/sqlite` creates the file under the process's umask, not a fixed `0600` as the plan's action text assumed ("rely on the driver's 0600 creation mode").
- **Fix:** Added `os.Chmod(path, 0o600)` in `store.Open`, applied after migrations succeed and before the `Store` is returned.
- **Files modified:** `internal/store/store.go`
- **Verification:** `TestOpen_StoreDirectoryAndFilePermissions` passes; re-ran `go test ./internal/store/... -race -count=1` — all green
- **Committed in:** `19edad8` (feat)

**2. [Rule 1 - Bug] Test data contained a hyphen that FTS5's query grammar interprets as an exclusion operator**
- **Found during:** Task 1, `fts_internal_test.go` first run
- **Issue:** Using `"alpha-team"` etc. as both stored content and the `MATCH ?` search term produced `SQL logic error: fts5: syntax error near "" (1)` — FTS5's *query-string* parser (not the content tokenizer) treats a bare `-` specially (column filter / NOT), so a hyphenated bareword query fails to parse.
- **Fix:** Changed test fixture team names to hyphen-free values (`alphateam`, `bravoteam`, `charlieteam`, `deltateam`). No production code changed — the trigger SQL was already correct.
- **Files modified:** `internal/store/fts_internal_test.go` (amended into the still-unpushed RED commit before adding the GREEN commit, since it was a same-session test-only correction, not a change to test intent)
- **Verification:** All three FTS5 subtests pass under `-race`
- **Committed in:** amended into `19e92b6` (test)

**3. [Rule 3 - Blocking] `golangci-lint`'s new config surfaced 9 pre-existing `errcheck` findings**
- **Found during:** Task 3, first `golangci-lint run`
- **Issue:** Unchecked `defer x.Close()` and unchecked `fmt.Fprintf` return values across `cmd/status.go`, `internal/store/migrate.go`, `internal/store/store.go`, and three test files (including wave-1's `store_test.go`).
- **Fix:** Wrapped each deferred `Close()` in `defer func() { _ = x.Close() }()`; checked `fmt.Fprintf`'s error in `status.go`'s `RunE`. No `nolint` suppressions used.
- **Files modified:** `cmd/status.go`, `internal/store/migrate.go`, `internal/store/store.go`, `internal/store/store_test.go`, `internal/store/migrate_test.go`, `internal/store/store_perms_test.go`
- **Verification:** `golangci-lint run` → `0 issues`; `go test ./... -race -count=1` still green after the fixes
- **Committed in:** `7979a74` (chore)

**4. [Rule 3 - Blocking] `make smoke`'s built binary and driver-created store directories were left untracked with no `.gitignore`**
- **Found during:** Task 3, after running `make smoke`
- **Issue:** The repo had no `.gitignore`; the `modular` binary produced by `make build`/`make smoke` appeared as an untracked file.
- **Fix:** Added `.gitignore` covering `/modular` and `*.db*` artifacts; removed the built binary from the working tree before committing.
- **Files modified:** `.gitignore` (new)
- **Verification:** `git status --short` shows no untracked build artifacts after `make smoke`
- **Committed in:** `7979a74` (chore)

---

**Total deviations:** 4 auto-fixed (2 Rule 1 - bugs, 2 Rule 3 - blocking/verification-environment). **Impact:** One deviation (#1) corrects a real security-relevant assumption in the plan itself (T-1-02); the rest are test-data or lint/tooling-environment corrections. No scope was reduced and every acceptance criterion in `01-02-PLAN.md` was re-verified passing after fixes.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None — `make smoke` runs with only the local Go toolchain; the CI workflow requires no repository secrets (it only runs `go test` and `golangci-lint-action`, both secret-free).

## Next Phase Readiness

- `go test ./... -race -count=1` is green, `golangci-lint run` reports 0 issues, `make smoke` builds and prints `journal_mode: wal`
- The store's guarantees Phase 2+ will build on are now regression-locked: reopen idempotency, live FTS5 trigger sync (the precondition for Phase 5's SRCH-03), `0700`/`0600` permissions, cleaned developer-supplied paths, and loud (non-zero exit, no usage dump) failures
- `-race` and lint now run automatically on every push touching `nerv-ecosystem/**` via `../.github/workflows/nerv-ecosystem-ci.yml`, so PLAT-03 cannot silently lapse in later phases
- Phase 1 (Platform Foundation) is complete — both plans (`01-01`, `01-02`) executed. Ready for `/gsd-plan-phase 2` (Generate)

## Self-Check: PASSED

All 8 created files and 6 modified files confirmed present on disk via `[ -f ]`. All 5 commit hashes (`19e92b6`, `15d8568`, `31e5347`, `19edad8`, `7979a74`) confirmed present in `git log --oneline --all`. `test(01-02):` commits (`19e92b6`, `31e5347`) both precede their corresponding `feat(01-02):` commits (`15d8568`, `19edad8`) in git log order. Full plan-level `<verification>` block re-run clean: `go test ./... -race -count=1` (PASS), `golangci-lint run` (0 issues), `make smoke` (prints `journal_mode: wal`), failing-path binary invocation (exit 1, no `Usage:` line), driver-created `MODULAR_HOME` directory (mode `700`).

---
*Phase: 01-platform-foundation*
*Completed: 2026-07-24*
