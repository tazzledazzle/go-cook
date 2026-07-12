# Phase 2 Research: Tracker Announce

**Researched:** 2026-07-10
**Confidence:** HIGH

## HTTP Announce Request (BEP 3)

GET `{announce}?info_hash=...&peer_id=...&port=...&uploaded=0&downloaded=0&left=...&compact=1&event=started`

| Param | Value (v1 download) |
|-------|---------------------|
| info_hash | 20 raw bytes, percent-encoded |
| peer_id | 20 bytes, percent-encoded |
| port | 6881 (uint16) |
| uploaded | 0 |
| downloaded | 0 |
| left | bytes remaining (file length on first announce) |
| compact | 1 |
| event | started (optional but common on first announce) |

### info_hash encoding (Pitfall 2)

Go approach that works:
```go
values := url.Values{}
values.Set("info_hash", string(infoHash[:])) // binary string; Encode() percent-escapes each byte
```

Golden test: known 20-byte hash → exact query substring `%90%DC%02%DB...` (uppercase hex per Go's QueryEscape).

## Tracker Response

Bencoded dictionary:
```
d8:intervali1800e5:peers12:......e
```

With `compact=1`, `peers` is a **string** (not list) of 6-byte chunks:
- bytes 0-3: IPv4 address (big-endian / network order)
- bytes 4-5: port (big-endian uint16)

Example one peer 192.168.1.100:6881:
`c0 a8 01 64 1a e1` → 192.168.1.100:6881

## TDD Test Fixtures

### Hand-crafted announce URL golden
Use info hash from `testdata/minimal.torrent`: `90dc02db9bd6d3808cbfdbba2633f4c6af7180f0`

### Hand-crafted compact peers
```go
// 127.0.0.1:6881
compact := []byte{127, 0, 0, 1, 0x1a, 0xe1}
```

### Mock httptest response body
Build bencode manually or use small helper:
`d6:peers6:<6 bytes>e` for one peer (length 6 string value)

## Package API (target)

```go
type Peer struct {
    IP   net.IP
    Port uint16
}

func BuildAnnounceURL(base string, req AnnounceRequest) (string, error)
func ParseCompactPeers(data []byte) ([]Peer, error)

type Client struct { HTTPClient *http.Client; PeerID [20]byte }
func (c *Client) Announce(t *torrent.Torrent) ([]Peer, error)
```

## Build Order (TDD slices)

1. **Announce URL** — pure function, no HTTP, golden query string test
2. **Compact peers parser** — pure function, table-driven
3. **Client.Announce** — httptest integration, decodes bencode response

## Sources

- BEP 3 tracker HTTP protocol
- `.planning/research/PITFALLS.md`
- Phase 1 golden hash in `testdata/README.md`

---
*Phase 2 research complete*
