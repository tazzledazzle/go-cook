# Feature Research

**Domain:** BitTorrent client (download-only, learning)
**Researched:** 2026-07-10
**Confidence:** HIGH

## Feature Landscape

### Table Stakes (Users Expect These)

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Parse `.torrent` file | Can't start without metadata | MEDIUM | Bencode + info dict |
| Tracker announce → peers | Only discovery path in v1 | MEDIUM | HTTP GET, compact peer format |
| Peer handshake | Gate to wire protocol | LOW | 68-byte fixed format |
| Download pieces via Request/Piece | Core transfer mechanism | HIGH | 16 KB blocks, state machine |
| SHA1 piece verification | Corrupt data is useless | LOW | Per-piece hash from `.torrent` |
| Write assembled file to disk | End goal | MEDIUM | Single-file: straightforward |
| Progress display | User feedback during long downloads | LOW | % complete, speed, peers |
| Graceful shutdown | Ctrl+C shouldn't corrupt partial file | MEDIUM | Signal handling, flush state |

### Differentiators (Competitive Advantage)

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Rarest-first piece selection | Better swarm performance | MEDIUM | Reference uses `pieces/selector.go` |
| Multi-peer parallel download | Speed | HIGH | Goroutine per peer |
| Resume / checkpoint | UX on large files | HIGH | Defer to v2 |
| Multitracker failover | Resilience | MEDIUM | Defer to v2 |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| Seeding in v1 | "Real clients upload" | Doubles state machine complexity | Download-only v1 |
| DHT / magnets first | Modern discovery | Huge scope; tracker is simpler | HTTP tracker for v1 |
| Full bencode encoder first | "Complete the module" | Info hash only needs decode + raw bytes | Decode + extract raw info dict |
| GUI early | Visual progress | Distraction from protocol learning | CLI progress bar |

## Feature Dependencies

```
Parse .torrent
    └──requires──> Bencode decoder
    └──requires──> Info hash (SHA1 of raw info dict)

Tracker announce
    └──requires──> Info hash + peer_id

Peer handshake
    └──requires──> Info hash + TCP dial

Download piece
    └──requires──> Handshake + bitfield/have + unchoke + request/piece

Write to disk
    └──requires──> Verified complete piece

Progress display
    └──requires──> Piece manager state
```

## MVP Definition

### Launch With (v1)

- [x] Bencode decode + info hash — foundation for everything
- [x] Tracker announce (single tracker) — peer discovery
- [x] Handshake + message loop — wire protocol
- [x] Single piece download end-to-end — proves the stack
- [x] Full file download + verify + write — core value
- [x] CLI progress + graceful shutdown — usable tool

### Add After Validation (v1.x)

- [ ] Multi-file torrents — after single-file works
- [ ] Multitracker / announce-list — resilience
- [ ] Resume from partial download — checkpoint file

### Future Consideration (v2+)

- [ ] Seeding / upload accounting
- [ ] DHT / magnet links
- [ ] Protocol encryption

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Parse torrent + info hash | HIGH | MEDIUM | P1 |
| Tracker announce | HIGH | MEDIUM | P1 |
| Handshake | HIGH | LOW | P1 |
| Download one piece | HIGH | HIGH | P1 |
| Full download + verify | HIGH | HIGH | P1 |
| Progress CLI | MEDIUM | LOW | P1 |
| Graceful shutdown | MEDIUM | MEDIUM | P1 |
| Rarest-first selection | MEDIUM | MEDIUM | P2 |
| Multi-file | MEDIUM | HIGH | P2 |
| Seeding | LOW (v1) | HIGH | P3 |

## Sources

- [BitTorrentClient ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md)
- BEP 3 peer wire protocol specification

---
*Feature research for: BitTorrent client*
*Researched: 2026-07-10*
