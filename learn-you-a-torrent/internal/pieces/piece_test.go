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

func minimalPieceHash() [20]byte {
	return [20]byte{
		0x98, 0xaa, 0xed, 0x44, 0x27, 0x21, 0xe0, 0xce,
		0xcd, 0xd9, 0xbf, 0x7b, 0x8c, 0xbb, 0x3e, 0x1f,
		0xf1, 0xb1, 0x53, 0x6a,
	}
}

func fillMinimalPiece(p *Piece) {
	content := []byte("hello world\n")
	copy(p.data, content)
	p.received[0] = true
}

func TestPieceValidate_matchesGoldenHash(t *testing.T) {
	p := NewPiece(16384)
	fillMinimalPiece(p)

	if !p.Validate(minimalPieceHash()) {
		t.Error("Validate() = false, want true for golden minimal piece hash")
	}
}

func TestPieceValidate_rejectsCorruptData(t *testing.T) {
	p := NewPiece(16384)
	fillMinimalPiece(p)
	p.data[0] ^= 0xff

	if p.Validate(minimalPieceHash()) {
		t.Error("Validate() = true, want false for corrupt data")
	}
}

func TestPieceReset_clearsState(t *testing.T) {
	p := NewPiece(16384)
	fillMinimalPiece(p)
	p.data[0] ^= 0xff

	if p.Validate(minimalPieceHash()) {
		t.Fatal("Validate() = true before reset on corrupt data")
	}

	p.Reset()
	if p.Complete() {
		t.Error("Complete() = true after Reset(), want false")
	}
	for i, b := range p.Bytes() {
		if b != 0 {
			t.Fatalf("Bytes()[%d] = %d after Reset(), want 0", i, b)
		}
	}
}
