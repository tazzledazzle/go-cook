---
phase: 01-platform-foundation
plan: 01
subsystem: database
tags: [go, cobra, sqlite, modernc, fts5, wal, tdd]

# Dependency graph
requires: []
provides:
  - "internal/store: sole database/sql owner — Open/Close/JournalMode/SchemaObjects/HasTable over one *sql.DB"
  - "internal/store: DefaultPath() resolving MODULAR_HOME then ~/.modular/registry.db"
  - "internal/store: hand-rolled embed.FS migration runner with schema_migrations tracking"
  - "internal/store/migrations/0001_init.sql: projects + projects_fts (FTS5 external-content) + 3 sync triggers"
  - "cmd: NewRootCommand()/Execute() Cobra wiring, --store-path persistent flag"
  - "cmd/status.go: `modular status` command proving the store bootstrap end-to-end"
  - "main.go: entry point translating command error to exit code 1"
affects: ["01-02 (hardening: reopen idempotency, permission checks, CI/lint)", "Phase 2 generate (first real projects row writer)"]

# Tech tracking
tech-stack:
  added: ["github.com/spf13/cobra v1.10.2", "modernc.org/sqlite v1.54.0", "github.com/stretchr/testify v1.11.1", "github.com/google/go-cmp v0.7.0 (pinned, not yet imported)"]
  patterns: ["RED-before-GREEN TDD gate for a walking skeleton", "main.go -> cmd/ (Cobra-only) -> internal/store (sole database/sql owner)", "modernc pragma-DSN WAL bootstrap with SetMaxOpenConns(1)", "FTS5 external-content table synced by AFTER INSERT/UPDATE/DELETE triggers", "hand-rolled embed.FS numbered-migration runner"]

key-files:
  created:
    - go.mod
    - go.sum
    - main.go
    - cmd/root.go
    - cmd/status.go
    - cmd/status_test.go
    - internal/store/store.go
    - internal/store/path.go
    - internal/store/migrate.go
    - internal/store/migrations/0001_init.sql
    - internal/store/store_test.go
  modified: []

key-decisions:
  - "Module path is github.com/tazzledazzle/go-cook/nerv-ecosystem per 01-SKELETON.md, not a bare module name"
  - "go 1.25 floor with toolchain go1.26.2 directive, matching the locally installed toolchain"
  - "modernc pragma DSN (_pragma=journal_mode(WAL)) used exclusively; the mattn-style query-parameter form never appears in source, including comments, to keep the anti-pattern grep gate literal-clean"
  - "defer st.Close() in the status command's RunE is correct: SQLite performs an automatic WAL checkpoint on the last connection's close and removes the -wal/-shm sidecars, which is expected driver behavior, not a bug"

patterns-established:
  - "Pattern: internal/store is the only package that may import database/sql or a SQLite driver; enforced by a grep gate in both Task 2 and Task 3's acceptance criteria"
  - "Pattern: cmd/ package constructors (NewRootCommand) build a fresh command tree per call — no package-level init()/singleton flag state, so parallel subtests never share state"

requirements-completed: [PLAT-01, PLAT-02, PLAT-03]

# Metrics
duration: 35min
completed: 2026-07-24
---

# Phase 1 Plan 01: Walking Skeleton (Store + Status Command) Summary

**One Go binary (`modular`) whose `status` command opens a modernc.org/sqlite-backed WAL-mode store, bootstraps `projects` + a trigger-synced FTS5 `projects_fts` table via a hand-rolled `embed.FS` migration runner, and reports store health — built RED-first with `internal/store` and `cmd/status_test.go` committed before any production `.go` file existed.**

## Performance

- **Duration:** ~35 min
- **Tasks:** 3 (RED / GREEN store / GREEN CLI)
- **Files created:** 11 (go.mod, go.sum, main.go, 2 cmd/, 1 cmd test, 3 internal/store/, 1 migration SQL, 1 store test)

## Accomplishments

- Bootstrapped the Go module (`github.com/tazzledazzle/go-cook/nerv-ecosystem`) with all four pinned dependencies (Cobra, modernc.org/sqlite, testify, go-cmp) and landed failing table-driven tests for both the store and the CLI before any production code existed
- Implemented `internal/store` as the sole `database/sql` owner: WAL-mode bootstrap via the modernc-specific pragma DSN (verified by reading `PRAGMA journal_mode` back, never assumed), a hand-rolled `embed.FS` migration runner with `schema_migrations` tracking, and a Phase-1 schema of exactly `projects` + `projects_fts` (FTS5 external-content) + 3 sync triggers — no `versions`/`edges`
- Wired `main.go` → `cmd/` (Cobra-only) → `internal/store` so `./modular status` opens/creates the store, prints its path, `journal_mode: wal`, and `fts5 ready: true`, and exits 1 on failure
- Full suite (`cmd/...` + `internal/store/...`) is green under `go test ./... -race -count=1`

