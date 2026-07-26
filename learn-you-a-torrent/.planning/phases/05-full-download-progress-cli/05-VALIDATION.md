---
phase: 5
slug: full-download-progress-cli
status: draft
nyquist_compliant: true
created: 2026-07-10
---

# Phase 5 — Validation Strategy

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + httptest + TCP mock peers |
| **Quick run** | `go test ./internal/torrent/... ./cmd/torrent/...` |
| **Full suite** | `go test ./...` |
| **Race** | `go test -race ./...` |
| **CLI smoke** | `go run ./cmd/torrent download testdata/minimal.torrent` (manual / plan 05-05) |

## TDD Gate

| Plan | Requirements | RED command |
|------|--------------|-------------|
| 05-01 | CLI-02 | `go test ./internal/torrent/... -run TestProgress -v` (fail) |
| 05-02 | PIEC-01 ext | `go test ./internal/pieces/... -run TestManager -v` (fail) |
| 05-03 | CLI-01 core | `go test ./internal/torrent/... -run TestDownloader_downloadsMinimal -v` (fail) |
| 05-04 | ROADMAP #4 | `go test ./internal/torrent/... -run TestDownloader_multiplePeers -v` (fail) |
| 05-05 | CLI-01, TEST-03 | `go test ./cmd/torrent/... -run TestDownloadCLI -v` (fail) |

## Phase Success Criteria

1. `go run ./cmd/torrent download file.torrent` completes full file
2. Progress shows % complete, speed, peer count
3. Output file matches expected content (minimal.torrent → `hello world\n`)
4. Multiple peer goroutines started from tracker list
