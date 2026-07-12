# Phase 4 Research: Download One Piece

**Researched:** 2026-07-10
**Confidence:** HIGH

## Request Message (id=6)

Payload: 12 bytes — three big-endian uint32:

| Field | Size | Description |
|-------|------|-------------|
| index | 4 | Piece index |
| begin | 4 | Byte offset within piece |
| length | 4 | Block length (typically 16384, shorter for last block) |

## Piece Message (id=7)

Payload: 8 + block bytes:

| Field | Size | Description |
|-------|------|-------------|
| index | 4 | Piece index |
| begin | 4 | Byte offset within piece |
| block | variable | Block data |

## Download State Machine (single peer)

```
TCP connect → handshake → read bitfield → send interested → wait unchoke →
for each block: send request → read piece message → WriteBlock(begin, block) →
Validate(SHA1) → WritePiece to disk
```

**Pitfall 5:** Request only after Unchoke received.  
**Pitfall 6:** Last block length = `min(16384, pieceLength - begin)`.

## minimal.torrent Fixture

| Field | Value |
|-------|-------|
| info.name | `test.txt` |
| info.length | 12 |
| info.piece length | 16384 |
| File content | `hello world\n` (12 bytes) |
| Piece 0 buffer | 12 bytes content + zero padding to 16384 |
| Piece 0 SHA1 | `98aaed442721e0cecdd9bf7b8cbb3e1ff1b1536a` |
| Blocks needed | 1 (begin=0, length=12) |

Regenerate piece hash:

```bash
python3 -c "import hashlib; p=b'hello world\n'+b'\x00'*(16384-12); print(hashlib.sha1(p).hexdigest())"
```

## Block Size Constant

```go
const BlockSize = 1 << 14 // 16384
```

## Testing Strategy

| Layer | Test approach |
|-------|---------------|
| Piece buffer | Pure unit tests — WriteBlock, Complete, Bytes |
| Validation | Golden SHA1 from minimal.torrent; corrupt byte → Validate false, Reset works |
| File writer | tempdir + os.ReadFile; assert 12 bytes at offset 0 |
| Request/Piece wire | bytes.Buffer roundtrip |
| Integration | net.Pipe mock peer: handshake → bitfield → unchoke → respond to Request with Piece |

## Mock Peer Integration Script (conceptual)

```go
go func() {
  read client handshake; write matching handshake
  WriteMessage(bitfield); WriteMessage(unchoke)
  for {
    msg := ReadMessage(client)
    if msg.ID == MsgRequest {
      idx, begin, length := parseRequest(msg)
      block := pieceData[begin:begin+length]
      WriteMessage(client, buildPiece(idx, begin, block))
    }
  }
}()
client: DownloadPiece(conn, torrent, 0, writer)
```

## TDD Build Order (vertical slice)

1. Piece buffer assembly (`pieces/piece.go`)
2. SHA1 validate + reset
3. File writer
4. Request/Piece helpers + Connection send methods
5. DownloadPiece integration with mock peer

## Sources

- BEP 3 — request/piece messages
- `.planning/research/PITFALLS.md` Pitfalls 5–7
- `testdata/gen_minimal.go` — fixture generation

---
*Phase 4 research complete*