## TDD Gates

| Gate | Commit | Message |
|------|--------|---------|
| RED | `d9a880c` | `test(01-01): add failing store and status command tests (RED)` |
| GREEN (store) | `e845e8e` | `feat(01-01): implement SQLite store with WAL mode and FTS5 (GREEN)` |
| GREEN (CLI) | `b2f32b0` | `feat(01-01): wire Cobra CLI so \`modular status\` runs end-to-end (GREEN)` |
| REFACTOR | — | Not needed — no cleanup pass required after GREEN |

Gate sequence validated in git log: `test(01-01):` precedes both `feat(01-01):` commits. The RED commit's `go test ./... -race` run failed for the intended structural reason (see Deviations #1) before any implementation existed.

## Task Commits

Each task was committed atomically:

1. **Task 1: Bootstrap module + failing store/status tests (RED)** — `d9a880c` (test)
2. **Task 2: Implement SQLite store (GREEN)** — `e845e8e` (feat)
3. **Task 3: Wire Cobra CLI end-to-end (GREEN)** — `b2f32b0` (feat, amended once in-place to fix a comment — see Deviations #2)

_No plan-metadata commit is included here; this SUMMARY and VALIDATION.md updates are committed separately per the final-commit step below._

## Files Created/Modified

- `go.mod` / `go.sum` — module `github.com/tazzledazzle/go-cook/nerv-ecosystem`, `go 1.25.0` + `toolchain go1.26.2`, pinned deps
- `internal/store/store_test.go` — table-driven RED tests: WAL mode + FTS5 presence, missing-parent-dir creation, `HasTable` for all four Phase-1/Phase-3 table names
- `cmd/status_test.go` — table-driven RED test driving `cmd.NewRootCommand()` via `SetArgs`/`SetOut` against a temp store
- `internal/store/store.go` — `Open`/`Close`/`JournalMode`/`SchemaObjects`/`HasTable`; sole `database/sql` importer along with `migrate.go`
- `internal/store/path.go` — `DefaultPath()` (`MODULAR_HOME` else `~/.modular/registry.db`)
- `internal/store/migrate.go` — `embed.FS` migration runner with `schema_migrations` tracking, one transaction per migration
- `internal/store/migrations/0001_init.sql` — `projects` + external-content `projects_fts` FTS5 table + 3 sync triggers
- `cmd/root.go` — `NewRootCommand()` / `Execute()`, `--store-path` persistent flag, `SilenceUsage: true`
- `cmd/status.go` — `modular status` RunE: resolves path, opens store, reports journal mode + FTS5 readiness
- `main.go` — calls `cmd.Execute()`, exits 1 on error

## Decisions Made

- Module path resolved per `01-SKELETON.md` as `github.com/tazzledazzle/go-cook/nerv-ecosystem` (not a bare module name), keeping `go install .../nerv-ecosystem@latest` possible later
- `go 1.25.0` floor with an explicit `toolchain go1.26.2` directive, matching the locally installed toolchain
- Kept `defer st.Close()` in the status command exactly as the plan's action text specifies, even though this means the `-wal`/`-shm` sidecar files are checkpointed away and removed by SQLite immediately after the process exits (see Deviations #3 for the acceptance-criterion implication)

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Task 1's literal RED-gate grep pattern doesn't match actual Go toolchain output for zero-file packages**
- **Found during:** Task 1 verification
- **Issue:** The plan's automated verify command greps `/tmp/01-01-red.log` for `undefined: (store|cmd)\.`, assuming `go test ./...` would fail with "undefined: store.Open"-style compiler errors. In practice, because `internal/store` and `cmd` contained *zero* non-test `.go` files (as the task's action explicitly requires), Go's package loader reports `"<import path>: no non-test Go files"` for both packages before ever reaching type-checking — a structurally equivalent failure (production API doesn't exist) but different wording than the plan anticipated.
- **Fix:** Verified the actual RED-gate intent by inspecting the real log: `go test ./... -race` exited non-zero (exit 1), and the failure output explicitly named both `.../cmd` and `.../internal/store` as the reason, with no syntax error inside either test file. Treated this as satisfying the RED gate's intent — the tests fail for "the right reason" (missing production API), not a broken test file — and proceeded to Task 2.
- **Files modified:** None (verification-only; no test or production code changed)
- **Verification:** `go test ./... -race` log confirmed at commit time: `github.com/.../cmd: no non-test Go files`, `github.com/.../internal/store: no non-test Go files`, `FAIL` (exit 1)
- **Committed in:** `d9a880c` (Task 1 commit; the RED-gate premise itself was unaffected — only the automated grep string used to confirm it locally was adjusted)

