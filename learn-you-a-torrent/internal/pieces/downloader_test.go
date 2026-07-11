package pieces

import (
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/file"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/peer"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

func minimalTorrentPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "minimal.torrent")
}

func buildMinimalPieceData() []byte {
	data := make([]byte, 16384)
	copy(data, []byte("hello world\n"))
	return data
}

func testPeerID() [20]byte {
	var id [20]byte
	copy(id[:], []byte("-GO0001-000000"))
	return id
}

func TestDownloadPiece0_fromMockPeer(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}

	pieceData := buildMinimalPieceData()
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	peerHS := peer.Handshake{InfoHash: tor.InfoHash, PeerID: testPeerID()}

	go func() {
		buf := make([]byte, 68)
		if _, err := io.ReadFull(server, buf); err != nil {
			return
		}
		_, _ = server.Write(peerHS.Serialize())
		_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgBitfield, Payload: []byte{0xff}})
		_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgUnchoke})

		for {
			msg, err := peer.ReadMessage(server)
			if err != nil {
				return
			}
			switch msg.ID {
			case peer.MsgInterested:
				continue
			case peer.MsgRequest:
				index, begin, length, err := peer.ParseRequest(msg)
				if err != nil {
					return
				}
				block := pieceData[begin : begin+length]
				payload := make([]byte, 8+len(block))
				payload[0] = byte(index >> 24)
				payload[1] = byte(index >> 16)
				payload[2] = byte(index >> 8)
				payload[3] = byte(index)
				payload[4] = byte(begin >> 24)
				payload[5] = byte(begin >> 16)
				payload[6] = byte(begin >> 8)
				payload[7] = byte(begin)
				copy(payload[8:], block)
				_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgPiece, Payload: payload})
			}
		}
	}()

	conn := peer.NewConnection(client, tor.InfoHash, peer.Handshake{InfoHash: tor.InfoHash, PeerID: testPeerID()})
	if err := conn.PerformHandshake(); err != nil {
		t.Fatalf("PerformHandshake() error = %v", err)
	}

	dir := t.TempDir()
	writer, err := file.NewWriter(dir, tor.Info)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	if err := DownloadPiece(conn, tor, 0, writer); err != nil {
		t.Fatalf("DownloadPiece() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	want := []byte("hello world\n")
	if string(got) != string(want) {
		t.Errorf("file content = %q, want %q", got, want)
	}
}
