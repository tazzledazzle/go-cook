---
phase: 01-platform-foundation
reviewed: 2026-07-24T17:16:00Z
depth: standard
files_reviewed: 15
files_reviewed_list:
  - nerv-ecosystem/main.go
  - nerv-ecosystem/go.mod
  - nerv-ecosystem/cmd/root.go
  - nerv-ecosystem/cmd/status.go
  - nerv-ecosystem/cmd/status_test.go
  - nerv-ecosystem/cmd/status_failure_test.go
  - nerv-ecosystem/internal/store/store.go
  - nerv-ecosystem/internal/store/migrate.go
  - nerv-ecosystem/internal/store/path.go
  - nerv-ecosystem/internal/store/migrations/0001_init.sql
  - nerv-ecosystem/internal/store/store_test.go
  - nerv-ecosystem/internal/store/migrate_test.go
  - nerv-ecosystem/internal/store/fts_internal_test.go
  - nerv-ecosystem/internal/store/store_perms_test.go
  - nerv-ecosystem/Makefile
  - nerv-ecosystem/.golangci.yml
  - .github/workflows/nerv-ecosystem-ci.yml
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 1: Code Review Report

**Reviewed:** 2026-07-24T17:16:00Z
**Depth:** standard
**Files Reviewed:** 15 non-test/config source files (plus 4 test files inspected for reliability only)
**Status:** issues_found (advisory only, no blockers)

## Summary

Reviewed the Phase 1 (Platform Foundation) walking-skeleton implementation: `internal/store` (SQLite/WAL/FTS5 store + migration runner), `cmd`/`main.go` (Cobra CLI wiring), the Phase-1 migration SQL, and the build/lint/CI harness (`Makefile`, `.golangci.yml`, `.github/workflows/nerv-ecosystem-ci.yml`). Independently re-ran `go build ./...`, `go vet ./...`, `go test ./... -race -count=1`, and `golangci-lint run` (v2.12.2, matching the pinned CI version) — all clean, confirming the SUMMARY/VERIFICATION claims.

The implementation is solid: `internal/store` is genuinely the sole `database/sql` owner, WAL mode and FTS5 sync are read back rather than assumed, permissions are regression-locked by tests, and TDD gate ordering holds in git history. No security or correctness defects were found that rise to blocker severity.

Adversarial testing did surface one real, reproducible correctness edge case in DSN construction (WR-01) that the existing test suite does not cover, plus a minor TOCTOU permission-tightening gap (WR-02) and two small quality nits. None of these block Phase 2; they are flagged for awareness and optional follow-up.

## Warnings

### WR-01: Unescaped `?` in a store path silently disables WAL mode with no error