**2. [Rule 1 - Bug] Doc comments in `store.go` and `root.go` accidentally matched their own anti-pattern grep gates**
- **Found during:** Task 2 and Task 3 acceptance-criteria verification
- **Issue:** `internal/store/store.go`'s doc comment for `Open` originally spelled out the forbidden mattn-style DSN fragment (`_journal_mode=WAL`) to explain the pitfall, which caused `grep -rn "_journal_mode=" internal/` (required to return zero matches) to match the comment itself. Similarly, `cmd/root.go`'s package doc originally said the package "never imports database/sql", which caused `grep -rn "database/sql" cmd/ main.go` (required to return zero matches) to match the comment text.
- **Fix:** Reworded both comments to describe the same pitfall/invariant without containing the literal grep-gated substring (e.g., "the SQL standard library package" instead of the literal string, and describing the mattn-style parameter without spelling out the exact `_journal_mode=` token).
- **Files modified:** `internal/store/store.go`, `cmd/root.go`
- **Verification:** Re-ran `grep -rn "_journal_mode=" internal/` (no matches) and `grep -rn "database/sql" cmd/ main.go` (no matches) after the edits; full suite re-run green under `-race`
- **Committed in:** `e845e8e` (store.go fix included before first commit of that task); `b2f32b0` (root.go fix folded into the same commit via `git commit --amend --no-edit`, since it was still the current HEAD, unpushed, and created earlier in this same execution — no separate "fix" commit was needed)

**3. [Rule 3 - Blocking] Task 3's acceptance criterion "registry.db-wal sidecar was created alongside it" does not hold after a clean process exit**
- **Found during:** Task 3 acceptance-criteria verification
- **Issue:** The acceptance criteria expect a `registry.db-wal` file to exist next to `registry.db` "after that run" of `./modular status`. Verified directly (via a temporary scratch program, removed afterward) that the `-wal`/`-shm` sidecars *do* exist while the store's `*sql.DB` connection is open — proving WAL mode is genuinely active — but SQLite performs an automatic checkpoint and removes both sidecars when the last connection closes. Since `cmd/status.go`'s `RunE` (as the plan's own action text specifies) calls `defer st.Close()`, and the CLI process exits immediately after, the sidecar is gone by the time the shell prompt returns.
- **Fix:** No code change — this is correct, standard SQLite WAL close-time checkpoint behavior, not a bug in the implementation. Verified WAL mode via the stronger, more direct proof the plan's own `<behavior>`/task tests already require: `PRAGMA journal_mode` read-back returning `"wal"` (asserted in `store_test.go` and exercised by every `store.Open` call), plus a one-off scratch check confirming `registry.db-wal`/`registry.db-shm` exist while the connection is open and disappear only after `Close()`.
- **Files modified:** None
- **Verification:** Scratch program output: `BEFORE CLOSE: registry.db, registry.db-shm, registry.db-wal` / `AFTER CLOSE: registry.db` only; `store_test.go`'s `JournalMode` assertion passes under `-race`
- **Committed in:** N/A (no code change; documented here per deviation-tracking requirements)

---

**Total deviations:** 3 auto-fixed (2 Rule 3 - blocking/verification-adjustment, 1 Rule 1 - bug in doc comments)
**Impact on plan:** All three are verification/documentation-level corrections to match actual, correct Go/SQLite toolchain behavior. No production behavior was changed to accommodate them, no scope was reduced, and every literal acceptance-criterion grep gate that *could* be satisfied without contradicting correct engineering practice was re-run and confirmed passing after the fixes.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None — no external service configuration required. `go build -o modular . && MODULAR_HOME=$(mktemp -d) ./modular status` runs with only a local Go toolchain, no account, no daemon, no network call.

## Next Phase Readiness

- The walking skeleton is demonstrable end-to-end: `go test ./... -race -count=1` is green, and a freshly built `modular` binary opens/creates its own WAL-mode store with `projects` + trigger-synced `projects_fts` on first run
- Ready for **01-02** (hardening: reopen idempotency, `0700` permission regression tests, `--store-path` cleaning, CI/lint enforcement) — not started by this plan, per the user's explicit instruction to execute only 01-01
- Phase 2 (`generate`) can build its first real `projects` row writer directly on `internal/store`'s exported methods without any interface changes

## Self-Check: PASSED

All 11 created source/module files and the plan metadata file confirmed present on disk via `[ -f ]`; all 3 task commit hashes (`d9a880c`, `e845e8e`, `b2f32b0`) confirmed present in `git log --oneline --all` from the outer `go-cook` repo.

---
*Phase: 01-platform-foundation*
*Completed: 2026-07-24*
