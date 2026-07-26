# Phase 1 Research: Bencode & Torrent Parsing

**Researched:** 2026-07-10
**Confidence:** HIGH

## Objective

Answer: What do we need to know to implement bencode decoding, torrent parsing, and correct info hash calculation in Go with TDD?

## Bencode Format (BEP 3)

| Type | Wire format | Example |
|------|-------------|---------|
| String | `{len}:{bytes}` | `4:spam` → `"spam"` |
| Integer | `i{num}e` | `i3e` → `3`, `i-3e` → `-3` |
| List | `l{items}e` | `li1ei2ei3ee` → `[1,2,3]` |
| Dict | `d{key}{value}...e` | Keys must be strings; order preserved in file |

**Decoder requirements:**
- Track `pos` cursor through input bytes
- Return decoded Go value + bytes consumed
- For torrent parsing: when key is `"info"`, record `infoStart` before value decode and `infoEnd` after (exclusive end = start of byte after closing `e` of info dict)

## Info Hash Calculation

```
info_hash = SHA1(raw_bytes[infoStart:infoEnd])
```

**Critical:** The slice includes the leading `d` and trailing `e` of the info dictionary. Do NOT:
- Re-serialize from a Go struct
- Sort dictionary keys
- Use JSON or any other encoding

**Verification:** Compare against `transmission-show -m file.torrent` or Python:
```python
import bencodepy, hashlib
data = open("file.torrent","rb").read()
# parse to find info dict offsets manually or use library that exposes raw
```

For CI, commit a fixture with a precomputed golden hash constant in the test.

## Torrent File Structure

Top-level dict keys (typical):
- `announce` — string URL
- `info` — dict with:
  - `name` — string (filename or directory name)
  - `length` — integer (single-file mode)
  - `piece length` — integer (usually 16384 or 262144)
  - `pieces` — string of concatenated 20-byte SHA1 hashes

Multi-file uses `files` list instead of `length` — **out of scope for Phase 1**.

## Go Implementation Patterns

```go
// internal/bencode/decode.go
type Decoder struct {
    data []byte
    pos  int
}

func (d *Decoder) Decode() (interface{}, error)
func (d *Decoder) InfoDictBytes() []byte  // set during decode when "info" key seen
```

Use `interface{}` or typed unions for decoded values: `string`, `int64`, `[]interface{}`, `map[string]interface{}`.

Dictionary keys in Go: `map[string]interface{}` — iteration order is random but you only need values by key for torrent struct mapping.

## Test Fixture Strategy (TEST-02)

Create `testdata/minimal.torrent` — hand-crafted bencode for deterministic CI:

```
d8:announce35:http://example.com/announce4:infod6:lengthi12e4:name8:test.txt12:piece lengthi16384e6:pieces20:<20-byte-sha1>e
```

Generate golden info hash once, hardcode in `parser_test.go` as `[20]byte` or hex string.

Also add `testdata/bencode/` with raw snippets for unit tests (no file I/O needed for bencode package tests).

## Reference Implementation Notes

Peek-only reference: [amaydixit11/BitTorrentClient internal/bencode](https://github.com/amaydixit11/BitTorrentClient/tree/main/internal/bencode)

Key patterns from reference architecture:
- `Decoder` exposes `Pos` and `Data` for raw extraction
- `extractRawInfoDict()` in parser — separate from struct mapping
- `parseInfoFromMap()` converts decoded map to `Info` struct

## Validation Architecture

| Requirement | Test type | Command |
|-------------|-----------|---------|
| BENC-01 | Unit table-driven | `go test ./internal/bencode/... -v` |
| BENC-02 | Unit offset test | Assert `InfoDictBytes()` matches manual slice |
| TORR-01 | Unit golden file | `go test ./internal/torrent/... -run TestParseMinimal` |
| TORR-02 | Unit hash compare | Golden SHA1 hex in test |
| TEST-02 | Fixture exists | `test -f testdata/minimal.torrent` |

## Dependencies

- None external beyond `go mod init` and optional `testify/assert`
- Phase 2 depends on correct info hash from this phase

## Sources

- [BEP 3](http://bittorrent.org/beps/bep_0003.html)
- `.planning/research/PITFALLS.md` — Pitfall 1 (wrong info hash)
- [Reference ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md)

---
*Phase 1 research complete*
