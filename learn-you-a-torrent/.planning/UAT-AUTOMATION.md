# UAT Automation Registry

Maps every `/gsd-verify-work` checkpoint to **two automated Go tests** in `internal/uat/`. These replace conversational UAT with CI-runnable assertions for future milestone verification.

**Run all UAT automation:**

```bash
go test ./internal/uat/... -v
```

**Run one phase:**

```bash
go test ./internal/uat/... -run 'TestUAT_3_' -v
```

**Optional live tracker check** (UAT 2.4b — requires a real `.torrent`, not the minimal fixture):

```bash
LIVE_TRACKER=1 LIVE_TORRENT_PATH=/path/to/public.torrent \
  go test ./internal/uat/... -run TestUAT_2_4b -v
```

`testdata/minimal.torrent` uses `http://example.com/announce` (placeholder). Override announce with `LIVE_TRACKER_URL` if needed.

---

## How this maps to verify-work

| verify-work step | Automated replacement |
|------------------|----------------------|
| Extract tests from `*-SUMMARY.md` | Rows below (source column) |
| Present checkpoint to user | `TestUAT_{phase}_{item}{a\|b}_*` |
| User types "yes" | Test passes in CI |
| User reports issue | Test fails → gap in UAT run |
| `{phase}-UAT.md` tracking | This file + test names |

---

## Phase 1 — Bencode & Torrent Parsing

| UAT ID | Checkpoint | Test A | Test B | Source |
|--------|------------|--------|--------|--------|
| 1.1 | Bencode decoder handles all core types | `TestUAT_1_1a_decodeAllCoreTypes` | `TestUAT_1_1b_decodeRejectsMalformed` | 01-02-SUMMARY, ROADMAP |
| 1.2 | Raw info dictionary bytes preserved | `TestUAT_1_2a_infoDictBytesMatchFileSlice` | `TestUAT_1_2b_infoDictIsRawDictNotReencoded` | 01-02-SUMMARY, ROADMAP |
| 1.3 | Parse minimal.torrent metadata | `TestUAT_1_3a_parseExtractsNameAndPieceLength` | `TestUAT_1_3b_parseExtractsLengthAndPieceCount` | 01-03-SUMMARY, ROADMAP |
| 1.4 | Info hash golden match | `TestUAT_1_4a_infoHashMatchesGolden` | `TestUAT_1_4b_infoHashStableAcrossReparse` | 01-03-SUMMARY, ROADMAP |
| 1.5 | Synthetic fixture available | `TestUAT_1_5a_minimalTorrentFileExists` | `TestUAT_1_5b_minimalTorrentIsNonEmpty` | 01-01-SUMMARY |
| 1.6 | Phase 1 full test suite green | `TestUAT_1_6a_bencodePackageTestsPass` | `TestUAT_1_6b_torrentPackageTestsPass` | 01-03-SUMMARY |

---

## Phase 2 — Tracker Announce

| UAT ID | Checkpoint | Test A | Test B | Source |
|--------|------------|--------|--------|--------|
| 2.1 | Announce URL encodes binary info_hash | `TestUAT_2_1a_announceURLPercentEncodesInfoHash` | `TestUAT_2_1b_announceURLIncludesRequiredParams` | 02-01-SUMMARY, ROADMAP |
| 2.2 | Compact peers parsed to IP:port list | `TestUAT_2_2a_parseSingleCompactPeer` | `TestUAT_2_2b_parseRejectsMalformedCompact` | 02-02-SUMMARY, ROADMAP |
| 2.3 | HTTP tracker announce returns peer list | `TestUAT_2_3a_clientAnnounceFromMockTracker` | `TestUAT_2_3b_clientAnnounceUsesGET` | 02-03-SUMMARY, ROADMAP |
| 2.4 | Live tracker peer discovery (optional) | `TestUAT_2_4a_fixtureAnnounceURLIsHTTP` | `TestUAT_2_4b_liveTrackerOptional` | ROADMAP |

---

## Phase 3 — Peer Handshake & Messages

| UAT ID | Checkpoint | Test A | Test B | Source |
|--------|------------|--------|--------|--------|
| 3.1 | Handshake wire format roundtrip | `TestUAT_3_1a_handshakeSerializeIs68Bytes` | `TestUAT_3_1b_handshakeRejectsShortBuffer` | 03-01-SUMMARY, ROADMAP |
| 3.2 | Successful handshake with matching info hash | `TestUAT_3_2a_handshakeExchangeSucceeds` | `TestUAT_3_2b_handshakeRoundtripPreservesFields` | 03-02-SUMMARY, ROADMAP |
| 3.3 | Mismatched info hash rejected | `TestUAT_3_3a_handshakeRejectsWrongHash` | `TestUAT_3_3b_handshakeMismatchClosesConnection` | 03-02-SUMMARY, ROADMAP |
| 3.4 | Wire message framing and parsing | `TestUAT_3_4a_messageRoundtripCoreIDs` | `TestUAT_3_4b_readMessageErrorsOnTruncation` | 03-03-SUMMARY, ROADMAP |
| 3.5 | Connection reads bitfield and unchoke | `TestUAT_3_5a_connectionReadsBitfieldThenUnchoke` | `TestUAT_3_5b_connectionSendInterested` | 03-04-SUMMARY, ROADMAP |
| 3.6 | Phase 3 race-safe suite | `TestUAT_3_6a_peerPackageTestsPass` | `TestUAT_3_6b_peerPackageRaceClean` | 03-04-SUMMARY |

---

## Phase 4 — Download One Piece

