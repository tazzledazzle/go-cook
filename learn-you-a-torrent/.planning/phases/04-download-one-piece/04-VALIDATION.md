---
phase: 4
slug: download-one-piece
status: draft
nyquist_compliant: true
created: 2026-07-10
---

# Phase 4 — Validation Strategy

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + net.Pipe + t.TempDir |
| **Quick run** | `go test ./internal/pieces/... ./internal/file/... ./internal/peer/...` |
| **Full suite** | `go test ./...` |
| **Race** | `go test -race ./...` |

## TDD Gate

Each plan: RED must fail before feat commit. Record failure in SUMMARY.md.

| Plan | Requirements | RED command |
|------|--------------|-------------|
| 04-01 | PIEC-01, PIEC-02 | `go test ./internal/pieces/... -run TestPiece -v` (fail) |
| 04-02 | PIEC-03, PIEC-04 | `go test ./internal/pieces/... -run TestPieceValidate -v` (fail) |
| 04-03 | FILE-01, FILE-02 | `go test ./internal/file/... -run TestWriter -v` (fail) |
| 04-04 | PEER-04 (wire) | `go test ./internal/peer/... -run 'Test(Request\|Piece\|Send)' -v` (fail) |
| 04-05 | PEER-04 (e2e) | `go test ./internal/pieces/... -run TestDownloadPiece -v` (fail) |

## Phase Success Criteria (ROADMAP)

1. Client sends Interested, receives Unchoke, sends Request for blocks
2. All blocks for piece index 0 assembled in buffer
3. SHA1 validation passes against hash from `.torrent`
4. Verified piece bytes written to output file at offset 0
5. Hash mismatch triggers piece reset (covered in 04-02)

## Manual Verification (optional)

Not required for Phase 4 CI gate. Phase 5 adds live torrent manual check (TEST-03).
