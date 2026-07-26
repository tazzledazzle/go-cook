package file

import (
	"os"
	"path/filepath"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

// Writer creates and writes verified piece data to the torrent output file.
type Writer struct {
	path string
	info torrent.Info
	file *os.File
}

// NewWriter creates the output file named from info.Name in dir.
func NewWriter(dir string, info torrent.Info) (*Writer, error) {
	path := filepath.Join(dir, info.Name)
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &Writer{path: path, info: info, file: f}, nil
}

// WritePiece writes verified piece bytes at index * pieceLength.
func (w *Writer) WritePiece(index int, data []byte) error {
	offset := int64(index) * w.info.PieceLength
	remaining := w.info.Length - offset
	if remaining <= 0 {
		return nil
	}
	writeLen := w.info.PieceLength
	if writeLen > remaining {
		writeLen = remaining
	}
	_, err := w.file.WriteAt(data[:writeLen], offset)
	return err
}

// Close closes the underlying file.
func (w *Writer) Close() error {
	if w.file == nil {
		return nil
	}
	return w.file.Close()
}
