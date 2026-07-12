# Project Research Summary

**Project:** Learn You a BitTorrent Client
**Domain:** BitTorrent peer-to-peer file transfer (Go, from-scratch)
**Researched:** 2026-07-10
**Confidence:** HIGH

## Executive Summary

Building a BitTorrent client from scratch is a layered protocol exercise: bencode metadata → HTTP tracker discovery → TCP peer wire protocol → piece assembly with SHA1 verification → disk writes. The [reference architecture](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md) decomposes this into six internal packages plus a thin `main.go` orchestrator — a proven layout for learning.

This project uses **vertical slice MVP with TDD**: each phase delivers a failing test that becomes green, proving progress toward a real download rather than finishing entire modules in isolation. Go stdlib covers TCP, HTTP, and SHA1; bencode and wire protocol are implemented manually. The highest-risk bugs are info hash calculation (must hash raw info dict bytes), tracker URL encoding, and peer state machine ordering (Interested → Unchoke → Request).

## Key Findings

### Recommended Stack

Go 1.22+ with stdlib for networking and crypto. Optional `testify` for readable tests. No torrent libraries — they defeat the learning goal.

**Core technologies:**
- Go stdlib (`net`, `net/http`, `crypto/sha1`) — peer connections, tracker, piece hashes
- Custom `internal/bencode/` — torrent file parsing
- `testify/assert` — TDD ergonomics

### Expected Features

**Must have (v1):** Parse `.torrent`, tracker announce, handshake, download/verify pieces, write single file, CLI progress, graceful shutdown.

**Defer (v2+):** Multi-file, seeding, DHT/magnets, resume.

### Architecture Approach

Six modules under `internal/` orchestrated from `cmd/torrent/main.go`. Vertical slices cut across modules: Slice 1 (parse + hash) → Slice 2 (tracker) → Slice 3 (handshake) → Slice 4 (one piece) → Slice 5 (full download + progress) → Slice 6 (shutdown).

### Critical Pitfalls

1. **Wrong info hash** — hash raw info dict bytes, don't re-encode
2. **Tracker binary encoding** — URL-encode 20-byte info_hash
3. **Request before unchoke** — peer state machine ordering
4. **Last block size** — 16 KB except final block of piece

## Implications for Roadmap

### Phase 1: Bencode & Torrent Parsing
**Rationale:** Everything depends on metadata and info hash
**Delivers:** Parse fixture `.torrent`, compute correct info hash
**Avoids:** Pitfall 1 (wrong info hash)

### Phase 2: Tracker Announce
**Rationale:** Need peers before wire protocol
**Delivers:** HTTP announce → peer list
**Avoids:** Pitfall 2 (URL encoding)

### Phase 3: Peer Wire Protocol
**Rationale:** Handshake + messages gate all transfers
**Delivers:** Connect, handshake, read bitfield/unchoke
**Avoids:** Pitfalls 3–4

### Phase 4: Download One Piece
**Rationale:** Proves end-to-end transfer before scaling
**Delivers:** Request blocks, assemble, SHA1 verify, write bytes
**Avoids:** Pitfalls 5–7

### Phase 5: Full Download & Progress CLI
**Rationale:** Core value — complete file with feedback
**Delivers:** Multi-peer download loop, progress display

### Phase 6: Graceful Shutdown
**Rationale:** Usable tool on real torrents
**Delivers:** Ctrl+C handling, clean exit

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | BEP 3 + Go stdlib well documented |
| Features | HIGH | Reference architecture + PROJECT.md scope |
| Architecture | HIGH | Reference repo validates module split |
| Pitfalls | HIGH | Classic gotchas documented widely |

**Overall confidence:** HIGH

---
*Research completed: 2026-07-10*
*Ready for roadmap: yes*
