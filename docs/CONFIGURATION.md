<!-- generated-by: gsd-doc-writer -->
# Configuration

This repository is a Go learning monorepo containing independent subprojects. None of the subprojects use a shared configuration file or a root-level `.env` file. Each subproject has its own configuration surface, described below.

---

## Environment Variables

Most subprojects have no required environment variables and start without any setup. The one exception is the `learn-you-a-torrent` test suite, which gates optional live-network tests behind env vars.

| Variable | Required | Default | Description |
|---|---|---|---|
| `LIVE_TRACKER` | Optional | _(unset)_ | Set to `1` to enable live HTTP tracker announce tests in `learn-you-a-torrent`. Skipped when unset. |
| `LIVE_TORRENT_PATH` | Optional | _(unset)_ | Absolute path to a real `.torrent` file used by the live tracker test (`TestUAT_2_4b_liveTrackerOptional`). |
| `LIVE_TRACKER_URL` | Optional | _(unset)_ | Override the announce URL from the `.torrent` file when running live tracker tests. |

All three variables are consumed only by `learn-you-a-torrent/internal/uat/phase02_test.go`. No other source file in the repository reads from the environment.

---

## Config File Format

### `learn-you-a-torrent` — no config file

The torrent CLI accepts a single positional argument (the `.torrent` file path) and reads all runtime parameters from the torrent file itself (announce URL, piece hashes, file length). There is no application config file.

```
torrent download <file.torrent>
```

### `design-kube` — hardcoded etcd endpoint

The `design-kube` API server connects to etcd using a hardcoded endpoint list. The endpoint is set directly in `design-kube/cmd/apiserver/main.go`:

```go
client, err := etcd.NewClient([]string{"localhost:2379"})
```

There is no config file or env var override for this value. To point the server at a different etcd cluster, edit the endpoint slice in `main.go` before building.

### `main.go` (root) — hardcoded HTTP listen address

The root-level HTTP server listens on a hardcoded address:

```go
http.ListenAndServe(":8080", r)
```

No config file or env var controls this port.

---

## Required vs Optional Settings

No setting in any subproject causes a startup failure if an environment variable is absent. The three `LIVE_*` variables in the torrent test suite call `t.Skip(...)` when unset — they do not crash the process.

The `design-kube` etcd dependency (`localhost:2379`) will produce a connection error at runtime if etcd is not running, but this is a runtime dependency, not a missing configuration value.

---

## Defaults

| Location | Setting | Default Value | Set In |
|---|---|---|---|
| `main.go` (root) | HTTP listen address | `:8080` | `main.go:14` |
| `design-kube/cmd/apiserver/main.go` | etcd endpoints | `["localhost:2379"]` | `main.go:12` |
| `design-kube/pkg/storage/etcd/client.go` | etcd dial timeout | `5s` | `client.go:15` |
| `learn-you-a-torrent/internal/tracker/client.go` | BitTorrent listen port (announce) | `6881` | `client.go:32` |
| `learn-you-a-torrent/cmd/torrent/main.go` | Peer ID prefix | `-GO0001-000000` | `main.go:55` |

---

## Per-Environment Overrides

There are no `.env.development`, `.env.production`, or `.env.test` files in this repository. No `NODE_ENV`-style environment switching exists — the projects are Go binaries without framework-managed environment profiles.

For local development with a non-default etcd address, modify the endpoint slice in `design-kube/cmd/apiserver/main.go` directly and rebuild:

```bash
cd design-kube && go build ./cmd/apiserver
```

For running the optional live torrent network tests:

```bash
LIVE_TRACKER=1 LIVE_TORRENT_PATH=/path/to/real.torrent go test ./internal/uat/...
```
