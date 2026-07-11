package main

import (
	"strings"
	"testing"
)

func TestRun_missingArgs(t *testing.T) {
	if err := Run(nil); err == nil {
		t.Fatal("Run(nil) expected error, got nil")
	}
	if err := Run([]string{"download"}); err == nil {
		t.Fatal("Run without torrent path expected error, got nil")
	}
}

func TestRun_invalidTorrentPath(t *testing.T) {
	err := Run([]string{"download", "/no/such/file.torrent"})
	if err == nil {
		t.Fatal("Run() expected error for missing file, got nil")
	}
	if !strings.Contains(err.Error(), "read torrent") && !strings.Contains(err.Error(), "no such file") {
		t.Fatalf("error = %v, want read/open failure", err)
	}
}
