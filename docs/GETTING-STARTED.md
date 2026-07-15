<!-- generated-by: gsd-doc-writer -->
# Getting Started

This guide walks you through cloning the repository and running each sub-project for the first time.

## Prerequisites

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | >= 1.22 | `learn-you-a-torrent` requires 1.26.2; `design-kube` requires 1.24.0 |
| git | any | To clone the repo |
| etcd | 3.x | Required only to run `design-kube` locally |

Check your Go version:

```bash
go version
```

## Clone the Repository

```bash
git clone https://github.com/tazzledazzle/go-cook.git
cd go-cook
```

There is no root `go.mod`. Each sub-project is an independent Go module — navigate into the relevant directory before running any Go commands.

## Run the Root HTTP Server

The root `main.go` is a standalone file with no `go.mod`. Run it directly:

```bash
go run main.go
```

Visit `http://localhost:8080/{any-text}` in a browser. The server renders a template response using the path segment as a variable.

> **Note:** `static/text.html` must exist at the working directory for the template to render. The file is not included in the repository; create it or the handler will silently fail.

## Run learn-you-a-torrent

```bash
cd learn-you-a-torrent
go mod download
go run ./cmd/torrent <path-to-file.torrent>
```

The CLI accepts a single positional argument — the path to a `.torrent` file. It downloads the torrent to the current working directory and prints progress to stdout.

A synthetic fixture is available for offline testing:

```bash
go run ./cmd/torrent testdata/minimal.torrent
```

## Run design-kube

`design-kube` requires a running etcd instance on `localhost:2379` (the default endpoint is hardcoded in the source):

```bash
# Start etcd locally (e.g. via Docker)
docker run -d --rm -p 2379:2379 quay.io/coreos/etcd:v3.5.0 \
  etcd --listen-client-urls http://0.0.0.0:2379 \
       --advertise-client-urls http://localhost:2379

cd design-kube
go mod download
go run ./cmd/apiserver
```

## Explore coding-problems

Each problem category is a self-contained directory with a `main.go` (and sometimes a `go.mod`):

```bash
cd coding-problems/arrays-and-strings
go run main.go
```

Navigate to any category under `coding-problems/` and run `go run main.go` to execute the solutions.

## Run the Tests

```bash
# Root package
go test ./...

# learn-you-a-torrent (unit + integration)
cd learn-you-a-torrent && go test ./...

# design-kube
cd design-kube && go test ./...

# A specific coding-problems category
cd coding-problems/arrays-and-strings && go test ./...
```

Live tracker tests in `learn-you-a-torrent` are skipped by default. See [docs/TESTING.md](TESTING.md) for how to enable them.

## Next Steps

- [docs/ARCHITECTURE.md](ARCHITECTURE.md) — overview of each sub-project and how the pieces fit together
- [docs/DEVELOPMENT.md](DEVELOPMENT.md) — build commands, code style, and branch conventions
- [docs/TESTING.md](TESTING.md) — test structure, naming conventions, and live-test env vars
- [docs/CONFIGURATION.md](CONFIGURATION.md) — all hardcoded values and environment variables
