# Roadmap: Learn You a BitTorrent Client

**Project:** Learn You a BitTorrent Client
**Phases:** 6
**Requirements:** 22 v1 (100% mapped)
**Mode:** Vertical slice MVP + TDD

## Phase Overview

| # | Phase | Goal | Requirements | Success Criteria |
|---|-------|------|--------------|------------------|
| 1 | Bencode & Torrent Parsing | Read `.torrent` files and compute info hash | BENC-01, BENC-02, TORR-01, TORR-02, TEST-02 | 4 |
| 2 | Tracker Announce | Discover peers via HTTP tracker | TRCK-01, TRCK-02, TRCK-03 | 3 |
| 3 | Peer Handshake & Messages | Connect and speak wire protocol | PEER-01, PEER-02, PEER-03 | 4 |
| 4 | Download One Piece | End-to-end piece transfer with verification | PEER-04, PIEC-01–04, FILE-01, FILE-02 | 5 |
| 5 | Full Download & Progress | Complete file download with CLI feedback | CLI-01, CLI-02, TEST-03 | 4 |
| 6 | Graceful Shutdown | Clean interrupt handling | CLI-03 | 3 |

---

### Phase 1: Bencode & Torrent Parsing
**Goal:** Parse a `.torrent` fixture and compute the correct 20-byte info hash
**Mode:** mvp
**Requirements:** BENC-01, BENC-02, TORR-01, TORR-02, TEST-02

**Success Criteria:**
1. `go test ./internal/bencode/...` passes with table-driven decode tests (string, int, list, dict)
2. `go test ./internal/torrent/...` parses `testdata/minimal.torrent` and extracts name, piece length, piece count
3. Info hash matches a known-good value for the fixture (golden test)
4. Raw info dict bytes extracted without re-encoding

**Implementation guide:**
- Create `go.mod`, `internal/bencode/decode.go`, `internal/torrent/parser.go`
- TDD: write `decode_test.go` first with inline bencode snippets, then `parser_test.go` with fixture file
- Reference module: [bencode/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/bencode), [torrent/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/torrent)

---

### Phase 2: Tracker Announce
**Goal:** Contact HTTP tracker and receive a list of peer IP:port pairs
**Mode:** mvp
**Requirements:** TRCK-01, TRCK-02, TRCK-03

**Success Criteria:**
1. Announce URL built with correctly URL-encoded binary info_hash
2. Mock HTTP test returns compact peers and parser extracts IP:port list
3. CLI subcommand or test prints peer count from a live tracker (optional manual check)

**Implementation guide:**
- `internal/tracker/client.go`, `announce.go`, `peers.go`
- TDD: httptest.Server returning bencoded `{"peers": <6-byte compact>}`
- Reference: [tracker/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/tracker)

---

### Phase 3: Peer Handshake & Messages
**Goal:** TCP connect to a peer, complete handshake, read bitfield and unchoke
**Mode:** mvp
**Requirements:** PEER-01, PEER-02, PEER-03

**Success Criteria:**
1. Handshake serialize/deserialize roundtrip test passes (68 bytes)
2. Client connects to peer and completes handshake with matching info hash
3. Message loop correctly parses keepalive, choke, unchoke, interested, bitfield, have
4. Mismatched info hash closes connection

**Implementation guide:**
- `internal/peer/handshake.go`, `message.go`, `connection.go`
- TDD: unit tests for message framing; integration test against real peer from Phase 2
- Reference: [peer/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/peer)

---

### Phase 4: Download One Piece
**Goal:** Request blocks, assemble one piece, SHA1 verify, write to disk
**Mode:** mvp
**Requirements:** PEER-04, PIEC-01, PIEC-02, PIEC-03, PIEC-04, FILE-01, FILE-02

**Success Criteria:**
1. Client sends Interested, receives Unchoke, sends Request for 16 KB blocks
2. All blocks for piece index 0 assembled in buffer
3. SHA1 validation passes against hash from `.torrent`
4. Verified piece bytes written to output file at offset 0
5. Hash mismatch triggers piece reset

**Implementation guide:**
- `internal/pieces/piece.go`, `manager.go`, `internal/file/writer.go`
- Start with single peer, single piece, sequential blocks
- Reference: [pieces/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/pieces), [file/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/file)

---

### Phase 5: Full Download & Progress CLI
**Goal:** Download entire single-file torrent with progress output
**Mode:** mvp
**Requirements:** CLI-01, CLI-02, TEST-03

**Success Criteria:**
1. `go run ./cmd/torrent download file.torrent` completes full file
2. Progress line updates with % complete, speed, peer count
3. Output file SHA1 matches expected (or manual check against public torrent documented in README)
4. Multiple peers connected concurrently (goroutine per peer)

**Implementation guide:**
- `cmd/torrent/main.go`, `internal/torrent/download.go` (Downloader orchestrator)
- Piece loop: select next missing piece, request from unchoked peers
- Reference: [download.go](https://github.com/amaydixit11/BitTorrentClient/blob/main/internal/torrent/download.go), [main.go](https://github.com/amaydixit11/BitTorrentClient/blob/main/main.go)

---

### Phase 6: Graceful Shutdown
**Goal:** Ctrl+C stops download cleanly without corrupting output
**Mode:** mvp
**Requirements:** CLI-03

**Success Criteria:**
1. SIGINT/SIGTERM caught in main
2. Peer goroutines cancelled via context
3. Partial file left in consistent state (no half-written piece at EOF)
4. Exit message reports progress at shutdown time

**Implementation guide:**
- `signal.Notify` + `context.Context` propagation
- Downloader `Stop()` closes connections and waits for goroutines

---

## Phase Ordering Rationale

Vertical slices follow the natural protocol stack dependency chain while delivering testable milestones early. Phase 4 (one piece) is the critical integration point — if one piece works, Phase 5 is scaling not new protocol logic. TDD is embedded in every phase via TEST-01.

## Reference Architecture Mapping

| Reference Module | Phases |
|----------------|--------|
| `bencode/` | 1 |
| `torrent/` | 1, 5 |
| `tracker/` | 2 |
| `peer/` | 3, 4 |
| `pieces/` | 4, 5 |
| `file/` | 4, 5 |
| `main.go` / Downloader | 5, 6 |

Full architecture doc: [ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md)

---
*Roadmap created: 2026-07-10*
