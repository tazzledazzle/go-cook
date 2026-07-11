package uat

import (
	"os"
	"reflect"
	"testing"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/bencode"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

const goldenInfoHash = "90dc02db9bd6d3808cbfdbba2633f4c6af7180f0"

// UAT 1.1 — Bencode decoder handles all core types

func TestUAT_1_1a_decodeAllCoreTypes(t *testing.T) {
	cases := []struct {
		input string
		want  interface{}
	}{
		{"4:spam", "spam"},
		{"i42e", int64(42)},
		{"li1ei2ei3ee", []interface{}{int64(1), int64(2), int64(3)}},
		{"d3:foo3:bare", map[string]interface{}{"foo": "bar"}},
	}
	for _, c := range cases {
		d := bencode.NewDecoder([]byte(c.input))
		got, err := d.Decode()
		if err != nil {
			t.Fatalf("Decode(%q) error = %v", c.input, err)
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Fatalf("Decode(%q) = %#v, want %#v", c.input, got, c.want)
		}
	}
}

func TestUAT_1_1b_decodeRejectsMalformed(t *testing.T) {
	for _, input := range []string{"", "i42", "5:abc"} {
		d := bencode.NewDecoder([]byte(input))
		if _, err := d.Decode(); err == nil {
			t.Fatalf("Decode(%q) expected error, got nil", input)
		}
	}
}

// UAT 1.2 — Raw info dictionary bytes preserved

func TestUAT_1_2a_infoDictBytesMatchFileSlice(t *testing.T) {
	data, err := os.ReadFile(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	d := bencode.NewDecoder(data)
	if _, err := d.Decode(); err != nil {
		t.Fatalf("Decode(): %v", err)
	}
	const infoOffset = 47
	const infoLength = 83
	got := d.InfoDictBytes()
	want := data[infoOffset : infoOffset+infoLength]
	if string(got) != string(want) {
		t.Fatalf("InfoDictBytes mismatch\ngot:  %q\nwant: %q", got, want)
	}
}

func TestUAT_1_2b_infoDictIsRawDictNotReencoded(t *testing.T) {
	data, err := os.ReadFile(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	d := bencode.NewDecoder(data)
	if _, err := d.Decode(); err != nil {
		t.Fatalf("Decode(): %v", err)
	}
	info := d.InfoDictBytes()
	if info[0] != 'd' || info[len(info)-1] != 'e' {
		t.Fatalf("InfoDictBytes = %q, want d...e wrapper", info)
	}
	if len(info) != 83 {
		t.Fatalf("InfoDictBytes length = %d, want 83", len(info))
	}
}

// UAT 1.3 — Parse minimal.torrent metadata

func TestUAT_1_3a_parseExtractsNameAndPieceLength(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if tor.Info.Name != "test.txt" {
		t.Errorf("Name = %q, want test.txt", tor.Info.Name)
	}
	if tor.Info.PieceLength != 16384 {
		t.Errorf("PieceLength = %d, want 16384", tor.Info.PieceLength)
	}
}

func TestUAT_1_3b_parseExtractsLengthAndPieceCount(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if tor.Info.Length != 12 {
		t.Errorf("Length = %d, want 12", tor.Info.Length)
	}
	if tor.Info.PieceCount() != 1 {
		t.Errorf("PieceCount() = %d, want 1", tor.Info.PieceCount())
	}
}

// UAT 1.4 — Info hash golden match

func TestUAT_1_4a_infoHashMatchesGolden(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if torrent.InfoHashHex(tor.InfoHash) != goldenInfoHash {
		t.Errorf("InfoHash = %s, want %s", torrent.InfoHashHex(tor.InfoHash), goldenInfoHash)
	}
}

func TestUAT_1_4b_infoHashStableAcrossReparse(t *testing.T) {
	path := minimalTorrentPath(t)
	first, err := torrent.ParseTorrent(path)
	if err != nil {
		t.Fatalf("first ParseTorrent() error = %v", err)
	}
	second, err := torrent.ParseTorrent(path)
	if err != nil {
		t.Fatalf("second ParseTorrent() error = %v", err)
	}
	if first.InfoHash != second.InfoHash {
		t.Fatalf("InfoHash changed across parses: %x vs %x", first.InfoHash, second.InfoHash)
	}
}

// UAT 1.5 — Synthetic fixture available

func TestUAT_1_5a_minimalTorrentFileExists(t *testing.T) {
	fileExists(t, minimalTorrentPath(t))
}

func TestUAT_1_5b_minimalTorrentIsNonEmpty(t *testing.T) {
	info, err := os.Stat(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Size() < 100 {
		t.Fatalf("minimal.torrent size = %d, want >= 100 bytes", info.Size())
	}
}

// UAT 1.6 — Phase 1 full test suite green

func TestUAT_1_6a_bencodePackageTestsPass(t *testing.T) {
	runGoTest(t, "./internal/bencode/...")
}

func TestUAT_1_6b_torrentPackageTestsPass(t *testing.T) {
	runGoTest(t, "./internal/torrent/...")
}
