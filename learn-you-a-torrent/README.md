# Personal BitTorrent Client

A from-scratch BitTorrent client in Go — built to learn how torrents work layer by layer.

**Planning docs:** `.planning/` (PROJECT.md, ROADMAP.md, REQUIREMENTS.md)

## System Overview

```
cmd/torrent/main.go          ← CLI entry, signal handling, progress loop
        │
        ├── internal/bencode/    ← Decode .torrent bytes
        ├── internal/torrent/      ← Parse metadata, info hash, Downloader
        ├── internal/tracker/      ← HTTP announce → peer list
        ├── internal/peer/         ← TCP handshake + wire messages
        ├── internal/pieces/       ← Block assembly, SHA1 verification
        └── internal/file/         ← Write verified pieces to disk
```

Reference: [BitTorrentClient ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md)

## Data Flow

### Startup
```
.torrent → bencode decode → Torrent struct → SHA1(raw info dict) → info hash
info hash → tracker announce → peer IP:port list
```

### Download
```
peer TCP connect → 68-byte handshake → bitfield / have / unchoke
→ request 16 KB blocks → piece messages → assemble piece
→ SHA1 verify → write to disk at piece offset
```

## Module Architecture

| Module     | Responsibility                    | Key types                                   |
|------------|-----------------------------------|---------------------------------------------|
| `bencode/` | Torrent serialization format      | `Decoder`, decode string/int/list/dict      |
| `torrent/` | `.torrent` parsing & coordination | `Torrent`, `Info`, `InfoHash`, `Downloader` |
| `tracker/` | Peer discovery via HTTP           | `TrackerClient`, compact peer parsing       |
| `peer/`    | BitTorrent wire protocol (BEP 3)  | `Handshake`, `Message`, `Connection`        |
| `pieces/`  | Piece/block state & verification  | `Piece`, `Manager`, `Block`                 |
| `file/`    | Disk writes                       | `Writer`, piece → byte offset               |

## Build Phases (Vertical Slice MVP)

Each phase is TDD-first: write failing test → implement → refactor.

| Phase | You build                                    | You can demo                        |
|-------|----------------------------------------------|-------------------------------------|
| **1** | Bencode decoder + torrent parser + info hash | `go test` prints parsed name & hash |
| **2** | HTTP tracker client                          | Print peer list from announce       |
| **3** | Handshake + message loop                     | Connect to peer, read bitfield      |
| **4** | Piece download + verify + write              | One verified piece on disk          |
| **5** | Full downloader + progress CLI               | Complete single-file download       |
| **6** | Graceful shutdown                            | Ctrl+C exits cleanly                |

Phases 1–5 implemented. Phase 6 adds signal handling.

## File Reference

Planned layout (created during implementation):

```
learn-you-a-torrent/
├── cmd/torrent/main.go
├── internal/
│   ├── bencode/decode.go
│   ├── torrent/parser.go, info_hash.go, progress.go
│   ├── downloader/download.go
│   ├── tracker/client.go, announce.go
│   ├── peer/handshake.go, message.go, connection.go
│   ├── pieces/piece.go, manager.go, downloader.go
│   └── file/writer.go
└── testdata/minimal.torrent
```

## Run

**Prerequisites:** Go 1.22+

```bash
# Quick run (no build step)
go run ./cmd/torrent download path/to/file.torrent

# Build the binary first
go build -o torrent ./cmd/torrent
./torrent download path/to/file.torrent
```

The client will:
1. Parse the `.torrent` file
2. Announce to the tracker and retrieve peers
3. Connect to peers via TCP, exchange handshakes, and request pieces
4. Verify each piece with SHA1
5. Write the completed file to the current directory

Progress is printed inline. Press `Ctrl+C` to shut down gracefully — the shutdown progress line shows how far the download got.

**Smoke test with the synthetic fixture** (no tracker or peers required):
```bash
go test ./...
go test -race ./...
```

`TestDownloader_downloadsMinimalTorrent` exercises the full path using `testdata/minimal.torrent` (expected output: `hello world\n`).

## Getting Started

```bash
cd learn-you-a-torrent
/gsd-discuss-phase 1    # clarify approach (optional)
/gsd-plan-phase 1       # create executable plan
/gsd-execute-phase 1    # implement with TDD
```
