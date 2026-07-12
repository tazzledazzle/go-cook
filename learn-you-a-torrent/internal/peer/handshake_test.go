package peer

import (
	"bytes"
	"io"
	"net"
	"strings"
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

func peerPeerID() [20]byte {
	var id [20]byte
	copy(id[:], []byte("-PE0001-000000000"))
	return id
}

func TestHandshakeExchange_success(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	expectedHash := minimalInfoHash()
	ours := Handshake{InfoHash: expectedHash, PeerID: testPeerID()}
	peerHS := Handshake{InfoHash: expectedHash, PeerID: peerPeerID()}

	go func() {
		buf := make([]byte, handshakeLength)
		if _, err := io.ReadFull(server, buf); err != nil {
			return
		}
		_, _ = server.Write(peerHS.Serialize())
	}()

	got, err := ExchangeHandshake(client, expectedHash, ours)
	if err != nil {
		t.Fatalf("ExchangeHandshake() error = %v", err)
	}
	if got.PeerID != peerHS.PeerID {
		t.Errorf("PeerID = %q, want %q", got.PeerID, peerHS.PeerID)
	}
}

func TestHandshakeExchange_rejectsMismatch(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	expectedHash := minimalInfoHash()
	ours := Handshake{InfoHash: expectedHash, PeerID: testPeerID()}
	wrong := Handshake{InfoHash: [20]byte{0x01}, PeerID: peerPeerID()}

	go func() {
		buf := make([]byte, handshakeLength)
		if _, err := io.ReadFull(server, buf); err != nil {
			return
		}
		_, _ = server.Write(wrong.Serialize())
	}()

	_, err := ExchangeHandshake(client, expectedHash, ours)
	if err == nil {
		t.Fatal("ExchangeHandshake() expected error for info_hash mismatch, got nil")
	}
	if !strings.Contains(err.Error(), "info_hash") {
		t.Fatalf("error = %q, want substring info_hash", err.Error())
	}
}
