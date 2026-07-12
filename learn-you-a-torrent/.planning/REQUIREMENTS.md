# Requirements: Learn You a BitTorrent Client

**Defined:** 2026-07-10
**Core Value:** Download a single-file torrent with verified pieces and understand every layer

## v1 Requirements

### Bencode & Metadata

- [ ] **BENC-01**: Decoder parses bencode strings, integers, lists, and dictionaries from bytes
- [ ] **BENC-02**: Decoder records byte offsets of the info dictionary for raw extraction
- [ ] **TORR-01**: Parser reads a `.torrent` file into a `Torrent` struct (announce URL, info name, piece length, piece hashes)
- [ ] **TORR-02**: Info hash computed as SHA1 of raw info dictionary bytes (matches known test vectors)

### Tracker

- [ ] **TRCK-01**: Client builds a valid HTTP announce URL with info_hash, peer_id, port, uploaded, downloaded, left, compact=1
- [ ] **TRCK-02**: Client parses compact peer response (6 bytes per peer: 4-byte IP + 2-byte port)
- [ ] **TRCK-03**: Client returns a list of peer addresses from tracker response

### Peer Wire Protocol

- [x] **PEER-01**: Client sends and receives 68-byte BitTorrent handshake (pstr, reserved, info_hash, peer_id)
- [x] **PEER-02**: Client verifies peer's info_hash matches local hash; rejects mismatch
- [x] **PEER-03**: Client reads length-prefixed messages (keepalive, choke, unchoke, interested, bitfield, have, request, piece)
- [x] **PEER-04**: Client sends Interested and waits for Unchoke before requesting blocks

### Pieces & Verification

- [x] **PIEC-01**: Piece manager tracks piece states (pending, downloading, complete, failed)
- [x] **PIEC-02**: Incoming Piece messages store 16 KB blocks at correct begin offset within piece buffer
- [x] **PIEC-03**: Complete piece validated against SHA1 hash from `.torrent` metadata
- [x] **PIEC-04**: Failed hash resets piece for re-download

### File I/O

- [x] **FILE-01**: Writer creates output file named from torrent info.name
- [x] **FILE-02**: Verified piece written at correct byte offset (pieceIndex × pieceLength)

### CLI & Orchestration

- [x] **CLI-01**: `torrent download <file.torrent>` starts download to current directory
- [x] **CLI-02**: Progress output shows percent complete, download speed, and connected peer count
- [x] **CLI-03**: Ctrl+C triggers graceful shutdown (stop peers, flush file, exit cleanly)

### Testing

- [ ] **TEST-01**: Each package has unit tests written TDD-first (test before implementation)
- [ ] **TEST-02**: Synthetic `.torrent` fixtures in testdata/ for CI without network
- [x] **TEST-03**: Manual end-to-end verification documented against a public Linux ISO torrent

## v2 Requirements

### Multi-File

- **MULT-01**: Parser handles multi-file info with path components
- **MULT-02**: Writer maps pieces spanning multiple files

### Resilience

- **RESU-01**: Multitracker announce-list failover
- **RESU-02**: Resume partial download from checkpoint

### Performance

- **PERF-01**: Rarest-first piece selection across peers
- **PERF-02**: Parallel block requests per peer (pipelining)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Seeding / uploading | Download-only v1; halves state machine complexity |
| DHT / magnet links | Tracker-only discovery for v1 |
| Protocol encryption (MSE/PE) | Plain wire protocol for learning clarity |
| GUI / web UI | CLI-only learning project |

## Traceability

| Requirement | Phase | Status |
|-------------|-------|--------|
| BENC-01 | Phase 1 | Complete |
| BENC-02 | Phase 1 | Complete |
| TORR-01 | Phase 1 | Complete |
| TORR-02 | Phase 1 | Complete |
| TRCK-01 | Phase 2 | Complete |
| TRCK-02 | Phase 2 | Complete |
| TRCK-03 | Phase 2 | Complete |
| PEER-01 | Phase 3 | Complete |
| PEER-02 | Phase 3 | Complete |
| PEER-03 | Phase 3 | Complete |
| PEER-04 | Phase 4 | Complete |
| PIEC-01 | Phase 4 | Complete |
| PIEC-02 | Phase 4 | Complete |
| PIEC-03 | Phase 4 | Complete |
| PIEC-04 | Phase 4 | Complete |
| FILE-01 | Phase 4 | Complete |
| FILE-02 | Phase 4 | Complete |
| CLI-01 | Phase 5 | Complete |
| CLI-02 | Phase 5 | Complete |
| CLI-03 | Phase 6 | Complete |
| TEST-01 | All phases | Pending |
| TEST-02 | Phase 1 | Complete |
| TEST-03 | Phase 5 | Complete |

**Coverage:**
- v1 requirements: 22 total
- Mapped to phases: 22
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-10*
*Last updated: 2026-07-10 after roadmap creation*
