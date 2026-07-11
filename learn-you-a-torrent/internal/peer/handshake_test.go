package peer

import (
	"bytes"
	"testing"
)

func minimalInfoHash() [20]byte {
	return [20]byte{
		0x90, 0xdc, 0x02, 0xdb, 0x9b, 0xd6, 0xd3, 0x80,
		0x8c, 0xbf, 0xdb, 0xba, 0x26, 0x33, 0xf4, 0xc6,
		0xaf, 0x71, 0x80, 0xf0,
	}
}

func testPeerID() [20]byte {
	var id [20]byte
	copy(id[:], []byte("-GO0001-000000"))
	return id
}

func TestHandshake_serializeLength(t *testing.T) {
	h := Handshake{InfoHash: minimalInfoHash(), PeerID: testPeerID()}
	got := h.Serialize()
	if len(got) != handshakeLength {
		t.Fatalf("Serialize() length = %d, want %d", len(got), handshakeLength)
	}
}

func TestHandshake_roundtrip(t *testing.T) {
	want := Handshake{InfoHash: minimalInfoHash(), PeerID: testPeerID()}
	got, err := Deserialize(want.Serialize())
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}
	if got.InfoHash != want.InfoHash {
		t.Errorf("InfoHash = %x, want %x", got.InfoHash, want.InfoHash)
	}
	if got.PeerID != want.PeerID {
		t.Errorf("PeerID = %q, want %q", got.PeerID, want.PeerID)
	}
}

func TestHandshake_deserializeRejectsShortBuffer(t *testing.T) {
	_, err := Deserialize(bytes.Repeat([]byte{0}, handshakeLength-1))
	if err == nil {
		t.Fatal("Deserialize() expected error for short buffer, got nil")
	}
}
