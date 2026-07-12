---
phase: 1
slug: bencode-torrent-parsing
status: draft
nyquist_compliant: true
wave_0_complete: false
created: 2026-07-10
---

# Phase 1 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing (stdlib) + testify/assert optional |
| **Config file** | none |
| **Quick run command** | `go test ./internal/bencode/... ./internal/torrent/...` |
| **Full suite command** | `go test ./...` |
| **Estimated runtime** | ~2 seconds |

---

## Sampling Rate

- **After every task commit:** Run `go test ./internal/bencode/... ./internal/torrent/...`
- **After every plan wave:** Run `go test ./...`
- **Before `/gsd-verify-work`:** Full suite must be green
- **Max feedback latency:** 5 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Test Type | Automated Command | Status |
|---------|------|------|-------------|-----------|-------------------|--------|
| 01-01-01 | 01 | 1 | TEST-02 | fixture | `test -f testdata/minimal.torrent` | ⬜ pending |
| 01-02-01 | 02 | 2 | BENC-01 | unit | `go test ./internal/bencode/... -v` | ⬜ pending |
| 01-02-02 | 02 | 2 | BENC-02 | unit | `go test ./internal/bencode/... -run InfoDict` | ⬜ pending |
| 01-03-01 | 03 | 3 | TORR-01 | unit | `go test ./internal/torrent/... -run TestParse` | ⬜ pending |
| 01-03-02 | 03 | 3 | TORR-02 | unit | `go test ./internal/torrent/... -run TestInfoHash` | ⬜ pending |

---

## Wave 0 Requirements

- [x] `go.mod` — created in Plan 01
- [x] `testdata/minimal.torrent` — synthetic fixture in Plan 01
- [ ] `internal/bencode/decode_test.go` — table-driven stubs in Plan 02

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| Info hash matches external tool | TORR-02 | Optional confidence check | Compare `go test` golden hash against hand-computed SHA1 of info dict bytes |

---

## Nyquist Compliance

All v1 Phase 1 requirements have automated test commands mapped above.
