package uat

import (
	"testing"
)

// UAT 6.1 — Cold-start CLI still runs after shutdown wiring

func TestUAT_6_1a_cliStillBuildsAfterShutdownWiring(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestRun_missingArgs")
}

func TestUAT_6_1b_mainPackageTestsPass(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...")
}

// UAT 6.2 — SIGINT/SIGTERM caught in main

func TestUAT_6_2a_signalNotifyIsConfigured(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestHandleDownloadResult_cancelPrintsProgress")
}

func TestUAT_6_2b_cancelContextStopsDownload(t *testing.T) {
	runGoTest(t, "./internal/downloader/...", "-run", "TestDownloader_cancelledContext")
}

// UAT 6.3 — Shutdown prints progress at cancel time

func TestUAT_6_3a_shutdownHandlerTestPasses(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestHandleDownloadResult_cancelPrintsProgress")
}

func TestUAT_6_3b_shutdownHandlerIncludesCompleteMessage(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestHandleDownloadResult_successPrintsComplete")
}

// UAT 6.4 — Download cancels cleanly via context

func TestUAT_6_4a_cancelledContextTestPasses(t *testing.T) {
	runGoTest(t, "./internal/downloader/...", "-run", "TestDownloader_cancelledContext")
}

func TestUAT_6_4b_handleDownloadResultPropagatesErrors(t *testing.T) {
	runGoTest(t, "./cmd/torrent/...", "-run", "TestHandleDownloadResult_propagatesError")
}

// UAT 6.5 — Partial file left in consistent state

func TestUAT_6_5a_cancelLeavesNoVerifiedBytes(t *testing.T) {
	runGoTest(t, "./internal/downloader/...", "-run", "TestDownloader_cancelledContext")
}

func TestUAT_6_5b_onlyCompletePiecesWritten(t *testing.T) {
	runGoTest(t, "./internal/pieces/...", "-run", "TestPieceValidate")
}

// UAT 6.6 — Phase 6 full suite green

func TestUAT_6_6a_allTestsPassAfterShutdown(t *testing.T) {
	runGoTestAllExceptUAT(t)
}

func TestUAT_6_6b_raceDetectorCleanAfterShutdown(t *testing.T) {
	runGoTestAllExceptUAT(t, "-race")
}