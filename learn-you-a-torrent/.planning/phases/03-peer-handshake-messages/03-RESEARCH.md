# Phase 3 Research: Peer Wire Protocol

**Researched:** 2026-07-10
**Confidence:** HIGH

## Handshake (68 bytes)

| Offset | Size | Field |
|--------|------|-------|
| 0 | 1 | pstrlen = 19 |
| 1 | 19 | `"BitTorrent protocol"` |
| 20 | 8 | reserved (zeros) |
| 28 | 20 | info_hash |
| 48 | 20 | peer_id |

Both peers send handshake immediately after TCP connect.

## Messages

```
[length:4 big-endian][id:1 if length>0][payload:length-1]
```

| ID | Name | Payload |
|----|------|---------|
| 0 | choke | none |
| 1 | unchoke | none |
| 2 | interested | none |
| 3 | not interested | none |
| 4 | have | piece index (4 bytes BE) |
| 5 | bitfield | bitfield bytes |
| 6 | request | index, begin, length (12 bytes) |
| 7 | piece | index, begin, data |
| 8 | cancel | index, begin, length |

Keepalive: length=0, no id.

## Testing with net.Pipe

```go
clientConn, serverConn := net.Pipe()
go func() {
  // mock peer: write handshake bytes, then bitfield message, then unchoke
  serverConn.Write(handshakeBytes)
  WriteMessage(serverConn, Message{ID: MsgBitfield, Payload: ...})
  WriteMessage(serverConn, Message{ID: MsgUnchoke})
}()
// client: Connection on clientConn, Exchange handshake, ReadMessage x2
```

## TDD Build Order

1. Handshake serialize/deserialize (pure bytes)
2. Handshake exchange + info_hash verify (net.Pipe)
3. Message read/write (bytes.Buffer or Pipe)
4. Connection reads post-handshake bitfield + unchoke (integration)

## Sources

- BEP 3
- `.planning/research/PITFALLS.md` Pitfalls 3–4

---
*Phase 3 research complete*
