package uat

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/file"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/peer"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/pieces"
	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

func minimalPieceHash() [20]byte {
	return [20]byte{
		0x98, 0xaa, 0xed, 0x44, 0x27, 0x21, 0xe0, 0xce,
		0xcd, 0xd9, 0xbf, 0x7b, 0x8c, 0xbb, 0x3e, 0x1f,
		0xf1, 0xb1, 0x53, 0x6a,
	}
}

func buildMinimalPieceData() []byte {
	data := make([]byte, 16384)
	copy(data, []byte("hello world\n"))
	return data
}

// UAT 4.1 — 16 KB block assembly for piece 0

func TestUAT_4_1a_pieceCompletesAfterFullBlock(t *testing.T) {
	p := pieces.NewPiece(16384)
	block := bytes.Repeat([]byte("x"), pieces.BlockSize)
	if err := p.WriteBlock(0, block); err != nil {
		t.Fatalf("WriteBlock() error = %v", err)
	}
	if !p.Complete() {
		t.Fatal("Complete() = false, want true after 16KB block")
	}
}

func TestUAT_4_1b_pieceIncompleteUntilAllBlocks(t *testing.T) {
	p := pieces.NewPiece(32768)
	second := []byte("tail")
	if err := p.WriteBlock(pieces.BlockSize, second); err != nil {
		t.Fatalf("WriteBlock() error = %v", err)
	}
	if p.Complete() {
		t.Fatal("Complete() = true, want false with only second block")
	}
}

// UAT 4.2 — SHA1 piece validation

func TestUAT_4_2a_validatedPieceMatchesGoldenHash(t *testing.T) {
	p := pieces.NewPiece(16384)
	content := []byte("hello world\n")
	if err := p.WriteBlock(0, content); err != nil {
		t.Fatalf("WriteBlock() error = %v", err)
	}
	if !p.Validate(minimalPieceHash()) {
		t.Fatal("Validate() = false, want true for minimal fixture piece 0")
	}
}

func TestUAT_4_2b_validateRequiresCompletePiece(t *testing.T) {
	p := pieces.NewPiece(16384)
	if p.Validate(minimalPieceHash()) {
		t.Fatal("Validate() = true on empty piece, want false")
	}
}

// UAT 4.3 — Hash mismatch triggers piece reset

func TestUAT_4_3a_corruptDataFailsValidation(t *testing.T) {
	p := pieces.NewPiece(16384)
	content := []byte("hello world\n")
	_ = p.WriteBlock(0, content)
	p.Bytes()[0] ^= 0xff
	if p.Validate(minimalPieceHash()) {
		t.Fatal("Validate() = true on corrupt data, want false")
	}
}

func TestUAT_4_3b_resetAllowsRedownload(t *testing.T) {
	p := pieces.NewPiece(16384)
	content := []byte("hello world\n")
	_ = p.WriteBlock(0, content)
	p.Reset()
	if err := p.WriteBlock(0, content); err != nil {
		t.Fatalf("WriteBlock() after reset error = %v", err)
	}
	if !p.Validate(minimalPieceHash()) {
		t.Fatal("Validate() = false after redownload, want true")
	}
}

// UAT 4.4 — Verified piece written to disk at correct offset

func TestUAT_4_4a_writerCreatesNamedFile(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	dir := t.TempDir()
	w, err := file.NewWriter(dir, tor.Info)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	fileExists(t, filepath.Join(dir, "test.txt"))
}

func TestUAT_4_4b_writerWritesPieceAtOffsetZero(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	dir := t.TempDir()
	w, err := file.NewWriter(dir, tor.Info)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	pieceData := make([]byte, tor.Info.PieceLength)
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
	if string(got) != "hello world\n" {
		t.Errorf("file = %q, want hello world\\n", got)
	}
}

// UAT 4.5 — End-to-end DownloadPiece0 integration

func TestUAT_4_5a_downloadPieceIntegrationPasses(t *testing.T) {
	runGoTest(t, "./internal/pieces/...", "-run", "TestDownloadPiece0_fromMockPeer")
}

func TestUAT_4_5b_downloadPieceProducesExpectedBytes(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}

	pieceData := buildMinimalPieceData()
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})

	hash := tor.InfoHash
	peerHS := peer.Handshake{InfoHash: hash, PeerID: testPeerID()}

	go func() {
		buf := make([]byte, 68)
		if _, err := io.ReadFull(server, buf); err != nil {
			return
		}
		_, _ = server.Write(peerHS.Serialize())
		_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgBitfield, Payload: []byte{0xff}})
		_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgUnchoke})
		for {
			msg, err := peer.ReadMessage(server)
			if err != nil {
				return
			}
			if msg.ID != peer.MsgRequest {
				continue
			}
			index, begin, length, err := peer.ParseRequest(msg)
			if err != nil {
				return
			}
			block := pieceData[begin : begin+length]
			payload := make([]byte, 8+len(block))
			payload[0] = byte(index >> 24)
			payload[1] = byte(index >> 16)
			payload[2] = byte(index >> 8)
			payload[3] = byte(index)
			payload[4] = byte(begin >> 24)
			payload[5] = byte(begin >> 16)
			payload[6] = byte(begin >> 8)
			payload[7] = byte(begin)
			copy(payload[8:], block)
			_ = peer.WriteMessage(server, peer.Message{ID: peer.MsgPiece, Payload: payload})
		}
	}()

	conn := peer.NewConnection(client, hash, peerHS)
	if err := conn.PerformHandshake(); err != nil {
		t.Fatalf("PerformHandshake() error = %v", err)
	}
	dir := t.TempDir()
	writer, err := file.NewWriter(dir, tor.Info)
	if err != nil {
		t.Fatalf("NewWriter() error = %v", err)
	}
	if err := pieces.DownloadPiece(conn, tor, 0, writer); err != nil {
		t.Fatalf("DownloadPiece() error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(got) != "hello world\n" {
		t.Errorf("file = %q, want hello world\\n", got)
	}
}
