# Phase 3 Context: Peer Handshake & Messages

**Gathered:** 2026-07-10
**Status:** Ready for planning
**Source:** TDD planning + ROADMAP Phase 3

<domain>
## Phase Boundary

BitTorrent peer wire protocol (BEP 3) through first messages: TCP connection, 68-byte handshake, verify info_hash, read length-prefixed messages (bitfield, unchoke, etc.). No piece downloading yet — that's Phase 4.
</domain>

<decisions>
## Implementation Decisions

### TDD discipline (unchanged from Phase 2)
- Strict RED → verify fail → GREEN → verify pass → optional REFACTOR
- Separate commits: `test(03-NN):` then `feat(03-NN):`
- Use `net.Pipe` for handshake/message tests — no live peers in CI
- Mock peer goroutine writes handshake + messages to pipe's other end

### Handshake (PEER-01, PEER-02)
- Exactly 68 bytes; pstrlen=19, protocol string 19 bytes, 8 reserved zeros
- `Handshake` struct with InfoHash [20]byte, PeerID [20]byte
- `Serialize()` / `Deserialize([]byte)` roundtrip
- `Exchange(rw io.ReadWriter, expected InfoHash, ours Handshake)` sends ours, reads theirs, verifies info_hash match

### Messages (PEER-03)
- Framing: 4-byte big-endian length, then if length>0: 1-byte id + payload
- length=0 → keepalive (no id byte)
- Phase 3 tests: choke, unchoke, interested, bitfield, have, keepalive
- Define constants for request(6)/piece(7) but defer tests to Phase 4

### Package layout
- `internal/peer/handshake.go`
- `internal/peer/message.go` — types, ReadMessage, WriteMessage
- `internal/peer/connection.go` — Connection wrapping net.Conn, Handshake + ReadMessage loop
- Reuse `tracker.Peer` for dial address (IP + Port)

### Claude's Discretion
- Read deadlines on conn (recommended: 30s for tests use none)
- Whether Connection sends Interested automatically (defer to Phase 4 — read only in Phase 3)
</decisions>

<canonical_refs>
## Canonical References

- `.planning/research/PITFALLS.md` — Pitfalls 3 (handshake length), 4 (message framing)
- `internal/torrent/` — InfoHash for handshake verification
- `internal/tracker/peer.go` — Peer address type for Connection.Dial target
- [Reference peer module](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/peer)
</canonical_refs>

---
*Phase: 03-peer-handshake-messages*
