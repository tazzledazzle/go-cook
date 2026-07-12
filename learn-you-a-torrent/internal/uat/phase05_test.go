package uat

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tazzledazzle/go-cook/learn-you-a-torrent/internal/torrent"
)

// UAT 5.1 — Cold-start CLI smoke (main.go exists)

func TestUAT_5_1a_torrentCLIBuilds(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", filepath.Join(t.TempDir(), "torrent"), "./cmd/torrent")
	cmd.Dir = moduleRoot(t)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, out)
	}
}

func TestUAT_5_1b_torrentCLIRejectsMissingArgs(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestRun_missingArgs")
}

// UAT 5.2 — Full file download completes

func TestUAT_5_2a_downloaderIntegrationPasses(t *testing.T) {
	runGoTest(t, "./internal/downloader/...", "-run", "TestDownloader_downloadsMinimalTorrent")
}

func TestUAT_5_2b_minimalTorrentIsSinglePiece(t *testing.T) {
	tor, err := torrent.ParseTorrent(minimalTorrentPath(t))
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if tor.Info.PieceCount() != 1 {
		t.Fatalf("PieceCount() = %d, want 1 (full download = one piece)", tor.Info.PieceCount())
	}
}

// UAT 5.3 — Progress line shows live stats

func TestUAT_5_3a_progressStringIncludesPercent(t *testing.T) {
	p := torrent.Progress{
		CompletedPieces: 0,
		TotalPieces:     1,
		ActivePeers:     1,
		Elapsed:         time.Second,
	}
	if !strings.Contains(p.String(), "0.0%") {
		t.Fatalf("Progress = %q, want percent", p.String())
	}
}

func TestUAT_5_3b_progressStringIncludesPeerCount(t *testing.T) {
	p := torrent.Progress{
		CompletedPieces: 1,
		TotalPieces:     1,
		DownloadedBytes: 12,
		TotalBytes:      12,
		ActivePeers:     2,
		Elapsed:         2 * time.Second,
	}
	got := p.String()
	for _, want := range []string{"100.0%", "2 peers"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Progress = %q, want substring %q", got, want)
		}
	}
}

// UAT 5.4 — Output file content verified

func TestUAT_5_4a_minimalFixtureExpectedContentDocumented(t *testing.T) {
	readme, err := os.ReadFile(filepath.Join(moduleRoot(t), "testdata", "README.md"))
	if err != nil {
		t.Fatalf("ReadFile README: %v", err)
	}
	if !strings.Contains(string(readme), "hello world") {
		t.Fatal("testdata README should document hello world content")
	}
}

func TestUAT_5_4b_downloaderOutputMatchesFixture(t *testing.T) {
	runGoTest(t, "./internal/downloader/...", "-run", "TestDownloader_downloadsMinimalTorrent")
}

// UAT 5.5 — Multiple peers connected concurrently

func TestUAT_5_5a_multiplePeersTestPasses(t *testing.T) {
	runGoTest(t, "./internal/downloader/...", "-run", "TestDownloader_multiplePeers")
}

func TestUAT_5_5b_pieceManagerClaimsMissingConcurrently(t *testing.T) {
	runGoTest(t, "./internal/pieces/...", "-run", "TestManager_nextMissingAndMarkComplete")
}

// UAT 5.6 — CLI integration test passes

func TestUAT_5_6a_cliInvalidTorrentPathFails(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestRun_invalidTorrentPath")
}

func TestUAT_5_6b_cliDownloadResultHandlersPass(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestHandleDownloadResult")
}

// UAT 5.7 — Phase 5 full suite green

func TestUAT_5_7a_fullModuleTestsPass(t *testing.T) {
	runGoTestAllExceptUAT(t)
}

func TestUAT_5_7b_fullModuleRaceClean(t *testing.T) {
	runGoTestAllExceptUAT(t, "-race")
}
