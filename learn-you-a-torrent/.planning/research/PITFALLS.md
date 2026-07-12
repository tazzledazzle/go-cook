# Pitfalls Research

**Domain:** BitTorrent client implementation in Go
**Researched:** 2026-07-10
**Confidence:** HIGH

## Critical Pitfalls

### Pitfall 1: Wrong Info Hash

**What goes wrong:** Tracker returns 0 peers or peers reject handshake — info hash mismatch.

**Why it happens:** Re-encoding the info dict instead of hashing the **exact raw bytes** from the `.torrent` file. Key ordering and string encoding matter.

**How to avoid:** During bencode decode, capture start/end offsets of the info dictionary. Hash `data[start:end]` directly with SHA1 — do not round-trip through structs.

**Warning signs:** Handshake succeeds with wrong peers but no data; tracker `peers` empty on valid torrent.

**Phase to address:** Phase 1 (Parse & Info Hash)

---

### Pitfall 2: Info Hash URL Encoding in Tracker Request

**What goes wrong:** Tracker returns HTTP 400 or garbage response.

**Why it happens:** `info_hash` must be URL-encoded as raw 20 bytes (`%a1%b2...`), not hex string.

**How to avoid:** Use proper URL encoding for binary values in query string; verify against known-good announce URL.

**Warning signs:** Tracker response is HTML error page or empty dict.

**Phase to address:** Phase 2 (Tracker Announce)

---

### Pitfall 3: Handshake Byte Order / Length

**What goes wrong:** Connection reset immediately after dial.

**Why it happens:** Handshake is exactly 68 bytes: `pstrlen(1) + "BitTorrent protocol"(19) + reserved(8) + info_hash(20) + peer_id(20)`. Off-by-one on pstrlen (must be 19).

**How to avoid:** Write struct with fixed-size fields; test serialize/deserialize roundtrip.

**Warning signs:** Peer closes connection within first read.

**Phase to address:** Phase 3 (Peer Handshake)

---

### Pitfall 4: Message Length Prefix

**What goes wrong:** Desync on message stream; parse garbage message IDs.

**Why it happens:** Every message is `4-byte big-endian length + 1-byte id + payload`. Reading payload without length causes drift.

**How to avoid:** Always read 4 bytes first; if length=0, keepalive (no id byte). Read exactly `length` bytes for payload.

**Warning signs:** Message ID > 8 or nonsensical piece indices.

**Phase to address:** Phase 3–4 (Peer messages)

---

### Pitfall 5: Requesting Before Unchoke

**What goes wrong:** Peer stops responding or disconnects.

**Why it happens:** BitTorrent requires: send Interested → wait for Unchoke → then Request blocks.

**How to avoid:** State machine per peer: track choked/interested/unchoke flags.

**Warning signs:** Requests sent but zero Piece messages received.

**Phase to address:** Phase 4 (Download Piece)

---

### Pitfall 6: Block Size Assumptions

**What goes wrong:** Last block of piece fails; hash mismatch on final piece.

**Why it happens:** Blocks are 16 KB (2^14) except last block of each piece may be shorter.

**How to avoid:** `blockLength = min(16384, pieceLength - begin)`.

**Warning signs:** SHA1 fail only on last piece or last block.

**Phase to address:** Phase 4 (Piece assembly)

---

### Pitfall 7: Piece Hash Mismatch After "Complete" Assembly

**What goes wrong:** Piece marked complete but hash fails; infinite re-download loop.

**Why it happens:** Missing block, duplicate block overwrite, wrong begin offset in piece buffer.

**How to avoid:** Track per-block bitmap; only validate when all blocks received; log which block indices filled.

**Warning signs:** Intermittent hash failure on same piece index.

**Phase to address:** Phase 4–5

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Single peer only | Simpler concurrency | Slow downloads | Phase 4–5 MVP |
| Sequential piece download | Easy debug | Very slow | Phase 4 only |
| No request pipelining | Simple send loop | Underutilizes bandwidth | v1 |
| Skip preallocation | Faster file create | Fragmented disk | v1 single-file |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| HTTP tracker | Forgetting `compact=1` | Request compact 6-byte peer format |
| HTTP tracker | Wrong `left` parameter | Set `left` = total file size initially |
| TCP peer | Not setting read deadline | Use deadlines to avoid hung goroutines |
| TCP peer | Little-endian for message length | Big-endian uint32 |

## "Looks Done But Isn't" Checklist

- [ ] **Download complete:** All piece hashes verified, not just byte count matches
- [ ] **Tracker:** Handles both compact and dictionary peer formats (or explicitly compact-only with test)
- [ ] **Progress:** Based on verified pieces, not requested bytes
- [ ] **Shutdown:** In-flight pieces reset or persisted; no corrupt trailing bytes in output file

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| Wrong info hash | Phase 1 | Unit test against known torrent hash |
| Tracker encoding | Phase 2 | Mock server returns peers |
| Handshake format | Phase 3 | Hex dump test of 68 bytes |
| Unchoke ordering | Phase 4 | Integration test with one peer |
| Block size | Phase 4 | Test torrent with non-aligned piece length |
| Piece hash | Phase 4–5 | Corrupt block → must fail validate |

## Sources

- [BitTorrentClient ARCHITECTURE.md](https://github.com/amaydixit11/BitTorrentClient/blob/main/ARCHITECTURE.md)
- BEP 3 specification
- Common build-your-own-torrent-client writeups

---
*Pitfalls research for: BitTorrent client*
*Researched: 2026-07-10*
