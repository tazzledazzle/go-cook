---
phase: 1
slug: platform-foundation
status: draft
nyquist_compliant: false
wave_0_complete: false
created: 2026-07-24
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
| 01-W0-01 | 01 | 0 | PLAT-03 | — | N/A | setup | `go get github.com/stretchr/testify@v1.11.1` | ❌ W0 | ⬜ pending |
| 01-XX-01 | TBD | 1 | PLAT-02 | T-1-01 | parameterized SQL only; store dir `0o700` | unit | `go test ./internal/store/... -run TestOpen -race` | ❌ W0 | ⬜ pending |
| 01-XX-02 | TBD | 1 | PLAT-02 | T-1-02 | WAL pragma verified (not silent ignore) | unit | `go test ./internal/store/... -run TestOpen -race` | ❌ W0 | ⬜ pending |
| 01-XX-03 | TBD | 1 | PLAT-03 | — | table-driven + `-race` | unit | `go test ./internal/store/... -race` | ❌ W0 | ⬜ pending |
| 01-XX-04 | TBD | 2 | PLAT-01 | T-1-03 | no network/daemon; filepath.Clean on store path | integration | `go test ./cmd/... -run TestStatusCommand -race` | ❌ W0 | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

*Planner MUST rewrite Task IDs above to match final PLAN.md task IDs and set `nyquist_compliant: true` when every plan task has an automated verify mapped.*

---

## Wave 0 Requirements

- [ ] `go.mod` / `go.sum` with testify (+ modernc.org/sqlite, cobra, viper per STACK)
- [ ] `internal/store/store_test.go` — PLAT-02 (WAL mode + FTS5 table existence)
- [ ] `internal/store/migrate_test.go` — migration idempotency (Open twice)
- [ ] `cmd/status_test.go` — PLAT-01 (Cobra SetOut/SetArgs/Execute against temp store)

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| `go build -o modular .` produces a runnable binary | PLAT-01 | Confirms install path outside unit tests | From module root: `go build -o modular . && ./modular status` exits 0 |

---

## Validation Sign-Off

- [ ] All tasks have `<automated>` verify or Wave 0 dependencies
- [ ] Sampling continuity: no 3 consecutive tasks without automated verify
- [ ] Wave 0 covers all MISSING references
- [ ] No watch-mode flags
- [ ] Feedback latency < 30s
- [ ] `nyquist_compliant: true` set in frontmatter

**Approval:** pending
