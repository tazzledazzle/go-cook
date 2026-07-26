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

func TestConnection_sendInterested(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	go func() {
		msg, err := ReadMessage(server)
		if err != nil {
			return
		}
		if msg.ID != MsgInterested {
			t.Errorf("message ID = %d, want %d", msg.ID, MsgInterested)
		}
	}()

	conn := NewConnection(client, minimalInfoHash(), Handshake{InfoHash: minimalInfoHash(), PeerID: testPeerID()})
	if err := conn.SendInterested(); err != nil {
		t.Fatalf("SendInterested() error = %v", err)
	}
}

func TestConnection_sendRequest(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	go func() {
		msg, err := ReadMessage(server)
		if err != nil {
			return
		}
		index, begin, length, err := ParseRequest(msg)
		if err != nil {
			t.Errorf("ParseRequest() error = %v", err)
			return
		}
		if index != 0 || begin != 0 || length != 12 {
			t.Errorf("request = (%d, %d, %d), want (0, 0, 12)", index, begin, length)
		}
	}()

	conn := NewConnection(client, minimalInfoHash(), Handshake{InfoHash: minimalInfoHash(), PeerID: testPeerID()})
	if err := conn.SendRequest(0, 0, 12); err != nil {
		t.Fatalf("SendRequest() error = %v", err)
	}
}
