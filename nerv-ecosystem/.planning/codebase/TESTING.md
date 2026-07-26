# Testing Patterns

**Analysis Date:** 2026-07-25

## Test Framework

**Runner:**
- Go stdlib `testing`, no external test runner
- Config: none needed beyond `go.mod`; race detector and count enforced via `Makefile:5-6` (`go test ./... -race -count=1`), never run without `-race` (PLAT-03 — "never split into a separate non-race run")

**Assertion Library:**
- `github.com/stretchr/testify` v1.11.1, `require` subpackage only (no `assert` usage observed) — every failed assertion aborts the test immediately (`require.NoError`, `require.Equal`, `require.Contains`, `require.ErrorIs`, etc.)

**Run Commands:**
```bash
make test               # go test ./... -race -count=1 — always with race detector
go test ./... -race -count=1 -v   # verbose variant for debugging
make lint                # golangci-lint run
make smoke                # build + run `./modular status` against a throwaway MODULAR_HOME
```
No coverage command is wired in `Makefile` today.

## Test File Organization

**Location:**
- Co-located with the production code they test, in the same directory (`cmd/status_test.go` next to `cmd/status.go`; `internal/store/store_test.go` next to `internal/store/store.go`)

**Naming:**
- `<subject>_test.go`, but grouped by *behavior* rather than 1:1 with a single production file — e.g. `store_perms_test.go` (permission regression tests) and `fts_internal_test.go` (FTS5 trigger sync tests) both exercise `store.go`+`migrate.go` but are split into separate files by concern

**Structure:**
```
cmd/
├── status.go
├── status_test.go            # happy path
└── status_failure_test.go    # failure + input-cleaning paths
internal/store/
├── store.go
├── migrate.go
├── store_test.go              # Open/JournalMode/HasTable (package store_test)
├── store_perms_test.go        # file/dir permission bits (package store_test)
├── migrate_test.go            # reopen/no-reapply + WAL multi-reader (package store_test)
└── fts_internal_test.go       # FTS5 trigger sync (package store — white-box)
```

## Test Structure

**Suite Organization:**
```go
func TestStatusCommand_ReportsStoreHealth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
	}{
		{name: "fresh store path via --store-path flag"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// ... arrange, act, assert
		})
	}
}
```
(from `cmd/status_test.go:13-46`)

**Patterns:**
- **Table-driven tests everywhere**, even for a single case — every test in this codebase uses the `tests := []struct{...}{...}` + `for _, tt := range tests { t.Run(tt.name, ...) }` shape, even `TestOpen_CreatesWALModeStoreWithFTS5Table` which only has one table entry. This is a locked convention (PLAT-03), not incidental style.
- `t.Parallel()` called both on the outer test function and inside each `t.Run` subtest closure
- `tt := tt` shadow-copy before `t.Run` is kept as an explicit convention even though Go 1.22+ loop-variable semantics make it technically unnecessary on this project's Go 1.25 floor
- Setup uses real resources, never mocks: `t.TempDir()` for filesystem isolation, real `store.Open(...)` calls against real SQLite files
- Cleanup via `defer func() { _ = st.Close() }()` or `t.Cleanup(func() { _ = s.Close() })`

## Mocking

**Framework:** None. No mocking library (no `gomock`, no `mockery`, no hand-rolled fakes) is used anywhere.

**Patterns:**
- N/A — see "What NOT to Mock" below; this project's explicit test philosophy for Phase 1 is "no mocks for SQLite."

**What to Mock:**
- Nothing yet. When Phase 2+ domain packages introduce interfaces (`store.ProjectRepo`, etc. per `.planning/research/ARCHITECTURE.md`), an in-memory fake implementation is the anticipated pattern for unit-testing domain logic without a real SQLite file — but this does not exist in the codebase today.

**What NOT to Mock:**
- SQLite itself. Locked convention from `01-SKELETON.md`: "No mocks for SQLite — the driver's real behavior (WAL pragma, FTS5 grammar) is exactly what must be proven." Every store test opens a real SQLite file under `t.TempDir()`.

## Fixtures and Factories

**Test Data:**
```go
func newFTSTestStore(t *testing.T) (*Store, context.Context) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "registry.db")
	s, err := Open(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = s.Close() })
	return s, context.Background()
}

func mustExec(t *testing.T, ctx context.Context, s *Store, query string, args ...any) {
	t.Helper()
	_, err := s.db.ExecContext(ctx, query, args...)
	require.NoError(t, err)
}
```
(from `internal/store/fts_internal_test.go:77-90`)

**Location:**
- No separate fixtures/factories package exists — small helper functions like `newFTSTestStore`/`mustExec` are defined directly in the test file that uses them (only `fts_internal_test.go` has helpers so far, since it's the only file needing raw `db.ExecContext` access to seed rows because no production `projects` writer exists yet — Phase 2's `generate` owns that API)

## Coverage

**Requirements:** None enforced today — no coverage threshold in CI or `Makefile`.

**View Coverage:**
```bash
go test ./... -race -count=1 -coverprofile=coverage.out
go tool cover -html=coverage.out
```
(Not currently wired into `Makefile` or CI — would need to be added if coverage tracking is desired.)

## Test Types

**Unit Tests:**
- Store-level (`internal/store/*_test.go`): exercise `Store` methods against real, isolated (`t.TempDir()`) SQLite files — effectively integration tests against the real driver, treated as the project's primary correctness signal for persistence code
- No pure-unit (no-I/O) tests exist yet since every current package touches either the filesystem (store) or Cobra command execution (cmd)

**Integration Tests:**
- `cmd/status_test.go` / `cmd/status_failure_test.go` are CLI-level integration tests: build a full `*cobra.Command` tree via `cmd.NewRootCommand()`, set args, capture stdout/stderr into buffers, call `root.Execute()`, and assert on the printed output — this is the standard pattern for testing any future subcommand
- `internal/store/migrate_test.go`'s `TestWALMultiReader_SecondHandleSeesFirstHandlesWrite` opens two independent raw `*sql.DB` handles onto the same bootstrapped file to prove WAL multi-reader visibility at the SQLite-file level, not just through the `Store` API

**E2E Tests:**
- `make smoke` (`Makefile:14-17`) is the closest thing to an E2E test — builds the real binary and runs `./modular status` against a throwaway `MODULAR_HOME`, but this is a Makefile target, not a Go test

## Common Patterns

**Async Testing:**
- Not applicable — no concurrency in application code beyond the SQLite connection pool itself. `t.Parallel()` is used purely to speed up the test suite, not to test concurrent application behavior (except `TestWALMultiReader_SecondHandleSeesFirstHandlesWrite`, which explicitly exercises concurrent-reader visibility as its subject).

**Error Testing:**
```go
st, err := store.Open(dbPath)
require.Error(t, err)
require.Nil(t, st)
require.Contains(t, err.Error(), "create store dir",
	"error must name the failed operation, not just surface a driver-internal string")
```
(from `internal/store/store_perms_test.go:75-79`) — the convention is to assert both that an error occurred *and* that its wrapped message names the specific failed operation, not just that `err != nil`.

---

*Testing analysis: 2026-07-25*
