package peer

import (
	"fmt"
	"io"
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

// ExchangeHandshake sends our handshake and reads the peer's, verifying info_hash.
func ExchangeHandshake(rw io.ReadWriter, expected [20]byte, ours Handshake) (Handshake, error) {
	if _, err := rw.Write(ours.Serialize()); err != nil {
		return Handshake{}, fmt.Errorf("handshake write: %w", err)
	}

	buf := make([]byte, handshakeLength)
	if _, err := io.ReadFull(rw, buf); err != nil {
		return Handshake{}, fmt.Errorf("handshake read: %w", err)
	}

	peer, err := Deserialize(buf)
	if err != nil {
		return Handshake{}, err
	}
	if peer.InfoHash != expected {
		return Handshake{}, fmt.Errorf("handshake: info_hash mismatch")
	}
	return peer, nil
}
