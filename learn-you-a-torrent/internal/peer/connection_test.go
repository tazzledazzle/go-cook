package peer

import (
	"bytes"
	"io"
	"net"
	"testing"
)

func TestConnection_readsBitfieldAndUnchoke(t *testing.T) {
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
		_ = WriteMessage(server, Message{ID: MsgBitfield, Payload: []byte{0xff}})
		_ = WriteMessage(server, Message{ID: MsgUnchoke})
	}()

	conn := NewConnection(client, expectedHash, ours)
	if err := conn.PerformHandshake(); err != nil {
		t.Fatalf("PerformHandshake() error = %v", err)
	}

	bitfield, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("first ReadMessage() error = %v", err)
	}
	if bitfield.ID != MsgBitfield {
		t.Fatalf("first message ID = %d, want %d", bitfield.ID, MsgBitfield)
	}
	if !bytes.Equal(bitfield.Payload, []byte{0xff}) {
		t.Fatalf("bitfield payload = %v, want [255]", bitfield.Payload)
	}

	unchoke, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("second ReadMessage() error = %v", err)
	}
	if unchoke.ID != MsgUnchoke {
		t.Fatalf("second message ID = %d, want %d", unchoke.ID, MsgUnchoke)
	}
}
