package pieces

import (
	"bytes"
	"crypto/sha1"
	"fmt"
)

const BlockSize = 1 << 14

// Piece holds in-progress piece data and received block tracking.
type Piece struct {
	data     []byte
	received map[int]bool
}

// NewPiece allocates a zeroed buffer for a piece of the given length.
func NewPiece(length int) *Piece {
	return &Piece{
		data:     make([]byte, length),
		received: make(map[int]bool),
	}
}

func (p *Piece) numBlocks() int {
	return (len(p.data) + BlockSize - 1) / BlockSize
}

// WriteBlock copies block data at begin within the piece buffer.
func (p *Piece) WriteBlock(begin int, block []byte) error {
	if begin < 0 || begin >= len(p.data) {
		return fmt.Errorf("pieces: begin %d out of range", begin)
	}
	if begin+len(block) > len(p.data) {
		return fmt.Errorf("pieces: block exceeds piece bounds")
	}
	copy(p.data[begin:], block)
	p.received[begin/BlockSize] = true
	return nil
}

// Complete reports whether all blocks for this piece have been received.
func (p *Piece) Complete() bool {
	for i := 0; i < p.numBlocks(); i++ {
		if !p.received[i] {
			return false
		}
	}
	return p.numBlocks() > 0
}

// Bytes returns the full piece buffer.
func (p *Piece) Bytes() []byte {
	return p.data
}

// Validate checks the piece buffer against the expected SHA1 hash.
func (p *Piece) Validate(expected [20]byte) bool {
	sum := sha1.Sum(p.data)
	return bytes.Equal(sum[:], expected[:])
}

// Reset clears piece data and block tracking for re-download.
func (p *Piece) Reset() {
	for i := range p.data {
		p.data[i] = 0
	}
	p.received = make(map[int]bool)
}
