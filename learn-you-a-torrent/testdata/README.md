# testdata fixtures

## minimal.torrent

Single-file torrent for Phase 1 golden tests.

| Field | Value |
|-------|-------|
| announce | `http://example.com/announce` |
| info.name | `test.txt` |
| info.length | 12 |
| info.piece length | 16384 |
| info.pieces | 1 × 20-byte SHA1 |
| Piece 0 SHA1 | `98aaed442721e0cecdd9bf7b8cbb3e1ff1b1536a` |
| File content (conceptual) | `hello world\n` (12 bytes, zero-padded to 16384 for piece hash) |

## Info dictionary

| Property | Value |
|----------|-------|
| Info dict offset | 47 |
| Info dict length | 83 bytes |
| Expected info_hash | `90dc02db9bd6d3808cbfdbba2633f4c6af7180f0` |

Regenerate with:

```bash
go run testdata/gen_minimal.go
```
