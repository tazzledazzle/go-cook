# Stack Research

**Domain:** BitTorrent client (Go, from-scratch learning project)
**Researched:** 2026-07-10
**Confidence:** HIGH

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.22+ | Language/runtime | Matches repo; excellent stdlib for TCP, HTTP, crypto/sha1, testing |
| `crypto/sha1` | stdlib | Piece hash + info hash | BEP 3 requires SHA1; no external dep |
| `net` / `net/http` | stdlib | Peer TCP + tracker HTTP | Wire protocol and announce are plain TCP/HTTP |
| `encoding` (custom) | — | Bencode | Implement yourself — core learning objective |
| `testing` + `testify/assert` | stdlib + v1.9+ | TDD | assert simplifies test readability; optional |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/stretchr/testify` | v1.9.0+ | Test assertions | All TDD slices — cleaner failure messages |
| None for bencode | — | — | Do NOT import bencode libraries |
| None for wire protocol | — | — | Do NOT import `anacrolix/torrent` etc. for v1 |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `go test ./...` | Unit/integration tests | Run per slice |
| `go test -race ./...` | Concurrency bugs | Run once peer goroutines exist |
| `golangci-lint` | Lint | Optional; keep minimal for learning |
| `bittorrent-test-fixtures/` | Synthetic `.torrent` bytes | Commit tiny fixtures for CI |

## Installation

```bash
cd learn-you-a-torrent
go mod init github.com/tazzledazzle/go-cook/learn-you-a-torrent  # or your module path
go get github.com/stretchr/testify@v1.9.0
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Stdlib-only core | `anacrolix/torrent` | Never for v1 — defeats learning goal |
| Hand-rolled bencode | `jackpal/bencode-go` | Only if blocked >2 days on bencode edge cases |
| testify | stdlib only | If zero-deps is a hard constraint |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Full torrent libraries (`anacrolix/torrent`, `cenkalti/rain`) | Hides all learning layers | Implement modules per architecture doc |
| `encoding/json` for `.torrent` | Wrong format — torrents are bencode | Custom bencode decoder |
| MD5/SHA256 for piece verify | BEP 3 specifies SHA1 for pieces | `crypto/sha1` |
| Gorilla/mux for tracker | Tracker is a single GET URL | `net/http` client |

## Stack Patterns by Variant

**If TDD-first vertical slice:**
- Each package gets `_test.go` before implementation
- Use table-driven tests for bencode decode/encode roundtrips
- Golden-file tests for `.torrent` parsing

**If single-file v1 only:**
- Skip multi-file `file/mapper.go` complexity until v2
- `file/writer.go` writes to one path from `info.name`

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| Go 1.22+ | testify v1.9+ | Works out of box |
| Go 1.22+ | sha1 stdlib | SHA1 still required by BEP 3 despite deprecation elsewhere |

## Sources

- [BitTorrentClient ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md) — module layout reference
- [BEP 3 — Peer Wire Protocol](http://bittorrent.org/beps/bep_0003.html) — protocol spec
- [BEP 12 — Multitracker](http://bittorrent.org/beps/bep_0012.html) — announce-list (v2)

---
*Stack research for: BitTorrent client in Go*
*Researched: 2026-07-10*
