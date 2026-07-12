# Walking Skeleton — Learn You a BitTorrent Client

**Phase:** 1
**Generated:** 2026-07-10

## Capability Proven End-To-End

A developer can run `go test ./internal/torrent/...` against a committed `testdata/minimal.torrent` fixture and get the correct 20-byte info hash — proving bencode decode → raw info extraction → SHA1 works before any network code exists.

## Architectural Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go 1.22+ | Matches repo; stdlib covers SHA1, I/O, testing |
| Module path | `github.com/tazzledazzle/go-cook/learn-you-a-torrent` | Under go-cook monorepo |
| Layout | `cmd/torrent/` + `internal/{bencode,torrent,...}` | Standard Go; internal packages not importable externally |
| Bencode | Hand-rolled decoder | Core learning objective; no jackpal/bencode-go |
| Info hash | SHA1 of raw info dict byte slice | BEP 3 requirement; avoids re-encode pitfall |
| Testing | TDD with table-driven tests + golden fixture | Fast CI, no network |
| Dependencies | stdlib + testify/assert optional | Minimal surface |

## Stack Touched in Phase 1

- [x] Project scaffold (`go.mod`, directory layout)
- [x] Bencode decode — real parser, not stub
- [x] Torrent metadata — real `.torrent` parse from bytes
- [x] Crypto — real SHA1 info hash
- [x] Test fixture — committed `testdata/minimal.torrent`
- [ ] CLI — deferred to Phase 5 (`cmd/torrent/main.go` stub only if needed)

## Out of Scope (Deferred to Later Slices)

- HTTP tracker announce (Phase 2)
- TCP peer wire protocol (Phase 3)
- Piece download and disk writes (Phase 4–5)
- Multi-file torrent `files` list (v2)
- Bencode encoder (only decode needed for v1; minimal encode optional for tests)
- Seeding, DHT, magnets

## Subsequent Slice Plan

Each later phase adds one vertical slice without altering Phase 1 decisions:

- **Phase 2:** Tracker announce using info hash from Phase 1 parser
- **Phase 3:** Peer handshake using same info hash
- **Phase 4:** Download one verified piece to disk
- **Phase 5:** Full download CLI with progress
- **Phase 6:** Graceful shutdown
