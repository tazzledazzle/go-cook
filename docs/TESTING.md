<!-- generated-by: gsd-doc-writer -->
# Testing

## Test Framework

All tests use the Go standard `testing` package. `learn-you-a-torrent` also uses `github.com/stretchr/testify/assert` for cleaner assertions.

## Running Tests

### All tests in a sub-project

```bash
cd learn-you-a-torrent && go test ./...
cd design-kube        && go test ./...
cd coding-problems/arrays-and-strings && go test ./...
```

### Root package

```bash
# From repo root (no go.mod — tests conv_test.go only)
go test .
```

### With race detector

```bash
cd learn-you-a-torrent && go test -race ./...
```

Run with `-race` whenever you modify peer goroutine code in `learn-you-a-torrent`.

### Verbose output

```bash
go test -v ./...
```

## Test Structure

### learn-you-a-torrent

Tests follow a TDD-first discipline: `_test.go` files are written before implementation files.

| Package | Test file | What it covers |
|---------|-----------|----------------|
| `internal/bencode` | `decode_test.go` | Bencode decode roundtrips |
| `internal/torrent` | `parser_test.go`, `progress_test.go` | `.torrent` parsing, download progress |
| `internal/tracker` | `announce_test.go`, `client_test.go`, `peers_test.go` | Tracker HTTP announce, peer list decode |
| `internal/peer` | `handshake_test.go`, `connection_test.go`, `message_test.go` | Peer wire protocol handshake and messages |
| `internal/pieces` | `piece_test.go`, `downloader_test.go`, `manager_test.go` | Piece SHA1 verification, download scheduling |
| `internal/file` | `writer_test.go` | Disk write correctness |
| `internal/uat` | `phase0{1-6}_test.go` | End-to-end UAT per development phase |
| `cmd/torrent` | `main_test.go` | CLI entry point |

**UAT tests** (`internal/uat/`) verify complete phase scenarios end-to-end. They are organized by phase and build on the fixtures in `testdata/`.

### design-kube

Tests live alongside source under `pkg/`:

```
design-kube/
├── pkg/api/
└── pkg/storage/
```

### coding-problems

Problem solutions may include table-driven tests in `main_test.go` (where present). Not all categories have tests.

## Live Tests (learn-you-a-torrent)

Some UAT tests require a real network and a real `.torrent` file. They are skipped by default using `t.Skip()` guards controlled by environment variables:

| Variable | Value | Effect |
|----------|-------|--------|
| `LIVE_TRACKER` | `1` | Enables live tracker announce test in `phase02_test.go` |
| `LIVE_TORRENT_PATH` | path to a `.torrent` file | Provides the torrent for the live announce test |
| `LIVE_TRACKER_URL` | tracker URL | Overrides the announce URL from the `.torrent` file |

To run live tests:

```bash
cd learn-you-a-torrent
LIVE_TRACKER=1 LIVE_TORRENT_PATH=/path/to/real.torrent go test ./internal/uat/...
```

Use only legitimate public-domain torrents for live testing.

## Test Fixtures

`learn-you-a-torrent/testdata/` contains:

| File | Purpose |
|------|---------|
| `minimal.torrent` | Synthetic single-file torrent for offline unit tests |
| `gen_minimal.go` | Generator script that produced `minimal.torrent` |

The `minimal.torrent` uses `http://example.com/announce` as the tracker URL — it will not connect to a real tracker. Use `LIVE_TORRENT_PATH` to point at a real torrent for integration testing.

## File Naming Conventions

- Unit test files: `<source_file>_test.go` in the same package directory
- UAT test files: `phase<NN>_test.go` in `internal/uat/`
- Shared test helpers: `helpers_test.go` in `internal/uat/`
