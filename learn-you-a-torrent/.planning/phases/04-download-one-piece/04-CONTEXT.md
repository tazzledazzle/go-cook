# Phase 4 Context: Download One Piece

**Gathered:** 2026-07-10
**Status:** Ready for planning
**Source:** TDD + vertical slice MVP + ROADMAP Phase 4

<domain>
## Phase Boundary

End-to-end download of **piece index 0** from a single mock peer: send Interested → receive Unchoke → Request 16 KB blocks → assemble piece buffer → SHA1 verify against `.torrent` metadata → write verified bytes to disk. Single peer, sequential blocks, no multi-piece loop (Phase 5 scales this).

Out of scope: multiple peers, pipelined requests, CLI, progress output, seeding.
</domain>

<decisions>
## Implementation Decisions

### TDD discipline (unchanged)
- Strict RED → verify fail → GREEN → verify pass → optional REFACTOR
- Separate commits: `test(04-NN):` then `feat(04-NN):`
- Record RED failure message in each `04-NN-SUMMARY.md`
- `net.Pipe` mock peer for integration — no live peers in CI

### Vertical slice MVP ordering
Plans build the **one-piece download slice** bottom-up, each plan testable in isolation before final integration:

1. Piece buffer assembly (blocks at `begin` offset)
2. SHA1 validation + reset on mismatch
3. File writer at correct offset
4. Request/Piece wire helpers + Connection send methods
5. Integration: mock peer delivers piece 0 → verify → write `test.txt`

### Piece assembly (PIEC-01, PIEC-02)
- `internal/pieces/piece.go`: `NewPiece(length int) *Piece`
- `WriteBlock(begin int, data []byte) error` — copy into buffer at `begin`
- `Complete() bool` — all block ranges filled (bitmap or received-bytes tracking)
- `Bytes() []byte` — full piece-length buffer
- Block size constant: `BlockSize = 16384` (2^14)

### Piece verification (PIEC-03, PIEC-04)
- `Validate(expected [20]byte) bool` — SHA1 of full piece buffer vs torrent hash
- `Reset()` — zero buffer and block tracking for re-download
- Golden hash for `testdata/minimal.torrent` piece 0: `98aaed442721e0cecdd9bf7b8cbb3e1ff1b1536a`

### File I/O (FILE-01, FILE-02)
- `internal/file/writer.go`: create output file named from `torrent.Info.Name`
- `WritePiece(index int, data []byte, info torrent.Info)` — offset = `index * pieceLength`, write `min(pieceLength, remaining file bytes)` for last piece
- Minimal fixture: 12-byte `test.txt` at offset 0

### Peer protocol extensions (PEER-04)
- `BuildRequest(index, begin, length uint32) Message` — 12-byte payload (3× uint32 BE)
- `ParsePiece(msg Message) (index, begin uint32, block []byte, err error)`
- `Connection.SendInterested()`, `Connection.SendRequest(...)`, `Connection.WriteMessage` delegate
- **Pitfall 5:** never Request before Unchoke — integration test asserts ordering
- **Pitfall 6:** `blockLen = min(BlockSize, pieceLength - begin)`; minimal torrent uses one 12-byte block

### Integration orchestrator
- `internal/pieces/downloader.go`: `DownloadPiece(conn *peer.Connection, t *torrent.Torrent, index int, w *file.Writer) error`
- Reads messages until piece complete, validates, writes to disk
- Phase 4 tests only piece index 0

### Claude's Discretion
- Block tracking via `map[int]bool` vs bitset — prefer simple map/set for v1
- Whether `torrent.Info.PieceHash(index int) [20]byte` helper lives in `torrent` or `pieces` — add to `torrent.Info` if tests need it
</decisions>

<canonical_refs>
## Canonical References

- `.planning/research/PITFALLS.md` — Pitfalls 5–7 (unchoke ordering, block size, hash mismatch)
- `testdata/minimal.torrent` + `testdata/README.md` — 12-byte file, 1 piece, golden hashes
- `internal/peer/` — Connection, ReadMessage, WriteMessage (Phase 3)
- [Reference pieces/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/pieces), [file/](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/file)
</canonical_refs>

---
*Phase: 04-download-one-piece*
