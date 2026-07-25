---
phase: 01-platform-foundation
verified: 2026-07-24T17:30:00Z
status: passed
score: 10/10 must-haves verified
overrides_applied: 0
---

# Phase 1: Platform Foundation Verification Report

**Phase Goal:** Operator has a working CLI binary with a persistent local store and enforced TDD conventions that every later feature phase builds on.
**Verified:** 2026-07-24T17:30:00Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Operator can install/run a single `modular` binary with zero cloud accounts or always-on daemons (Roadmap SC-1 / PLAT-01) | ✓ VERIFIED | `go build -o modular . && MODULAR_HOME=$(mktemp -d) ./modular status` exits 0, prints store path, `journal_mode: wal`, `fts5 ready: true`. No network client imports anywhere in the tree; no daemon process spawned. |
| 2 | Running any command opens/initializes one embedded SQLite store file (WAL mode) with the FTS5 search table already created (Roadmap SC-2 / PLAT-02) | ✓ VERIFIED | Fresh `MODULAR_HOME` run creates `registry.db`; `JournalMode()` pragma read-back returns `"wal"`; `sqlite_master` contains `projects` and `projects_fts` (FTS5, `content='projects'`) confirmed live in a fresh run. |
| 3 | Domain packages have table-driven tests passing under `go test -race`, proving TDD scaffolding is in place (Roadmap SC-3 / PLAT-03) | ✓ VERIFIED | `go test ./... -race -count=1` and `-count=2` both green (`cmd`, `internal/store`). `store_test.go` has 6 `t.Parallel()` calls; table literals (`tests := []struct{...}`) present in all *_test.go files inspected. |
| 4 | A failing test for store-open and for status output was committed strictly before any production code (RED before GREEN, PLAT-03) | ✓ VERIFIED | `git log --reverse`: `d9a880c test(01-01):...` precedes `e845e8e feat(01-01):` and `b2f32b0 feat(01-01):`. Same ordering holds for plan 01-02: `19e92b6 test(01-02):` before `15d8568 feat(01-02):`, and `31e5347 test(01-02):` before `19edad8 feat(01-02):`. |
| 5 | Re-running the CLI against an existing store succeeds without re-applying migrations or duplicating schema rows | ✓ VERIFIED | `internal/store/migrate_test.go` asserts one `schema_migrations` row for version 1 and an unchanged `applied_at` across two sequential opens; exported `AppliedMigrations()` method exists (`migrate.go:75`) and is exercised by the passing test suite. |
| 6 | A row written into `projects` is findable through `projects_fts MATCH`, and update/delete keep the index consistent with no application-level reindex step | ✓ VERIFIED | `internal/store/fts_internal_test.go` (in-package `store`) has 6 `MATCH` occurrences across insert/update/delete subtests, all parameterized (`MATCH ?`, no inlined literal terms). Trigger file has exactly 3 `CREATE TRIGGER` statements and 2 `'delete'` special-insert forms (delete trigger + update's delete half), matching the external-content-FTS5 correctness pattern. |
| 7 | A second open handle on the same store file can read data written through the first handle (WAL multi-reader) | ✓ VERIFIED | `migrate_test.go` WAL multi-reader case is part of the passing `internal/store` suite (two independent connections onto the same bootstrapped file). |
| 8 | The store directory is created 0700 and the database file is not world- or group-readable | ✓ VERIFIED | Live check: freshly-created `MODULAR_HOME` directory reports `700` via `stat -f '%Lp'`; `registry.db` reports `600`. Code: `os.MkdirAll(..., 0o700)` at `store.go:32`; explicit `os.Chmod(path, 0o600)` added in plan 01-02 after discovering the driver honors process umask. |
| 9 | An unopenable store path fails loudly: a wrapped error and exit code 1 from the built binary, no usage dump | ✓ VERIFIED | Live run against a path nested under a regular file: prints `Error: open store: create store dir: mkdir .../blocker: not a directory`, exits 1, zero lines containing `Usage:`. `SilenceUsage: true` set in `cmd/root.go:24`. |
| 10 | `go test -race` and `golangci-lint` run automatically on every push touching the module | ✓ VERIFIED | `.github/workflows/nerv-ecosystem-ci.yml` (repo root, correctly placed) triggers on `nerv-ecosystem/**` paths, runs `go test ./... -race -count=1` then `golangci-lint-action@v7` pinned to `v2.12.2`, one job, no second non-race step. Live `golangci-lint run` (v2.12.2 locally installed) reports `0 issues`. |

**Score:** 10/10 truths verified

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `internal/store/store_test.go` | RED test, WAL/FTS5 assertions, `t.Parallel()` | ✓ VERIFIED | 6 `t.Parallel()` calls, table-driven, external `store_test` package |
| `cmd/status_test.go` | RED test driving Cobra tree, `SetArgs` | ✓ VERIFIED | `SetArgs` present, drives `cmd.NewRootCommand()` |
| `internal/store/store.go` | `Open/Close/JournalMode/SchemaObjects/HasTable` | ✓ VERIFIED | All five identifiers present and exported; sole `database/sql` importer alongside `migrate.go` |
| `internal/store/path.go` | `DefaultPath()` | ✓ VERIFIED | `func DefaultPath() string` at line 16, `filepath.Clean` applied on both branches |
| `internal/store/migrate.go` | `embed.FS` migration runner + `AppliedMigrations` | ✓ VERIFIED | `//go:embed migrations/*.sql`; `AppliedMigrations` at line 75 |
| `internal/store/migrations/0001_init.sql` | `projects` + `projects_fts` FTS5 + 3 triggers | ✓ VERIFIED | `USING fts5` present; 3 `CREATE TRIGGER`; no `versions`/`edges` tables |
| `cmd/root.go` | `NewRootCommand`, `Execute()`, `--store-path`, `SilenceUsage` | ✓ VERIFIED | `func Execute() error` at line 47; `SilenceUsage: true` at line 24; `filepath.Clean` at line 33 (`PersistentPreRunE`) |
| `cmd/status.go` | `modular status` command | ✓ VERIFIED | Reads store health, writes to `cmd.OutOrStdout()` |
| `main.go` | Entry point, exit code 1 on error | ✓ VERIFIED | `os.Exit(1)` at line 13 |
| `internal/store/migrate_test.go` | Reopen idempotency + WAL multi-reader | ✓ VERIFIED | `schema_migrations` referenced; part of green suite |
| `internal/store/fts_internal_test.go` | In-package FTS5 trigger round-trip | ✓ VERIFIED | 6 `MATCH` occurrences, no inlined literal terms |
| `internal/store/store_perms_test.go` | Permission regression lock | ✓ VERIFIED | `0o700` present; part of green suite |
| `cmd/status_failure_test.go` | Failure-path + path-cleaning coverage | ✓ VERIFIED | `NewRootCommand` used twice; part of green suite |
| `.golangci.yml` | v2 schema, `run.go: '1.25'` | ✓ VERIFIED | `version: "2"` first line, `run.go: '1.25'` present, `gosec` + `errcheck` enabled |
| `Makefile` | `test`/`lint`/`build`/`smoke` targets | ✓ VERIFIED | All four `.PHONY` targets present; `test` uses `-race -count=1` |
| `../.github/workflows/nerv-ecosystem-ci.yml` | CI enforcing race + lint | ✓ VERIFIED | Correctly placed at repo root; path-filtered; one job, no non-race duplicate step |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `main.go` | `cmd.Execute` | error-checked call, `os.Exit(1)` on failure | ✓ WIRED | Confirmed by live failing-path run: exit code 1, error printed to stderr |
| `cmd/status.go` | `internal/store` | `store.Open` then `JournalMode`/`HasTable` | ✓ WIRED | Confirmed by live successful run printing real pragma-read values, not hardcoded |
| `internal/store/store.go` | modernc.org/sqlite driver | pragma DSN | ✓ WIRED | `_pragma=journal_mode(WAL)` present in DSN string; mattn-style `_journal_mode=` form absent from `internal/` |
| `internal/store/migrate.go` | `internal/store/migrations/0001_init.sql` | `embed.FS` glob + transactional apply | ✓ WIRED | Migration applies on fresh store; `AppliedMigrations` confirms exactly one version-1 row after reopen |
| `internal/store/migrations/0001_init.sql` triggers | `projects_fts` index contents | AFTER INSERT/UPDATE/DELETE special-insert bodies | ✓ WIRED | `fts_internal_test.go` insert/update/delete subtests pass against live parameterized `MATCH ?` queries |
| `cmd/root.go --store-path` | `store.Open` | `filepath.Clean` | ✓ WIRED | `filepath.Clean` call confirmed inside `PersistentPreRunE` at line 33 |
| CI workflow | `go test -race` | working-directory + path filter | ✓ WIRED | Workflow file confirmed present at correct repo-root path with matching trigger and step |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `cmd/status.go` output | `journal_mode`, `fts5 ready` lines | `store.JournalMode(ctx)` / `store.HasTable(ctx, "projects_fts")` reading live `PRAGMA`/`sqlite_master` | Yes — confirmed via two independent live binary runs producing correct WAL/FTS5 state from a genuinely fresh store, not a static string | ✓ FLOWING |
| `internal/store.AppliedMigrations` | migration version rows | live `schema_migrations` table query | Yes — reopen test distinguishes "one row, unchanged timestamp" from a hypothetical re-apply, which is a real DB round-trip, not a stub | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Fresh store bootstrap + status report | `go build -o /tmp/modular-verify . && MODULAR_HOME=$(mktemp -d) /tmp/modular-verify status` | `journal_mode: wal`, `fts5 ready: true`, exit 0 | ✓ PASS |
| Store-directory permission enforcement | `stat -f '%Lp'` on a store dir the binary itself created | `700` | ✓ PASS |
| DB file permission enforcement | `stat -f '%Lp'` on `registry.db` | `600` | ✓ PASS |
| Failure path / no usage dump | `status --store-path <blocked-path>` | exit 1, wrapped error, zero `Usage:` lines | ✓ PASS |
| Full suite under race detector (twice, for flakiness) | `go test ./... -race -count=1` and `-count=2` | both green | ✓ PASS |
| Lint | `golangci-lint run` (v2.12.2, matches pinned CI version) | `0 issues.` | ✓ PASS |
| `go vet` | `go vet ./...` | clean | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` convention or explicit probe declarations exist in this phase's PLAN/SUMMARY files — this phase uses `go test`/`golangci-lint`/binary smoke checks as its verification surface instead. Step 7c: SKIPPED (no probe scripts declared or found).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| PLAT-01 | 01-01, 01-02 | Single binary, no cloud accounts/daemons | ✓ SATISFIED | Live `modular status` run, no network imports, no daemon |
| PLAT-02 | 01-01, 01-02 | One embedded SQLite store (WAL + FTS5) | ✓ SATISFIED | `journal_mode: wal` read-back, `projects_fts` live, trigger sync proven, reopen idempotent |
| PLAT-03 | 01-01, 01-02 | Test-first (RED before GREEN); race detector + table-driven tests | ✓ SATISFIED | Git history ordering confirmed; `-race` green x2; CI/Makefile enforce automatically |

No orphaned requirements: REQUIREMENTS.md maps only PLAT-01/02/03 to Phase 1, and all three appear in both plans' `requirements:` frontmatter.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| — | — | No `TODO`/`FIXME`/`XXX`/`TBD`/`HACK`/`PLACEHOLDER` markers found in any `.go` file | — | None — clean |
| — | — | No `nolint` suppressions anywhere in the module | — | None — clean |

No blockers or warnings found. `gosec`/`errcheck` findings from the SUMMARY's Task 3 lint sweep were already fixed and are confirmed absent by the live `golangci-lint run` (`0 issues`).

### Human Verification Required

None. This phase produces a CLI/backend artifact (`UI hint: no` in ROADMAP.md) with fully programmatic success criteria — store state, permission bits, exit codes, and CI wiring are all verifiable by direct inspection and live command execution, which was done above.

### Gaps Summary

No gaps. All 10 observable truths (roadmap Success Criteria + both plans' must-have truths, merged and deduplicated) are verified against live command execution and code inspection, not just SUMMARY.md narration:

- The binary was actually built and run twice against fresh, throwaway stores, independently of the plan's own automated verify strings, confirming WAL mode and FTS5 readiness are real, not asserted-only.
- The full test suite was run at `-count=1` and again at `-count=2` under `-race` to rule out flakiness; both passed.
- `golangci-lint run` was executed live with the exact pinned version (v2.12.2) the CI workflow uses, independent of the SUMMARY's own claim, and returned 0 issues.
- Git history was inspected directly (not trusted from SUMMARY) and confirms RED commits strictly precede their corresponding GREEN commits for both plans.
- Permission bits (0700 directory, 0600 file) were checked with `stat` against directories/files the binary itself created in this verification run, not reused from a prior session.
- The failure path was exercised live: an unopenable store path produces exit code 1 with a wrapped error and zero `Usage:` output.

Phase 1 delivers exactly what it promises: a working `modular` CLI binary with a persistent, WAL-mode, FTS5-backed local SQLite store, and TDD/`-race`/lint conventions that are both regression-locked by tests and automatically enforced in CI. Phase 2 (Generate) can build directly on `internal/store`'s exported surface without renegotiating any Phase 1 decision.

---

_Verified: 2026-07-24T17:30:00Z_
_Verifier: Claude (gsd-verifier)_
