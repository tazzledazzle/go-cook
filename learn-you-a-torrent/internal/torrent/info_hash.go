package torrent

import "crypto/sha1"

// InfoHashFromBytes computes the BitTorrent info hash (SHA1 of raw info dict bytes).
func InfoHashFromBytes(raw []byte) [20]byte {
	return sha1.Sum(raw)
}

// InfoHashHex returns a lowercase hex string for hash.
func InfoHashHex(hash [20]byte) string {
	const hex = "0123456789abcdef"
	out := make([]byte, 40)
	for i, b := range hash {
		out[i*2] = hex[b>>4]
		out[i*2+1] = hex[b&0x0f]
	}
	return string(out)
}
