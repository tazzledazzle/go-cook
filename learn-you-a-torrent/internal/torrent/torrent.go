package torrent

import "fmt"

// Info holds metadata from the torrent info dictionary (single-file).
type Info struct {
	Name        string
	PieceLength int64
	Length      int64
	Pieces      []byte
}

// PieceCount returns the number of piece hashes in Pieces.
func (i Info) PieceCount() int {
	return len(i.Pieces) / 20
}

// PieceHash returns the SHA1 hash for the piece at index.
func (i Info) PieceHash(index int) ([20]byte, error) {
	start := index * 20
	if start+20 > len(i.Pieces) {
		return [20]byte{}, fmt.Errorf("torrent: piece index %d out of range", index)
	}
	var hash [20]byte
	copy(hash[:], i.Pieces[start:start+20])
	return hash, nil
}

// Torrent represents parsed .torrent metadata.
type Torrent struct {
	Announce string
	Info     Info
	InfoHash [20]byte
}
