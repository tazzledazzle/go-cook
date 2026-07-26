# Phase 5 Research: Full Download & Progress CLI

**Researched:** 2026-07-10
**Confidence:** HIGH

## Downloader Architecture

```
Parse .torrent → Downloader.Download(ctx, tor, dir)
  → tracker.Announce → []Peer
  → file.NewWriter(dir, tor.Info)
  → pieces.NewManager(tor.Info.PieceCount())
  → for each peer: go peerWorker(ctx, peer, ...)
  → wait until manager.Complete() or ctx cancelled
  → writer.Close()
```

## Peer Worker (per goroutine)

```
net.Dial(tcp, peer) → Connection → PerformHandshake
→ read bitfield/unchoke → SendInterested
→ loop: index := manager.NextMissing(); if done return
→ DownloadPiece(conn, tor, index, writer) — without closing writer
→ manager.MarkComplete(index) → progress callback
```

## Testing Without Live Network

| Component | Test double |
|-----------|-------------|
| Tracker | httptest.Server returning bencoded `d5:peers6:<compact>e` |
| Peer | `net.Listen("tcp", "127.0.0.1:0")` + handler goroutine (reuse Phase 4 mock logic) |
| Torrent | `testdata/minimal.torrent` (1 piece, 12 bytes) |

Compact peer bytes for 127.0.0.1:PORT (big-endian port):

```go
port := listener.Addr().(*net.TCPAddr).Port
compact := []byte{127, 0, 0, 1, byte(port >> 8), byte(port)}
```

## Progress Calculation

- **Percent:** `completedPieces / totalPieces * 100` (not requested bytes — Pitfall "Looks Done But Isn't")
- **Speed:** `downloadedBytes / elapsedSeconds` since download start
- **Peers:** count of active worker goroutines (atomic/int guarded by mutex)

## CLI Args (stdlib)

```go
// os.Args: torrent download path/to/file.torrent
```

No external CLI framework — matches minimal-deps constraint.

## Phase 4 Refactor Required

`DownloadPiece` currently calls `w.Close()` — breaks multi-piece downloads. Phase 5 plan 02 removes Close; Downloader closes once.

## TDD Build Order

1. Progress.String() pure tests
2. Manager NextMissing/MarkComplete
3. Refactor DownloadPiece + Downloader single-peer E2E
4. Multi-peer worker test (2 connections)
5. CLI + README

## Sources

- ROADMAP Phase 5
- Reference [download.go](https://github.com/amaydixit11/BitTorrentClient/blob/main/internal/torrent/download.go)

---
*Phase 5 research complete*
