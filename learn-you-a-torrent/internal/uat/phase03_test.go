package uat

import (
	"bytes"
	"io"
	"net"
	"testing"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/peer"
)

func minimalInfoHashBytes() [20]byte {
	return [20]byte{
		0x90, 0xdc, 0x02, 0xdb, 0x9b, 0xd6, 0xd3, 0x80,
		0x8c, 0xbf, 0xdb, 0xba, 0x26, 0x33, 0xf4, 0xc6,
		0xaf, 0x71, 0x80, 0xf0,
	}
}

func peerPeerID() [20]byte {
	var id [20]byte
	copy(id[:], []byte("-PE0001-000000000"))
	return id
}

// UAT 3.1 — Handshake wire format roundtrip

func TestUAT_3_1a_handshakeSerializeIs68Bytes(t *testing.T) {
	h := peer.Handshake{InfoHash: minimalInfoHashBytes(), PeerID: testPeerID()}
	if len(h.Serialize()) != 68 {
		t.Fatalf("handshake length = %d, want 68", len(h.Serialize()))
	}
}

func TestUAT_3_1b_handshakeRejectsShortBuffer(t *testing.T) {
	if _, err := peer.Deserialize(bytes.Repeat([]byte{0}, 67)); err == nil {
		t.Fatal("Deserialize() expected error for 67-byte buffer, got nil")
	}
}

// UAT 3.2 — Successful handshake with matching info hash

func TestUAT_3_2a_handshakeExchangeSucceeds(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	hash := minimalInfoHashBytes()
	ours := peer.Handshake{InfoHash: hash, PeerID: testPeerID()}
	theirs := peer.Handshake{InfoHash: hash, PeerID: peerPeerID()}

	go func() {
		buf := make([]byte, 68)
		_, _ = io.ReadFull(server, buf)
		_, _ = server.Write(theirs.Serialize())
	}()

	got, err := peer.ExchangeHandshake(client, hash, ours)
	if err != nil {
		t.Fatalf("ExchangeHandshake() error = %v", err)
	}
	if got.PeerID != theirs.PeerID {
		t.Errorf("PeerID = %q, want %q", got.PeerID, theirs.PeerID)
	}
}

func TestUAT_3_2b_handshakeRoundtripPreservesFields(t *testing.T) {
	want := peer.Handshake{InfoHash: minimalInfoHashBytes(), PeerID: testPeerID()}
	got, err := peer.Deserialize(want.Serialize())
	if err != nil {
		t.Fatalf("Deserialize() error = %v", err)
	}
	if got.InfoHash != want.InfoHash || got.PeerID != want.PeerID {
		t.Fatalf("roundtrip mismatch: got %+v want %+v", got, want)
	}
}

// UAT 3.3 — Mismatched info hash rejected

func TestUAT_3_3a_handshakeRejectsWrongHash(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	expected := minimalInfoHashBytes()
	ours := peer.Handshake{InfoHash: expected, PeerID: testPeerID()}
	wrong := peer.Handshake{InfoHash: [20]byte{0x01}, PeerID: peerPeerID()}

	go func() {
		buf := make([]byte, 68)
		_, _ = io.ReadFull(server, buf)
		_, _ = server.Write(wrong.Serialize())
	}()

	if _, err := peer.ExchangeHandshake(client, expected, ours); err == nil {
		t.Fatal("ExchangeHandshake() expected error for hash mismatch, got nil")
	}
}

func TestUAT_3_3b_handshakeMismatchClosesConnection(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	expected := minimalInfoHashBytes()
	go func() {
		buf := make([]byte, 68)
		_, _ = io.ReadFull(server, buf)
		_, _ = server.Write(peer.Handshake{InfoHash: [20]byte{0xff}, PeerID: peerPeerID()}.Serialize())
	}()

	_, err := peer.ExchangeHandshake(client, expected, peer.Handshake{InfoHash: expected, PeerID: testPeerID()})
	if err == nil {
		t.Fatal("expected handshake failure")
	}
}

// UAT 3.4 — Wire message framing and parsing

func TestUAT_3_4a_messageRoundtripCoreIDs(t *testing.T) {
	ids := []byte{peer.MsgChoke, peer.MsgUnchoke, peer.MsgInterested, peer.MsgHave, peer.MsgBitfield}
	for _, id := range ids {
		msg := peer.Message{ID: id, Payload: []byte{0x01}}
		if id == peer.MsgBitfield {
			msg.Payload = []byte{0xff}
		}
		var buf bytes.Buffer
		if err := peer.WriteMessage(&buf, msg); err != nil {
			t.Fatalf("WriteMessage(%d) error = %v", id, err)
		}
		got, err := peer.ReadMessage(&buf)
		if err != nil {
			t.Fatalf("ReadMessage(%d) error = %v", id, err)
		}
		if got.ID != id {
			t.Errorf("ID = %d, want %d", got.ID, id)
		}
	}
}

func TestUAT_3_4b_readMessageErrorsOnTruncation(t *testing.T) {
	if _, err := peer.ReadMessage(bytes.NewReader([]byte{0, 0, 0, 2, peer.MsgChoke})); err == nil {
		t.Fatal("ReadMessage() expected error on truncated payload, got nil")
	}
}

// UAT 3.5 — Connection reads bitfield and unchoke

func TestUAT_3_5a_connectionReadsBitfieldThenUnchoke(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	hash := minimalInfoHashBytes()
	go func() {
		buf := make([]byte, 68)
		_, _ = io.ReadFull(server, buf)
		_, _ = server.Write(peer.Handshake{InfoHash: hash, PeerID: peerPeerID()}.Serialize())
		_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgBitfield, Payload: []byte{0xff}})
		_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgUnchoke})
	}()

	conn := peer.NewConnection(client, hash, peer.Handshake{InfoHash: hash, PeerID: testPeerID()})
	if err := conn.PerformHandshake(); err != nil {
		t.Fatalf("PerformHandshake() error = %v", err)
	}
	bitfield, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("first ReadMessage() error = %v", err)
	}
	if bitfield.ID != peer.MsgBitfield {
		t.Fatalf("first message ID = %d, want bitfield", bitfield.ID)
	}
	unchoke, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("second ReadMessage() error = %v", err)
	}
	if unchoke.ID != peer.MsgUnchoke {
		t.Fatalf("second message ID = %d, want unchoke", unchoke.ID)
	}
}

func TestUAT_3_5b_connectionSendInterested(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	hash := minimalInfoHashBytes()
	go func() {
		buf := make([]byte, 68)
		_, _ = io.ReadFull(server, buf)
		_, _ = server.Write(peer.Handshake{InfoHash: hash, PeerID: peerPeerID()}.Serialize())
		msg, _ := peer.ReadMessage(server)
		if msg.ID != peer.MsgInterested {
			t.Errorf("server saw message ID = %d, want interested", msg.ID)
		}
	}()

	conn := peer.NewConnection(client, hash, peer.Handshake{InfoHash: hash, PeerID: testPeerID()})
	if err := conn.PerformHandshake(); err != nil {
		t.Fatalf("PerformHandshake() error = %v", err)
	}
	if err := conn.SendInterested(); err != nil {
		t.Fatalf("SendInterested() error = %v", err)
	}
}

// UAT 3.6 — Phase 3 race-safe suite

func TestUAT_3_6a_peerPackageTestsPass(t *testing.T) {
	runGoTest(t, "./internal/peer/...")
}

func TestUAT_3_6b_peerPackageRaceClean(t *testing.T) {
	runGoTest(t, "./internal/peer/...", "-race")
}
