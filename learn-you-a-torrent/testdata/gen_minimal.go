//go:build ignore

package main

import (
	"crypto/sha1"
	"fmt"
	"os"
)

func encodeString(s string) []byte {
	return append([]byte(fmt.Sprintf("%d:", len(s))), s...)
}

func encodeInt(n int64) []byte {
	return []byte(fmt.Sprintf("i%de", n))
}

func encodeDict(pairs ...[]byte) []byte {
	out := []byte("d")
	for _, p := range pairs {
		out = append(out, p...)
	}
	out = append(out, 'e')
	return out
}

func main() {
	fileContent := []byte("hello world\n") // 12 bytes
	pieceLength := int64(16384)
	piece := make([]byte, pieceLength)
	copy(piece, fileContent)
	pieceHash := sha1.Sum(piece)

	infoDict := encodeDict(
		append(encodeString("length"), encodeInt(int64(len(fileContent)))...),
		append(encodeString("name"), encodeString("test.txt")...),
		append(encodeString("piece length"), encodeInt(pieceLength)...),
		append(encodeString("pieces"), append([]byte("20:"), pieceHash[:]...)...),
	)

	announcePair := append(encodeString("announce"), encodeString("http://example.com/announce")...)
	infoKey := encodeString("info")
	infoStart := len(encodeDict(append(append([]byte{}, announcePair...), append(infoKey, infoDict...)...))) - len(infoDict)

	torrent := encodeDict(
		append(announcePair, append(infoKey, infoDict...)...),
	)

	// Recompute infoStart by finding info dict in torrent bytes
	infoMarker := append(infoKey, 'd')
	for i := 0; i < len(torrent)-len(infoMarker); i++ {
		if string(torrent[i:i+len(infoMarker)]) == string(infoMarker) {
			infoStart = i + len(infoKey)
			break
		}
	}

	infoHash := sha1.Sum(infoDict)

	outPath := "testdata/minimal.torrent"
	if err := os.WriteFile(outPath, torrent, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Wrote %s (%d bytes)\n", outPath, len(torrent))
	fmt.Printf("Info dict offset: %d\n", infoStart)
	fmt.Printf("Info dict length: %d bytes\n", len(infoDict))
	fmt.Printf("Expected info_hash: %x\n", infoHash)
}
