package torrent

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

const goldenInfoHash = "90dc02db9bd6d3808cbfdbba2633f4c6af7180f0"

func minimalTorrentPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "minimal.torrent")
}

func TestParseMinimalTorrent(t *testing.T) {
	path := minimalTorrentPath(t)
	tor, err := ParseTorrent(path)
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if tor.Info.Name != "test.txt" {
		t.Errorf("Name = %q, want test.txt", tor.Info.Name)
	}
	if tor.Info.PieceLength != 16384 {
		t.Errorf("PieceLength = %d, want 16384", tor.Info.PieceLength)
	}
	if tor.Info.Length != 12 {
		t.Errorf("Length = %d, want 12", tor.Info.Length)
	}
	if tor.Info.PieceCount() != 1 {
		t.Errorf("PieceCount() = %d, want 1", tor.Info.PieceCount())
	}
	if !strings.Contains(tor.Announce, "example.com") {
		t.Errorf("Announce = %q, want example.com", tor.Announce)
	}
}

func TestInfoHashGolden(t *testing.T) {
	path := minimalTorrentPath(t)
	tor, err := ParseTorrent(path)
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	got := InfoHashHex(tor.InfoHash)
	if got != goldenInfoHash {
		t.Errorf("InfoHash = %s, want %s", got, goldenInfoHash)
	}
}

func TestInfoHash_notReencoded(t *testing.T) {
	// Re-encoding the info map would change key order and break the hash.
	// InfoHashFromBytes must use raw bytes from the file, not a reconstructed dict.
	path := minimalTorrentPath(t)
	tor, err := ParseTorrent(path)
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if InfoHashHex(tor.InfoHash) != goldenInfoHash {
		t.Fatal("golden hash must come from raw info dict bytes in file")
	}
}
