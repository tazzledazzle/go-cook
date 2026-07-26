<!-- GSD:project-start source:PROJECT.md -->
## Project

**Learn You a BitTorrent Client**

A personal, from-scratch BitTorrent client in Go built as a learning project inside `learn-you-a-torrent/`. The goal is to understand how torrents work end-to-end — bencode parsing, tracker communication, peer wire protocol, piece assembly, and disk writes — by implementing each layer yourself with tests first. The [reference architecture](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md) defines the module boundaries and data flow; code is written independently and referenced only when stuck.

**Core Value:** You can point the CLI at a single-file `.torrent`, watch it download with progress output, verify pieces by SHA1, write the file to disk, and shut down gracefully — and you understand every layer that made it happen.

### Constraints

- **Language**: Go — matches repo and reference implementation
- **Dependencies**: Minimize external deps; stdlib preferred for wire protocol, HTTP tracker, bencode (implement bencode yourself)
- **Protocol**: BitTorrent peer wire protocol (BEP 3) — choke/unchoke, bitfield, request/piece
- **Testing**: TDD-first — write failing tests, implement, refactor per slice
- **Legal**: Integration tests use synthetic fixtures; live tests use legitimate public torrents only
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->
## Technology Stack

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
- Each package gets `_test.go` before implementation
- Use table-driven tests for bencode decode/encode roundtrips
- Golden-file tests for `.torrent` parsing
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
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
