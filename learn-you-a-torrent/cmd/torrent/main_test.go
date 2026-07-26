package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
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

func TestHandleDownloadResult_cancelPrintsProgress(t *testing.T) {
	var buf bytes.Buffer
	last := torrent.Progress{
		CompletedPieces: 0,
		TotalPieces:     1,
		ActivePeers:     1,
		Elapsed:         time.Second,
	}

	err := handleDownloadResult(context.Canceled, last, &buf, "test.txt", "/tmp")
	if err != nil {
		t.Fatalf("handleDownloadResult() error = %v, want nil", err)
	}
	got := buf.String()
	if !strings.Contains(got, "Shutdown:") {
		t.Fatalf("output = %q, want Shutdown prefix", got)
	}
	if !strings.Contains(got, "0.0%") {
		t.Fatalf("output = %q, want progress percent", got)
	}
}

func TestHandleDownloadResult_successPrintsComplete(t *testing.T) {
	var buf bytes.Buffer
	err := handleDownloadResult(nil, torrent.Progress{}, &buf, "test.txt", "/tmp")
	if err != nil {
		t.Fatalf("handleDownloadResult() error = %v", err)
	}
	if !strings.Contains(buf.String(), "Downloaded test.txt") {
		t.Fatalf("output = %q, want download complete message", buf.String())
	}
}

func TestHandleDownloadResult_propagatesError(t *testing.T) {
	err := handleDownloadResult(errors.New("boom"), torrent.Progress{}, io.Discard, "x", "/tmp")
	if err == nil || err.Error() != "boom" {
		t.Fatalf("handleDownloadResult() error = %v, want boom", err)
	}
}