| UAT ID | Checkpoint | Test A | Test B | Source |
|--------|------------|--------|--------|--------|
| 4.1 | 16 KB block assembly for piece 0 | `TestUAT_4_1a_pieceCompletesAfterFullBlock` | `TestUAT_4_1b_pieceIncompleteUntilAllBlocks` | 04-01-SUMMARY, ROADMAP |
| 4.2 | SHA1 piece validation | `TestUAT_4_2a_validatedPieceMatchesGoldenHash` | `TestUAT_4_2b_validateRequiresCompletePiece` | 04-02-SUMMARY, ROADMAP |
| 4.3 | Hash mismatch triggers piece reset | `TestUAT_4_3a_corruptDataFailsValidation` | `TestUAT_4_3b_resetAllowsRedownload` | 04-02-SUMMARY, ROADMAP |
| 4.4 | Verified piece written to disk at offset 0 | `TestUAT_4_4a_writerCreatesNamedFile` | `TestUAT_4_4b_writerWritesPieceAtOffsetZero` | 04-03-SUMMARY, ROADMAP |
| 4.5 | End-to-end DownloadPiece0 integration | `TestUAT_4_5a_downloadPieceIntegrationPasses` | `TestUAT_4_5b_downloadPieceProducesExpectedBytes` | 04-05-SUMMARY, ROADMAP |

---

## Phase 5 — Full Download & Progress CLI

| UAT ID | Checkpoint | Test A | Test B | Source |
|--------|------------|--------|--------|--------|
| 5.1 | Cold-start CLI smoke | `TestUAT_5_1a_torrentCLIBuilds` | `TestUAT_5_1b_torrentCLIRejectsMissingArgs` | 05-05-SUMMARY, ROADMAP |
| 5.2 | Full file download completes | `TestUAT_5_2a_downloaderIntegrationPasses` | `TestUAT_5_2b_minimalTorrentIsSinglePiece` | 05-03-SUMMARY, ROADMAP |
| 5.3 | Progress line shows live stats | `TestUAT_5_3a_progressStringIncludesPercent` | `TestUAT_5_3b_progressStringIncludesPeerCount` | 05-01-SUMMARY, ROADMAP |
| 5.4 | Output file content verified | `TestUAT_5_4a_minimalFixtureExpectedContentDocumented` | `TestUAT_5_4b_downloaderOutputMatchesFixture` | 05-05-SUMMARY, TEST-03 |
| 5.5 | Multiple peers connected concurrently | `TestUAT_5_5a_multiplePeersTestPasses` | `TestUAT_5_5b_pieceManagerClaimsMissingConcurrently` | 05-04-SUMMARY, ROADMAP |
| 5.6 | CLI integration test passes | `TestUAT_5_6a_cliInvalidTorrentPathFails` | `TestUAT_5_6b_cliDownloadResultHandlersPass` | 05-05-SUMMARY |
| 5.7 | Phase 5 full suite green | `TestUAT_5_7a_fullModuleTestsPass` | `TestUAT_5_7b_fullModuleRaceClean` | 05-05-SUMMARY |

---

## Phase 6 — Graceful Shutdown

| UAT ID | Checkpoint | Test A | Test B | Source |
|--------|------------|--------|--------|--------|
| 6.1 | Cold-start CLI still runs after shutdown wiring | `TestUAT_6_1a_cliStillBuildsAfterShutdownWiring` | `TestUAT_6_1b_mainPackageTestsPass` | 06-02-SUMMARY, ROADMAP |
| 6.2 | SIGINT/SIGTERM caught in main | `TestUAT_6_2a_signalNotifyIsConfigured` | `TestUAT_6_2b_cancelContextStopsDownload` | 06-02-SUMMARY, ROADMAP |
| 6.3 | Shutdown prints progress at cancel time | `TestUAT_6_3a_shutdownHandlerTestPasses` | `TestUAT_6_3b_shutdownHandlerIncludesCompleteMessage` | 06-02-SUMMARY, ROADMAP |
| 6.4 | Download cancels cleanly via context | `TestUAT_6_4a_cancelledContextTestPasses` | `TestUAT_6_4b_handleDownloadResultPropagatesErrors` | 06-01-SUMMARY, ROADMAP |
| 6.5 | Partial file left in consistent state | `TestUAT_6_5a_cancelLeavesNoVerifiedBytes` | `TestUAT_6_5b_onlyCompletePiecesWritten` | 06-01-SUMMARY, ROADMAP |
| 6.6 | Phase 6 full suite green | `TestUAT_6_6a_allTestsPassAfterShutdown` | `TestUAT_6_6b_raceDetectorCleanAfterShutdown` | 06-02-SUMMARY |

---

## Summary

| Phase | UAT checkpoints | Automated tests |
|-------|-----------------|-----------------|
| 1 | 6 | 12 |
| 2 | 4 | 8 |
| 3 | 6 | 12 |
| 4 | 5 | 10 |
| 5 | 7 | 14 |
| 6 | 6 | 12 |
| **Total** | **34** | **68** |

---

## Future verify-work integration

When running `/gsd-verify-work {phase}` on this project:

1. Run `go test ./internal/uat/... -run 'TestUAT_{phase}_'`
2. If all pass → auto-mark UAT checkpoints as pass in `{phase}-UAT.md`
3. If any fail → feed failing test name + output into gap diagnosis

Manual checkpoints that remain human-only:

- **2.4b** live tracker — set `LIVE_TRACKER=1` and `LIVE_TORRENT_PATH` to a legitimate public single-file torrent (minimal fixture uses `example.com`)
- **5.1a** cold-start CLI against real tracker + peers (`go run ./cmd/torrent download …`)

*Created: 2026-07-10 — maps verify-work SUMMARY extraction to `internal/uat/` tests.*
