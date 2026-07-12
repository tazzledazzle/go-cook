package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

func minimalInfo() torrent.Info {
	return torrent.Info{
		Name:        "test.txt",
		Length:      12,
		PieceLength: 16384,
	}
}

func TestWriter_createsNamedFile(t *testing.T) {
	dir := t.TempDir()
	info := minimalInfo()

	w, err := NewWriter(dir, info)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	t.Cleanup(func() { _ = w.Close() })

	path := filepath.Join(dir, "test.txt")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Stat(%q) error = %v", path, err)
	}
}

func TestWriter_writePieceAtOffsetZero(t *testing.T) {
	dir := t.TempDir()
	info := minimalInfo()

	w, err := NewWriter(dir, info)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}

	pieceData := make([]byte, info.PieceLength)
	copy(pieceData, []byte("hello world\n"))
	if err := w.WritePiece(0, pieceData); err != nil {
		t.Fatalf("WritePiece() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
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