**File:** `nerv-ecosystem/internal/store/store.go:36-40`
**Issue:** The DSN is built with `fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)`. `path` (developer-supplied via `--store-path`, only `filepath.Clean`-ed, never otherwise validated/escaped) is spliced directly ahead of the DSN's own `?` query-string delimiter. If `path` itself contains a literal `?` (a valid POSIX filename character), the driver's DSN parser sees two `?` characters and misparses the query string, so **none of the three pragmas are applied** — no error is returned, and `sql.Open`/`db.Open` succeed normally. Reproduced independently outside the test suite:
```
path := "/tmp/dsntest/weird?name.db"
dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", path)
db, _ := sql.Open("sqlite", dsn)
db.QueryRow("PRAGMA journal_mode").Scan(&mode) // mode == "delete", not "wal"
```
This directly undermines the store's own documented philosophy ("verified by reading `PRAGMA journal_mode` back, never assumed") for this one input class — the read-back would correctly report `"delete"`, so `modular status` wouldn't lie about it, but the operator gets a silently non-WAL, non-FTS5-safe-durability store with zero error surfaced, purely because of an unescaped path character. (Confirmed the driver tolerates spaces, `&`, and `#` in the path without issue — only `?` triggers the misparse, since it collides with the DSN's own delimiter.)
**Fix:** Percent-encode the path segment before splicing it into the DSN, e.g. via `url.URL{Scheme: "file", Opaque: path}` construction or `strings.ReplaceAll(path, "?", "%3F")`, or reject/validate paths containing `?` up front in `cmd/root.go`'s `PersistentPreRunE`. Simplest fix:
```go
import "net/url"

dsn := fmt.Sprintf(
    "file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)",
    url.PathEscape(path), // escapes '?', '#', etc. that collide with DSN syntax
)
```
(Verify `url.PathEscape` round-trips correctly with this driver's path decoding before relying on it — some SQLite Go drivers expect a raw filesystem path rather than a percent-decoded one. A regression test asserting `journal_mode() == "wal"` for a path containing `?` would catch this permanently.)

### WR-02: Brief TOCTOU window where the store file is created world/group-readable before being chmod'd to 0600

**File:** `nerv-ecosystem/internal/store/store.go:31-66`
**Issue:** `Open` calls `sql.Open` (which lazily creates the file under the process umask on first write, typically `0644`) and runs migrations *before* calling `os.Chmod(path, 0o600)` at the end. Between file creation (inside `runMigrations`) and the explicit `Chmod` call, the database file exists on disk with default, umask-derived permissions (group/other-readable in the exact scenario `store_perms_test.go` was written to catch). On a genuinely multi-user machine this is a real, if narrow, exposure window for a file the project explicitly treats as sensitive enough to warrant a dedicated permission regression test.
**Fix:** Pre-create the file with the desired mode before handing it to the driver, e.g.:
```go
if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
    return nil, fmt.Errorf("create store dir: %w", err)
}
f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
if err != nil {
    return nil, fmt.Errorf("create store file: %w", err)
}
_ = f.Close() // file now exists with 0600 before sql.Open ever touches it
```
Then keep the existing post-migration `os.Chmod` call as a defense-in-depth belt-and-suspenders check (cheap, and it also covers the pre-existing-file-with-loose-perms case that `OpenFile` alone won't fix).

## Info

### IN-01: `errcheck` is explicitly enabled even though it's already part of the `standard` default linter set

**File:** `nerv-ecosystem/.golangci.yml:6-10`
**Issue:** `linters.default: standard` already includes `errcheck` in golangci-lint v2's standard preset; the explicit `enable: [gosec, errcheck]` re-lists it redundantly. Harmless, but slightly misleading about which linter the `enable` block is actually adding (only `gosec` is new).
**Fix:** Drop `errcheck` from the `enable` list (or add a comment noting it's already included, if the redundancy is intentional for clarity/explicitness).

### IN-02: Migration version parsing/sorting assumes a permanent 4-digit zero-padding convention with no validation

**File:** `nerv-ecosystem/internal/store/migrate.go:28-37`
**Issue:** `sort.Strings(entries)` (lexical sort of filenames) relies on every future migration file being zero-padded to exactly 4 digits (as the comment on line 32 acknowledges: `"0001_init.sql" < "0002_....sql" lexically`). If that convention is ever broken (e.g., a migration numbered without matching zero-padding width, or the count exceeds `9999`), lexical sort and numeric migration order silently diverge, and `fmt.Sscanf(name, "migrations/%04d_", &version)`'s width specifier further truncates version parsing at 4 digits without any comment documenting that ceiling. Not a bug today (only one migration exists), but it's an undocumented invariant with no assertion enforcing it.
**Fix:** Either add a brief comment/constant documenting the 4-digit width ceiling as an explicit, intentional constraint (e.g. `const migrationVersionWidth = 4 // supports up to 9999 migrations, zero-padded`), or replace the lexical sort with an explicit numeric sort by parsed version after parsing all filenames, removing the ordering's dependence on padding width entirely.

---

_Reviewed: 2026-07-24T17:16:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
