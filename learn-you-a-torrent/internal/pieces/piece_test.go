package pieces

import (
	"bytes"
	"testing"
)

func TestPiece_writeBlockAtBegin(t *testing.T) {
	p := NewPiece(16384)
	want := []byte("hello world\n")
	if err := p.WriteBlock(0, want); err != nil {
		t.Fatalf("WriteBlock() error = %v", err)
	}
	if got := p.Bytes()[0:len(want)]; !bytes.Equal(got, want) {
		t.Errorf("Bytes()[0:12] = %q, want %q", got, want)
	}
	if !p.Complete() {
		t.Error("Complete() = false, want true after single block for 16384-byte piece")
	}
}

func TestPiece_writeBlockAtNonZeroBegin(t *testing.T) {
	p := NewPiece(32768)
	first := bytes.Repeat([]byte("a"), BlockSize)
	second := []byte("tail")

	if err := p.WriteBlock(BlockSize, second); err != nil {
		t.Fatalf("WriteBlock() error = %v", err)
	}
	if got := p.Bytes()[BlockSize : BlockSize+len(second)]; !bytes.Equal(got, second) {
		t.Errorf("second block = %q, want %q", got, second)
	}
	if p.Complete() {
		t.Error("Complete() = true, want false with only second block received")
	}

	if err := p.WriteBlock(0, first); err != nil {
		t.Fatalf("WriteBlock() error = %v", err)
	}
	if !p.Complete() {
		t.Error("Complete() = false, want true after both blocks received")
	}
}

func TestPiece_completeWhenAllBlocksReceived(t *testing.T) {
	p := NewPiece(16384)
	if p.Complete() {
		t.Fatal("Complete() = true on empty piece, want false")
	}

	block := bytes.Repeat([]byte("x"), BlockSize)
	if err := p.WriteBlock(0, block); err != nil {
		t.Fatalf("WriteBlock() error = %v", err)
	}
	if !p.Complete() {
		t.Error("Complete() = false, want true")
	}
}
