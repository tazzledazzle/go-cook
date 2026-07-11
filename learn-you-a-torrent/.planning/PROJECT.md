# Learn You a BitTorrent Client

## What This Is

A personal, from-scratch BitTorrent client in Go built as a learning project inside `learn-you-a-torrent/`. The goal is to understand how torrents work end-to-end — bencode parsing, tracker communication, peer wire protocol, piece assembly, and disk writes — by implementing each layer yourself with tests first. The [reference architecture](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md) defines the module boundaries and data flow; code is written independently and referenced only when stuck.

## Core Value

You can point the CLI at a single-file `.torrent`, watch it download with progress output, verify pieces by SHA1, write the file to disk, and shut down gracefully — and you understand every layer that made it happen.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Parse `.torrent` files and compute the 20-byte info hash
- [ ] Announce to HTTP tracker and receive a peer list
- [ ] Perform BitTorrent handshake and exchange wire-protocol messages
- [ ] Download, assemble, and SHA1-verify pieces (16 KB blocks)
- [ ] Write verified pieces to disk for single-file torrents
- [ ] Display CLI progress (%, speed, peers connected)
- [ ] Handle graceful shutdown on Ctrl+C
- [ ] TDD-first: each vertical slice ships with tests before implementation

### Out of Scope

- Seeding / uploading — download-only v1; simplifies peer state machine
- DHT / magnet links — tracker-based discovery only for v1
- Multi-file torrents — deferred to v2 after single-file path works
- Protocol encryption (MSE/PE) — plain wire protocol for clarity
- GUI / web UI — CLI only

## Context

- Lives in `go-cook/learn-you-a-torrent/`, a Go portfolio/learning repo alongside other experiments (`design-kube`, `first-go-application`)
- Reference blueprint: [BitTorrentClient ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md) — modules: `bencode/`, `torrent/`, `tracker/`, `peer/`, `pieces/`, `file/`, orchestrated from `main.go`
- Build approach: **vertical slice MVP** (each phase delivers a testable increment toward a real download), not horizontal "finish all of bencode then all of tracker"
- Verification: synthetic `.torrent` fixtures for unit/integration tests + optional live public torrent (e.g. Ubuntu/Debian ISO) for manual end-to-end check
- README skeleton already mirrors the architecture doc sections (System Overview, Data Flow, Module Architecture, File Reference)

## Constraints

- **Language**: Go — matches repo and reference implementation
- **Dependencies**: Minimize external deps; stdlib preferred for wire protocol, HTTP tracker, bencode (implement bencode yourself)
- **Protocol**: BitTorrent peer wire protocol (BEP 3) — choke/unchoke, bitfield, request/piece
- **Testing**: TDD-first — write failing tests, implement, refactor per slice
- **Legal**: Integration tests use synthetic fixtures; live tests use legitimate public torrents only

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Learn-by-building, reference when stuck | Deep understanding over copy-paste | — Pending |
| Vertical slice MVP + TDD | Each phase proves progress; tests document behavior | — Pending |
| Single-file torrents in v1 | Simplest path to "it downloads" | — Pending |
| Download-only (no seeding) | Cuts upload state machine complexity for v1 | — Pending |
| Synthetic fixtures + live manual test | Fast CI + confidence on real torrents | — Pending |
| Module layout follows reference architecture | Proven decomposition; maps 1:1 to learning topics | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-07-10 after initialization*
