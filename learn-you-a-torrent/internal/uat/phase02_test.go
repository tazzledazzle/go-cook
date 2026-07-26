package uat

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/tracker"
)

func minimalInfoHash(t *testing.T) [20]byte {
	t.Helper()
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	return tor.InfoHash
}

func testPeerID() [20]byte {
	var id [20]byte
	copy(id[:], []byte("-GO0001-000000"))
	return id
}

// UAT 2.1 — Announce URL encodes binary info_hash

func TestUAT_2_1a_announceURLPercentEncodesInfoHash(t *testing.T) {
	req := tracker.AnnounceRequest{
		InfoHash: minimalInfoHash(t),
		PeerID:   testPeerID(),
		Port:     6881,
		Left:     12,
	}
	got, err := tracker.BuildAnnounceURL("http://example.com/announce", req)
	if err != nil {
		t.Fatalf("BuildAnnounceURL() error = %v", err)
	}
	if !strings.Contains(got, "info_hash=%90%DC%02%DB") {
		t.Errorf("URL missing encoded info_hash prefix: %s", got)
	}
}

func TestUAT_2_1b_announceURLIncludesRequiredParams(t *testing.T) {
	req := tracker.AnnounceRequest{
		InfoHash: minimalInfoHash(t),
		PeerID:   testPeerID(),
		Port:     6881,
		Left:     12,
	}
	got, err := tracker.BuildAnnounceURL("http://example.com/announce", req)
	if err != nil {
		t.Fatalf("BuildAnnounceURL() error = %v", err)
	}
	for _, want := range []string{"compact=1", "left=12", "port=6881", "peer_id="} {
		if !strings.Contains(got, want) {
			t.Errorf("URL missing %q: %s", want, got)
		}
	}
}

// UAT 2.2 — Compact peers parsed to IP:port list

func TestUAT_2_2a_parseSingleCompactPeer(t *testing.T) {
	peers, err := tracker.ParseCompactPeers([]byte{127, 0, 0, 1, 0x1a, 0xe1})
	if err != nil {
		t.Fatalf("ParseCompactPeers() error = %v", err)
	}
	if len(peers) != 1 || peers[0].Port != 6881 {
		t.Fatalf("peers = %#v, want one peer on 6881", peers)
	}
}

func TestUAT_2_2b_parseRejectsMalformedCompact(t *testing.T) {
	if _, err := tracker.ParseCompactPeers([]byte{127, 0, 0, 1}); err == nil {
		t.Fatal("ParseCompactPeers() expected error for short input, got nil")
	}
}

// UAT 2.3 — HTTP tracker announce returns peer list

func TestUAT_2_3a_clientAnnounceFromMockTracker(t *testing.T) {
	compact := []byte{127, 0, 0, 1, 0x1a, 0xe1}
	body := append([]byte("d5:peers6:"), compact...)
	body = append(body, 'e')

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	tor.Announce = srv.URL

	client := tracker.NewClient(testPeerID(), nil)
	peers, err := client.Announce(tor)
	if err != nil {
		t.Fatalf("Announce() error = %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("len(peers) = %d, want 1", len(peers))
	}
}

func TestUAT_2_3b_clientAnnounceUsesGET(t *testing.T) {
	var method string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_, _ = w.Write([]byte("d5:peers0:e"))
	}))
	defer srv.Close()

	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	tor.Announce = srv.URL

	client := tracker.NewClient(testPeerID(), nil)
	if _, err := client.Announce(tor); err != nil {
		t.Fatalf("Announce() error = %v", err)
	}
	if method != http.MethodGet {
		t.Errorf("request method = %q, want GET", method)
	}
}

// UAT 2.4 — Live tracker peer discovery (optional manual)

func TestUAT_2_4a_fixtureAnnounceURLIsHTTP(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if !strings.HasPrefix(tor.Announce, "http://") {
		t.Errorf("Announce = %q, want http:// prefix for v1 tracker", tor.Announce)
	}
}

func TestUAT_2_4b_liveTrackerOptional(t *testing.T) {
	if os.Getenv("LIVE_TRACKER") != "1" {
		t.Skip("set LIVE_TRACKER=1 to run live tracker announce check")
	}

	torrentPath := os.Getenv("LIVE_TORRENT_PATH")
	if torrentPath == "" {
		t.Skip("set LIVE_TORRENT_PATH to a .torrent with a real announce URL (testdata/minimal.torrent uses placeholder http://example.com/announce)")
	}

	tor, err := torrent.ParseTorrent(torrentPath)
	if err != nil {
		t.Fatalf("ParseTorrent(%q) error = %v", torrentPath, err)
	}
	if override := os.Getenv("LIVE_TRACKER_URL"); override != "" {
		tor.Announce = override
	}
	if strings.Contains(tor.Announce, "example.com") {
		t.Skip("announce URL is still example.com placeholder; use a real .torrent or set LIVE_TRACKER_URL")
	}

	client := tracker.NewClient(testPeerID(), nil)
	peers, err := client.Announce(tor)
	if err != nil {
		t.Fatalf("Announce() error = %v", err)
	}
	if len(peers) == 0 {
		t.Fatal("live announce returned zero peers (swarm may be dead or torrent inactive)")
	}
	for _, p := range peers {
		if p.IP == nil || p.Port == 0 {
			t.Fatalf("invalid peer: %#v", p)
		}
	}
}
