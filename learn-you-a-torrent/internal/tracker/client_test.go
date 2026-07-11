package tracker

import (
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"runtime"
	"testing"

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

func TestClient_Announce_mockTracker(t *testing.T) {
	compact := []byte{127, 0, 0, 1, 0x1a, 0xe1}
	responseBody := append([]byte("d5:peers6:"), compact...)
	responseBody = append(responseBody, 'e')

	var gotCompact bool
	var gotInfoHash bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		gotCompact = q.Get("compact") == "1"
		gotInfoHash = q.Get("info_hash") != ""
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseBody)
	}))
	defer srv.Close()

	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	tor.Announce = srv.URL

	var peerID [20]byte
	copy(peerID[:], []byte("-GO0001-000000"))

	client := NewClient(peerID, srv.Client())
	peers, err := client.Announce(tor)
	if err != nil {
		t.Fatalf("Announce() error = %v", err)
	}

	if !gotCompact {
		t.Error("tracker request missing compact=1")
	}
	if !gotInfoHash {
		t.Error("tracker request missing info_hash")
	}

	if len(peers) != 1 {
		t.Fatalf("Announce() peer count = %d, want 1", len(peers))
	}
	if !peers[0].IP.Equal(net.IP{127, 0, 0, 1}) {
		t.Errorf("peer IP = %v, want 127.0.0.1", peers[0].IP)
	}
	if peers[0].Port != 6881 {
		t.Errorf("peer Port = %d, want 6881", peers[0].Port)
	}
}
