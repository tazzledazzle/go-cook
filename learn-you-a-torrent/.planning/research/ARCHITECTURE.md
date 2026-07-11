# Architecture Research

**Domain:** BitTorrent client in Go
**Researched:** 2026-07-10
**Confidence:** HIGH

## Standard Architecture

### System Overview

Reference layout from [ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md):

```
main.go (orchestration)
    ├── bencode/     Decode .torrent bytes
    ├── torrent/     Parse metadata, info hash, Downloader
    ├── tracker/     HTTP announce → peer list
    ├── peer/        TCP + handshake + wire messages
    ├── pieces/      Piece state, blocks, verification, selection
    └── file/        Map pieces → disk writes
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| `bencode/` | Serialize/deserialize bencode | Recursive decoder; encoder only for info hash if needed |
| `torrent/` | `Torrent` struct, parse `.torrent`, `Downloader` | Open file → decode → extract raw info dict → SHA1 |
| `tracker/` | Build announce URL, parse compact peers | `http.Get`, 6-byte peer entries (4 IP + 2 port) |
| `peer/` | Connection, handshake, message read/write | `net.Conn`, 4-byte length prefix messages |
| `pieces/` | Block assembly, SHA1 validate, rarest-first | Mutex-protected manager, 16 KB blocks |
| `file/` | Preallocate, write piece at offset | Single file: write at `pieceIndex * pieceLength` |

## Recommended Project Structure

```
learn-you-a-torrent/
├── go.mod
├── cmd/
│   └── torrent/
│       └── main.go              # CLI entry, signal handling
├── internal/
│   ├── bencode/
│   │   ├── decode.go
│   │   ├── decode_test.go
│   │   └── encode.go            # minimal, for info hash verification tests
│   ├── torrent/
│   │   ├── torrent.go
│   │   ├── parser.go
│   │   ├── info_hash.go
│   │   └── parser_test.go
│   ├── tracker/
│   │   ├── client.go
│   │   ├── announce.go
│   │   └── client_test.go
│   ├── peer/
│   │   ├── handshake.go
│   │   ├── message.go
│   │   ├── connection.go
│   │   └── handshake_test.go
│   ├── pieces/
│   │   ├── piece.go
│   │   ├── manager.go
│   │   └── manager_test.go
│   └── file/
│       ├── writer.go
│       └── writer_test.go
├── testdata/
│   ├── minimal.torrent          # synthetic single-file fixture
│   └── bencode/                 # raw bencode snippets
└── README.md
```

### Structure Rationale

- **`internal/`:** Go convention — packages not importable outside module
- **`cmd/torrent/`:** Keeps `main` thin; orchestration delegates to `torrent.Downloader`
- **`testdata/`:** Committed fixtures for TDD without network

## Vertical Slice Build Order (TDD MVP)

Horizontal module order (bencode → torrent → tracker → peer → pieces → file) is the **dependency graph**. Vertical slices cut across it to deliver testable milestones:

| Slice | Delivers | Modules Touched | Test |
|-------|----------|-----------------|------|
| **1** | Parse fixture `.torrent`, print name + info hash | bencode, torrent | Unit: golden `.torrent` → expected hash |
| **2** | Announce to tracker, print peer count | + tracker | Integration: mock HTTP or recorded response |
| **3** | Handshake with one peer, read bitfield | + peer | Integration: requires peer fixture or live peer |
| **4** | Download and verify ONE piece | + pieces, file | Integration: smallest torrent possible |
| **5** | Full download + progress CLI | all + main | E2E: synthetic or tiny live torrent |
| **6** | Graceful shutdown | main, pieces | Test: signal → clean exit, partial state |

Each slice: **write failing test → implement minimum → refactor**.

## Data Flow

### Startup Flow
```
.torrent → bencode.Decode → Torrent struct → SHA1(raw info) → InfoHash
InfoHash → tracker.Announce → []Peer{IP, Port}
```

### Download Flow
```
Peer → Handshake → Bitfield/Have/Unchoke
→ pieces.GetPieceToRequest → peer.Request(16KB block)
→ peer.Piece message → pieces.HandlePieceMessage
→ piece.Validate(SHA1) → file.WritePiece
```

## Threading Model

```
main goroutine: parse, tracker, spawn peers, progress loop
per peer: goroutine with readLoop + message handler
shared (mutex): PieceManager, FileWriter
```

## Anti-Patterns

### Anti-Pattern 1: Horizontal "finish bencode completely first"

**What people do:** Perfect bencode encoder/decoder for weeks before touching torrents.
**Why it's wrong:** No feedback loop; motivation dies.
**Do this instead:** Slice 1 needs decode + raw info extraction only; expand bencode as tests demand.

### Anti-Pattern 2: Big-bang integration test only

**What people do:** One test that downloads Ubuntu ISO.
**Why it's wrong:** 20 modules fail at once; impossible to debug.
**Do this instead:** Unit tests per package + one-piece integration before full file.

## Integration Points

| Boundary | Communication | Notes |
|----------|---------------|-------|
| torrent → tracker | InfoHash, peer_id, port stats | URL-encode binary info_hash |
| peer → pieces | Piece messages | Index, begin, block data |
| pieces → file | Verified piece bytes | Piece index → byte offset |

## Sources

- [BitTorrentClient ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md)

---
*Architecture research for: BitTorrent client*
*Researched: 2026-07-10*
