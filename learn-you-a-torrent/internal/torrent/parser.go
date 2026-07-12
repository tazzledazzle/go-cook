package torrent

import (
	"fmt"
	"os"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/bencode"
)

// ParseTorrent reads and parses a .torrent file from path.
func ParseTorrent(path string) (*Torrent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read torrent: %w", err)
	}
	return ParseBytes(data)
}

// Open is an alias for ParseTorrent.
func Open(path string) (*Torrent, error) {
	return ParseTorrent(path)
}

// ParseBytes parses .torrent metadata from raw bytes.
func ParseBytes(data []byte) (*Torrent, error) {
	dec := bencode.NewDecoder(data)
	root, err := dec.Decode()
	if err != nil {
		return nil, fmt.Errorf("decode torrent: %w", err)
	}
	top, ok := root.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("torrent root is not a dictionary")
	}

	t := &Torrent{}
	if announce, ok := top["announce"].(string); ok {
		t.Announce = announce
	}

	infoRaw := dec.InfoDictBytes()
	if infoRaw == nil {
		return nil, fmt.Errorf("info dictionary not found")
	}
	t.InfoHash = InfoHashFromBytes(infoRaw)

	infoMap, ok := top["info"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("info is not a dictionary")
	}
	if err := parseInfo(infoMap, &t.Info); err != nil {
		return nil, err
	}
	return t, nil
}

func parseInfo(m map[string]interface{}, info *Info) error {
	if name, ok := m["name"].(string); ok {
		info.Name = name
	}
	if length, ok := m["length"].(int64); ok {
		info.Length = length
	}
	if pl, ok := m["piece length"].(int64); ok {
		info.PieceLength = pl
	}
	if pieces, ok := m["pieces"].(string); ok {
		info.Pieces = []byte(pieces)
	}
	return nil
}
