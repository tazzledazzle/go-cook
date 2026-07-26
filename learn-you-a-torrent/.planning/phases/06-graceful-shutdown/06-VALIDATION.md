---
phase: 6
slug: graceful-shutdown
status: draft
nyquist_compliant: true
created: 2026-07-10
---

# Phase 6 — Validation Strategy

| Plan | Requirements | RED command |
|------|--------------|-------------|
| 06-01 | CLI-03 | `go test ./internal/downloader/... -run TestDownloader_cancel -v` (fail) |
| 06-02 | CLI-03 | `go test ./cmd/torrent/... -run TestRunDownload_cancel -v` (fail) |

Full: `go test ./... && go test -race ./...`
