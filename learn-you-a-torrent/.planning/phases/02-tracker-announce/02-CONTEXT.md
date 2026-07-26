# Phase 2 Context: Tracker Announce

**Gathered:** 2026-07-10
**Status:** Ready for planning
**Source:** TDD approach switch + ROADMAP Phase 2

<domain>
## Phase Boundary

HTTP tracker announce: build announce URL from torrent metadata, GET the tracker, parse compact peer list into `[]Peer{IP, Port}`.

Uses Phase 1 outputs: `torrent.Torrent` (Announce URL, InfoHash, Info.Length for `left` param).
</domain>

<decisions>
## Implementation Decisions

### TDD discipline (Phase 2+)
- **Strict RED-GREEN-REFACTOR** per superpowers TDD skill — no production code without a failing test first
- Each TDD plan produces separate commits: `test(02-NN): failing test` → `feat(02-NN): implement` → optional `refactor(02-NN)`
- Executor MUST run the test command in RED phase and confirm failure before writing implementation
- Phase 1 code is kept as-is (already shipped); do not rewrite unless a regression test exposes a bug

### Tracker protocol
- HTTP GET to `torrent.Announce` with query params per BEP 3
- `compact=1` always — parse 6-byte peer entries only for v1
- `info_hash` URL-encoded as raw 20 bytes (NOT hex string)
- `peer_id` is 20-byte client ID (generate once per client, alphanumeric prefix `-GO0001-` + random suffix)
- `left` = total bytes remaining (= file length initially for download-only)
- `uploaded=0`, `downloaded=0`, `port=6881` (listening port placeholder for v1)

### Testing strategy
- **No live tracker in CI** — use `net/http/httptest` with hand-crafted bencode response bodies
- Golden test for announce URL query string with known info_hash bytes
- Table-driven tests for compact peer parsing (1 peer, 2 peers, empty, malformed)
- Integration test: mock server → `Client.Announce` → peer list

### Package layout
- `internal/tracker/peer.go` — `Peer` struct (net.IP or string IP + Port uint16)
- `internal/tracker/announce.go` — URL builder
- `internal/tracker/peers.go` — compact format parser
- `internal/tracker/client.go` — HTTP client + response decode

### Claude's Discretion
- Whether to use `net/http.Client` injection vs package-level default (prefer injection for testability)
- Exact peer_id generation algorithm (must be 20 bytes)
- Minimal bencode response decode — reuse `internal/bencode` decoder on response body
</decisions>

<canonical_refs>
## Canonical References

- `.planning/research/PITFALLS.md` — Pitfall 2 (info_hash URL encoding)
- `.planning/phases/01-bencode-torrent-parsing/01-03-SUMMARY.md` — Phase 1 torrent parser output
- `internal/torrent/parser.go` — Torrent struct with InfoHash and Announce
- `testdata/minimal.torrent` — fixture for integration tests
- [Reference tracker module](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/tracker)
</canonical_refs>

<deferred>
## Deferred Ideas

- Multitracker announce-list (v2)
- Dictionary-format peer responses (compact only for v1)
- Live tracker manual smoke test (optional, not in CI)
</deferred>

---
*Phase: 02-tracker-announce*
*Context gathered: 2026-07-10 — TDD switch*
