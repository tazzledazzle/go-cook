---
phase: 2
slug: tracker-announce
status: draft
nyquist_compliant: true
wave_0_complete: true
created: 2026-07-10
---

# Phase 2 — Validation Strategy

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + httptest |
| **Quick run command** | `go test ./internal/tracker/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~3 seconds |

## Sampling Rate

- **After RED commit:** Run failing test — must FAIL
- **After GREEN commit:** Run `go test ./internal/tracker/...` — must PASS
- **After each plan:** Run `go test ./...`

## Per-Task Verification Map

| Task | Plan | Requirement | Command | Status |
|------|------|-------------|---------|--------|
| RED announce URL | 02-01 | TRCK-01 | `go test ./internal/tracker/... -run TestBuildAnnounceURL -v` (fail) | ⬜ |
| GREEN announce URL | 02-01 | TRCK-01 | same (pass) | ⬜ |
| RED compact peers | 02-02 | TRCK-02 | `go test ./internal/tracker/... -run TestParseCompactPeers -v` (fail) | ⬜ |
| GREEN compact peers | 02-02 | TRCK-02 | same (pass) | ⬜ |
| RED client announce | 02-03 | TRCK-03 | `go test ./internal/tracker/... -run TestClient_Announce -v` (fail) | ⬜ |
| GREEN client announce | 02-03 | TRCK-03 | same (pass) | ⬜ |

## TDD Gate

Executor must record in SUMMARY.md:
- RED: exact failure message observed
- GREEN: test output showing pass
- No skipping RED verification
