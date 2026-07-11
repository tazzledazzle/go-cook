package pieces

import (
	"fmt"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/file"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/peer"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

// DownloadPiece downloads a single piece from conn, validates it, and writes to w.
func DownloadPiece(conn *peer.Connection, tor *torrent.Torrent, index int, w *file.Writer) error {
	unchoked := false
	for !unchoked {
		msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("download piece: wait unchoke: %w", err)
		}
		if msg.ID == peer.MsgUnchoke {
			unchoked = true
		}
	}

	if err := conn.SendInterested(); err != nil {
		return fmt.Errorf("download piece: send interested: %w", err)
	}

	pieceLen := int(tor.Info.PieceLength)
	piece := NewPiece(pieceLen)

	for begin := 0; begin < pieceLen; begin += BlockSize {
		blockLen := BlockSize
		if remain := pieceLen - begin; remain < blockLen {
			blockLen = remain
		}
		if err := conn.SendRequest(uint32(index), uint32(begin), uint32(blockLen)); err != nil {
			return fmt.Errorf("download piece: send request: %w", err)
		}

		if err := readPieceBlock(conn, index, begin, piece); err != nil {
			return err
		}
	}

	hash, err := tor.Info.PieceHash(index)
	if err != nil {
		return fmt.Errorf("download piece: piece hash: %w", err)
	}
	if !piece.Validate(hash) {
		return fmt.Errorf("download piece: hash mismatch for piece %d", index)
	}
	if err := w.WritePiece(index, piece.Bytes()); err != nil {
		return fmt.Errorf("download piece: write file: %w", err)
	}
	return nil
}

func readPieceBlock(conn *peer.Connection, index, begin int, piece *Piece) error {
	for {
		msg, err := conn.ReadMessage()
		if err != nil {
			return fmt.Errorf("download piece: read message: %w", err)
		}
		if msg.ID != peer.MsgPiece {
			continue
		}
		pieceIndex, pieceBegin, block, err := peer.ParsePiece(msg)
		if err != nil {
			return fmt.Errorf("download piece: parse piece: %w", err)
		}
		if pieceIndex != uint32(index) || int(pieceBegin) != begin {
			continue
		}
		if err := piece.WriteBlock(begin, block); err != nil {
			return fmt.Errorf("download piece: write block: %w", err)
		}
		return nil
	}
}
