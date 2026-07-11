package torrent

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

// Torrent represents parsed .torrent metadata.
type Torrent struct {
	Announce string
	Info     Info
	InfoHash [20]byte
}
