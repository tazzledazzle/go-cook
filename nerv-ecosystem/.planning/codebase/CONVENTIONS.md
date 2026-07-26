# Coding Conventions

**Analysis Date:** 2026-07-25

## Naming Patterns

**Files:**
- `<concern>.go` for production code, named after the noun/behavior it owns: `store.go`, `migrate.go`, `path.go`, `root.go`, `status.go`
- Test files are named after the *behavior under test*, not strictly the production file basename: `store_perms_test.go` and `fts_internal_test.go` both test `store.go`/`migrate.go` behavior but are split by concern (permissions vs. FTS5 sync)

**Functions:**
- Exported constructors return concrete types or interfaces with `New`-prefixed names: `store.Open`, `cmd.NewRootCommand`
- Unexported command factories are `new<Verb>Command`: `newStatusCommand` (`cmd/status.go:15`)
- Test functions: `Test<Subject>_<ExpectedBehavior>` — e.g. `TestOpen_CreatesWALModeStoreWithFTS5Table`, `TestStatusCommand_UnopenableStorePathFailsLoudly`, `TestReopen_DoesNotReapplyMigrations`. The underscore separates the subject from a plain-English description of the expected outcome.

**Variables:**
- Short receiver names: `s *Store`, `st *Store` (varies by file — `store.go`/`migrate.go` use `s`, `cmd/status.go` uses `st` for a locally constructed store)
- `tt` for table-driven test-case loop variables, always re-assigned (`tt := tt`) before `t.Run` for pre-Go-1.22 capture safety even though this project's Go floor (1.25) no longer requires it — kept as an explicit, intentional convention

**Types:**
- Exported struct names are short nouns: `Store`, `MigrationRecord`
- No interfaces defined yet anywhere in the codebase (single-implementation code so far) — see `ARCHITECTURE.md` "Planned Architecture" for when to introduce them

## Code Style

**Formatting:**
- Standard `gofmt` formatting (no custom formatter config found)
- `golangci-lint` v2.12.2 with `linters.default: standard` — this pulls in the standard preset (includes `gofmt`/`goimports`-equivalent checks, `govet`, `staticcheck`, etc.)

**Linting:**
- Config: `.golangci.yml` (`version: "2"` schema)
- `run.go: '1.25'` — matches the module's Go floor
- `linters.default: standard` plus explicitly enabled: `gosec` (security linter), `errcheck` (redundant with `standard` per `01-REVIEW.md` IN-01 — not yet cleaned up)

## Import Organization

**Order:** Standard Go grouping observed consistently — stdlib first, blank line, then third-party/project imports together (not further subdivided into "third-party" vs "internal" groups):
```go
import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/tazzledazzle/go-cook/nerv-ecosystem/internal/store"
)
```
(from `cmd/status.go:3-9`)

**Path Aliases:**
- None used — full module path `github.com/tazzledazzle/go-cook/nerv-ecosystem/...` is written out in every internal import
- Blank import for driver registration: `_ "modernc.org/sqlite"` (`internal/store/store.go:14`)

## Error Handling

**Patterns:**
- Every returned error is wrapped with `fmt.Errorf("<short present-tense description>: %w", err)` — never returned bare. Examples: `"create store dir: %w"`, `"open store: %w"`, `"migrate store: %w"` (`internal/store/store.go`)
- Cleanup/`defer`red calls whose errors are intentionally discarded are explicitly marked `_ = ...` inside a closure, never a bare unchecked call: `defer func() { _ = st.Close() }()` (`cmd/status.go:29`, and throughout test files)
- No panics in application code; all fallible paths return `(T, error)`
- Cobra's `SilenceUsage: true` is set on the root command (`cmd/root.go:24`) so runtime errors don't dump usage text — this is a deliberate, test-locked convention (`cmd/status_failure_test.go` asserts `"Usage:"` is absent from both stdout and stderr on failure)

## Logging

**Framework:** None — no `log`/`slog`/third-party logging library anywhere in the codebase.

**Patterns:**
- All operator-facing output goes through Cobra's `cmd.OutOrStdout()` / `cmd.OutOrStderr()` (not raw `os.Stdout`), which keeps commands testable by swapping in a `bytes.Buffer` (see every test in `cmd/status_test.go`, `cmd/status_failure_test.go`)
- `main.go` is the only place that writes directly to `os.Stderr` (top-level fatal error before exit)

## Comments

**When to Comment:**
- Package-level doc comments state ownership/architectural boundaries explicitly, e.g. `cmd/root.go:1-4` ("Package cmd holds Cobra command wiring only... this package never imports the SQL standard library package or a SQLite driver directly") and `internal/store/store.go:1-4` ("Package store is the sole owner of database/sql and the SQLite schema...")
- Function/method comments explain *why*, not just *what* — e.g. `store.go:23-30`'s `Open` doc comment explains the DSN pragma syntax pitfall (modernc-specific `_pragma=name(value)` vs. the silently-ignored mattn-style `_journal_mode=WAL`)
- Inline comments justify non-obvious decisions at the point of the decision, e.g. `store.go:56-60` explaining why `os.Chmod` can't be left to the driver's default umask behavior, with a pointer to the regression test that proved it (`store_perms_test.go`)
- Test file comments explain *why a test is structured the way it is*, especially when using an internal (`package store`) vs. external (`package store_test`) test package — see `fts_internal_test.go:12-17`

**JSDoc/TSDoc:**
- Not applicable (Go project) — standard Go doc comments (`// FuncName does X`) are used consistently on every exported identifier

## Function Design

**Size:** Small, single-purpose functions — the largest production function (`Open` in `store.go`) is ~35 lines including error handling; most functions are under 20 lines

**Parameters:** Context-first convention for any function touching I/O: `func (s *Store) JournalMode(ctx context.Context) (string, error)`. Config/flags are passed as pointers into command constructors when shared with the parent command (`newStatusCommand(storePath *string)`, `cmd/status.go:15`)

**Return Values:** Consistently `(value, error)` or just `error` — no bare panics, no sentinel zero-value-means-error patterns

## Module Design

**Exports:** Deliberately narrow — `internal/store` exports only `Store`, `Open`, `DefaultPath`, `MigrationRecord`, and methods on `*Store` (`Close`, `JournalMode`, `SchemaObjects`, `HasTable`, `AppliedMigrations`). No `*sql.DB` or `*sql.Rows` ever crosses the package boundary.

**Barrel Files:** Not applicable — Go packages are the natural "barrel"; no re-export indirection files exist.

---

*Convention analysis: 2026-07-25*
