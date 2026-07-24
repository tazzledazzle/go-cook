---
phase: 1
slug: platform-foundation
status: in-progress
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-24
updated: 2026-07-24
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go stdlib `testing` + `testify` (require/assert) v1.11.1 |
| **Config file** | none — `go.mod` alone |
| **Quick run command** | `go test ./internal/... -race` |
| **Full suite command** | `go test ./... -race -count=1` |
| **Estimated runtime** | ~5–15 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/... -race`
- **After every plan wave:** Run `go test ./... -race -count=1`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 30 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 01-01-01 | 01-01 | 1 | PLAT-03 | T-1-SC | pinned deps + `go mod verify`; RED gate before any production code (this task **is** Wave 0) | setup + RED | `go mod verify && { go test ./... -race > /tmp/01-01-red.log 2>&1; test $? -ne 0; } && grep -qE "undefined: (store\|cmd)\." /tmp/01-01-red.log` | ✅ `internal/store/store_test.go`, `cmd/status_test.go` | ✅ green (commit d9a880c; verify command adjusted — see 01-01-SUMMARY.md deviations) |
| 01-01-02 | 01-01 | 1 | PLAT-02 | T-1-01, T-1-02, T-1-03, T-1-04 | parameterized SQL only; store dir `0o700`; WAL pragma read back rather than assumed; `SetMaxOpenConns(1)` | unit | `go test ./internal/store/... -race -count=1` | ✅ `internal/store/{store,path,migrate}.go`, `migrations/0001_init.sql` | ✅ green (commit e845e8e) |
| 01-01-03 | 01-01 | 1 | PLAT-01 | T-1-03 | no network/daemon; `cmd/` imports no `database/sql`; store failure returns a wrapped error, not a panic | integration | `go test ./... -race -count=1 && go build -o /tmp/modular-01-01 . && MODULAR_HOME=$(mktemp -d) /tmp/modular-01-01 status \| grep -q "journal_mode: wal"` | ✅ `main.go`, `cmd/{root,status}.go` | ✅ green (commit b2f32b0) |
| 01-02-01 | 01-02 | 2 | PLAT-02 | T-1-01, T-1-03, T-1-06 | `MATCH ?` bound parameter; `('delete', …)` special-insert trigger form; reopen applies no duplicate migration | unit | `go test ./internal/store/... -race -count=1` | ❌ created here | ⬜ pending |
| 01-02-02 | 01-02 | 2 | PLAT-01 | T-1-02, T-1-05 | store dir exactly `0700`, db file has no group/other bits; `filepath.Clean` on `--store-path`; exit code 1 and no usage dump on failure | integration | `go test ./... -race -count=1 && go build -o /tmp/modular-01-02 . && D=$(mktemp -d) && printf x > "$D/blocker" && ! /tmp/modular-01-02 status --store-path "$D/blocker/nested/registry.db" && H=$(mktemp -d)/created-by-store && MODULAR_HOME=$H /tmp/modular-01-02 status >/dev/null && { stat -f '%Lp' "$H" 2>/dev/null \|\| stat -c '%a' "$H"; } \| grep -qx 700` | ❌ created here | ⬜ pending |
| 01-02-03 | 01-02 | 2 | PLAT-03 | T-1-07 | `gosec` enabled; `-race` + lint run on every push; no unexplained `nolint` | lint + smoke | `golangci-lint run && make test && make smoke` | ❌ created here | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

**Nyquist:** every task above carries an `<automated>` verify — no three consecutive tasks lack automated
feedback. Wave 0 (dependency install + failing test stubs) is folded into task `01-01-01`, which must land a
`test(01-01):` commit before any production `.go` file exists. All commands run from the module root
(`nerv-ecosystem/`).

---

## Wave 0 Requirements

Delivered by task `01-01-01` (Viper is deliberately excluded — RESEARCH defers it to Phase 2 because Phase 1 has
no config surface to layer):

- [x] `go.mod` / `go.sum` with testify v1.11.1, modernc.org/sqlite v1.54.0, cobra v1.10.2, go-cmp v0.7.0
- [x] `internal/store/store_test.go` — PLAT-02 (WAL mode + FTS5 table existence)
- [x] `cmd/status_test.go` — PLAT-01 (Cobra `SetArgs`/`SetOut`/`Execute` against a temp store)

Follow-on test files created in wave 2 (plan 01-02), not part of Wave 0:

- [ ] `internal/store/migrate_test.go` — reopen idempotency + WAL multi-reader
- [ ] `internal/store/fts_internal_test.go` — FTS5 trigger sync on insert/update/delete
- [ ] `internal/store/store_perms_test.go` — 0700 directory, db file mode, open-failure path
- [ ] `cmd/status_failure_test.go` — non-zero exit / no usage dump, `--store-path` cleaning

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| *(none)* | — | The binary build + `status` run is automated in tasks `01-01-03` / `01-02-02` and exposed as `make smoke`, so no manual-only verification remains | — |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references (folded into task `01-01-01`)
- [x] No watch-mode flags
- [x] Feedback latency < 30s (full suite ~5–15s)
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** planned — awaiting execution
