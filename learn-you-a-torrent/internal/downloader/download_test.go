package downloader

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

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

func encodeCompactPeer(ip net.IP, port uint16) []byte {
	b := make([]byte, 6)
	copy(b[0:4], ip.To4())
	b[4] = byte(port >> 8)
	b[5] = byte(port)
	return b
}

func startMockPeer(t *testing.T, tor *torrent.Torrent, pieceData []byte) (port uint16, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	peerHS := peer.Handshake{InfoHash: tor.InfoHash, PeerID: testPeerID()}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleMockPeer(conn, peerHS, pieceData)
		}
	}()

	return uint16(addr.Port), func() { _ = ln.Close() }
}

func handleMockPeer(conn net.Conn, peerHS peer.Handshake, pieceData []byte) {
	defer conn.Close()
	buf := make([]byte, 68)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}
	_, _ = conn.Write(peerHS.Serialize())
	_ = peer.WriteMessage(conn, peer.Message{ID: peer.MsgBitfield, Payload: []byte{0xff}})
	_ = peer.WriteMessage(conn, peer.Message{ID: peer.MsgUnchoke})

	for {
		msg, err := peer.ReadMessage(conn)
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
			_ = peer.WriteMessage(conn, peer.Message{ID: peer.MsgPiece, Payload: payload})
		}
	}
}

func TestDownloader_downloadsMinimalTorrent(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}

	pieceData := buildMinimalPieceData()
	port, cleanupPeer := startMockPeer(t, tor, pieceData)
	defer cleanupPeer()

	compact := encodeCompactPeer(net.IP{127, 0, 0, 1}, port)
	responseBody := append([]byte("d5:peers6:"), compact...)
	responseBody = append(responseBody, 'e')

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseBody)
	}))
	defer srv.Close()
	tor.Announce = srv.URL

	dir := t.TempDir()
	d := &Downloader{
		PeerID: testPeerID(),
		ListPeers: func(t *torrent.Torrent) ([]PeerAddress, error) {
			return []PeerAddress{{IP: net.IP{127, 0, 0, 1}, Port: port}}, nil
		},
	}

	if err := d.Download(context.Background(), tor, dir); err != nil {
		t.Fatalf("Download() error = %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "hello world\n" {
		t.Errorf("file content = %q, want %q", got, "hello world\n")
	}
}

func startBlockingMockPeer(t *testing.T, tor *torrent.Torrent) (port uint16, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	peerHS := peer.Handshake{InfoHash: tor.InfoHash, PeerID: testPeerID()}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handleBlockingMockPeer(conn, peerHS)
		}
	}()

	return uint16(addr.Port), func() { _ = ln.Close() }
}

func handleBlockingMockPeer(conn net.Conn, peerHS peer.Handshake) {
	defer conn.Close()
	buf := make([]byte, 68)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return
	}
	_, _ = conn.Write(peerHS.Serialize())
	_ = peer.WriteMessage(conn, peer.Message{ID: peer.MsgBitfield, Payload: []byte{0xff}})
	_ = peer.WriteMessage(conn, peer.Message{ID: peer.MsgUnchoke})

	for {
		if _, err := peer.ReadMessage(conn); err != nil {
			return
		}
	}
}

func TestDownloader_cancelledContext(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}

	port, cleanupPeer := startBlockingMockPeer(t, tor)
	defer cleanupPeer()

	dir := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	d := &Downloader{
		PeerID: testPeerID(),
		ListPeers: func(t *torrent.Torrent) ([]PeerAddress, error) {
			return []PeerAddress{{IP: net.IP{127, 0, 0, 1}, Port: port}}, nil
		},
	}

	err = d.Download(ctx, tor, dir)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Download() error = %v, want context.Canceled", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("file length = %d, want 0 verified bytes on cancel", len(got))
	}
}

func TestDownloader_multiplePeers(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}

	pieceData := buildMinimalPieceData()
	port, cleanupPeer := startMockPeer(t, tor, pieceData)
	defer cleanupPeer()

	dir := t.TempDir()
	var maxPeers atomic.Int32
	d := &Downloader{
		PeerID: testPeerID(),
		ListPeers: func(t *torrent.Torrent) ([]PeerAddress, error) {
			return []PeerAddress{
				{IP: net.IP{127, 0, 0, 1}, Port: port},
				{IP: net.IP{127, 0, 0, 1}, Port: port},
			}, nil
		},
		OnProgress: func(p torrent.Progress) {
			for {
				current := maxPeers.Load()
				if int32(p.ActivePeers) <= current {
					return
				}
				if maxPeers.CompareAndSwap(current, int32(p.ActivePeers)) {
					return
				}
			}
		},
	}

	if err := d.Download(context.Background(), tor, dir); err != nil {
		t.Fatalf("Download() error = %v", err)
	}
	if maxPeers.Load() < 2 {
		t.Fatalf("max active peers = %d, want >= 2", maxPeers.Load())
	}

	got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "hello world\n" {
		t.Errorf("file content = %q, want %q", got, "hello world\n")
	}
}
