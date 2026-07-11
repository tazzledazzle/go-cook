# Phase 5 Context: Full Download & Progress CLI

**Gathered:** 2026-07-10
**Status:** Ready for planning
**Source:** TDD + vertical slice MVP + ROADMAP Phase 5

<domain>
## Phase Boundary

Download the **entire single-file torrent** with CLI entrypoint and live progress output. Scales Phase 4's one-piece path to all pieces, adds tracker-driven peer discovery, concurrent peer goroutines, and `torrent download <file.torrent>`. Graceful shutdown (SIGINT) deferred to Phase 6 — use `context.Context` hooks but no signal handling yet.

Out of scope: seeding, DHT, multi-file torrents, request pipelining, Phase 6 shutdown UX.
</domain>

<decisions>
## Implementation Decisions

### TDD discipline (unchanged)
- Strict RED → GREEN commits per plan
- Mock TCP peers + httptest tracker for CI (no live network)
- Record RED failures in SUMMARY.md

### Vertical slice ordering
1. Progress formatting (CLI-02 unit layer)
2. Piece manager + refactor `DownloadPiece` (no per-piece writer Close)
3. Downloader orchestrator — announce, dial, download all pieces (CLI-01 core)
4. Multi-peer concurrent workers (ROADMAP criterion 4)
5. CLI `cmd/torrent/main.go` + README manual verification (CLI-01, TEST-03)

### Progress (CLI-02)
- `internal/torrent/progress.go`: percent from **verified pieces**, download speed from bytes / elapsed, active peer count
- Format: `42.0% complete | 1.2 MB/s | 3 peers` (stable for testing)

### Piece manager
- `internal/pieces/manager.go`: track completed piece indices, `NextMissing()`, `MarkComplete(index)`
- Thread-safe (mutex) for multi-peer access

### DownloadPiece refactor
- Remove `w.Close()` from `DownloadPiece` — Downloader owns writer lifecycle
- Update Phase 4 test to Close writer explicitly after DownloadPiece

### Downloader (CLI-01)
- `internal/torrent/download.go`: `Downloader` with tracker client, `Download(ctx, tor, dir)`
- Flow: announce → create writer → spawn peer goroutine per tracker peer → workers pull missing pieces until all complete
- Inject `net.Dial` in tests; production uses `net.Dialer`

### Multi-peer (ROADMAP #4)
- One goroutine per peer address from tracker
- Shared Manager coordinates which piece index to download next
- Progress reports active peer count

### CLI
- `cmd/torrent/main.go`: stdlib `flag` + args — `torrent download <path>`
- Output dir: current working directory
- Print progress line on each piece completion (or ticker — keep simple: on piece complete)

### TEST-03
- README section: manual verification steps with `testdata/minimal.torrent` or public ISO torrent
</decisions>

<canonical_refs>
- Phase 4: `DownloadPiece`, `file.Writer`, `peer.Connection`
- `internal/tracker/` — Announce, compact peers
- `.planning/research/PITFALLS.md` — progress from verified pieces only
</canonical_refs>

---
*Phase: 05-full-download-progress-cli*
