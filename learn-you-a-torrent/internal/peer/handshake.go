package peer

import (
	"fmt"
)

const (
	protocolString = "BitTorrent protocol"
	pstrLen        = 19
	reservedLen    = 8
	handshakeLength = 1 + pstrLen + reservedLen + 20 + 20
)

// Handshake is the BitTorrent peer wire handshake payload.
type Handshake struct {
	InfoHash [20]byte
	PeerID   [20]byte
}

// Serialize encodes the handshake to its 68-byte wire format.
func (h Handshake) Serialize() []byte {
	buf := make([]byte, handshakeLength)
	buf[0] = pstrLen
	copy(buf[1:20], protocolString)
	copy(buf[28:48], h.InfoHash[:])
	copy(buf[48:68], h.PeerID[:])
	return buf
}

// Deserialize decodes a 68-byte handshake from wire format.
func Deserialize(data []byte) (Handshake, error) {
	if len(data) != handshakeLength {
		return Handshake{}, fmt.Errorf("handshake: expected %d bytes, got %d", handshakeLength, len(data))
	}
	if data[0] != pstrLen {
		return Handshake{}, fmt.Errorf("handshake: invalid pstrlen %d", data[0])
	}
	if string(data[1:20]) != protocolString {
		return Handshake{}, fmt.Errorf("handshake: invalid protocol string")
	}

	var h Handshake
	copy(h.InfoHash[:], data[28:48])
	copy(h.PeerID[:], data[48:68])
	return h, nil
}
