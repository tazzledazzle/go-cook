package tracker

import (
	"strings"
	"testing"
)

func minimalInfoHash() [20]byte {
	return [20]byte{
		0x90, 0xdc, 0x02, 0xdb, 0x9b, 0xd6, 0xd3, 0x80,
		0x8c, 0xbf, 0xdb, 0xba, 0x26, 0x33, 0xf4, 0xc6,
		0xaf, 0x71, 0x80, 0xf0,
	}
}

func testPeerID() [20]byte {
	var peerID [20]byte
	copy(peerID[:], []byte("-GO0001-000000"))
	return peerID
}

func TestBuildAnnounceURL_minimalFixture(t *testing.T) {
	req := AnnounceRequest{
		InfoHash:   minimalInfoHash(),
		PeerID:     testPeerID(),
		Port:       6881,
		Uploaded:   0,
		Downloaded: 0,
		Left:       12,
	}

	got, err := BuildAnnounceURL("http://example.com/announce", req)
	if err != nil {
		t.Fatalf("BuildAnnounceURL() error = %v", err)
	}

	if !strings.Contains(got, "compact=1") {
		t.Errorf("URL missing compact=1: %s", got)
	}
	if !strings.Contains(got, "left=12") {
		t.Errorf("URL missing left=12: %s", got)
	}
	if !strings.Contains(got, "port=6881") {
		t.Errorf("URL missing port=6881: %s", got)
	}
	// Go QueryEscape uses uppercase hex for binary info_hash bytes
	if !strings.Contains(got, "info_hash=%90%DC%02%DB") {
		t.Errorf("URL missing percent-encoded info_hash prefix: %s", got)
	}
}
