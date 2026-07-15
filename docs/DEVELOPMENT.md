<!-- generated-by: gsd-doc-writer -->
# Development

## Local Setup

Each sub-project is an independent Go module. There is no root `go.mod` — set up each project separately:

```bash
# learn-you-a-torrent (Go 1.26.2)
cd learn-you-a-torrent && go mod download

# design-kube (Go 1.24.0)
cd design-kube && go mod download

# coding-problems/arrays-and-strings (has its own go.mod)
cd coding-problems/arrays-and-strings && go mod download
```

The root `main.go` has no module — run it with `go run main.go` directly from the repo root.

## Build and Run Commands

### Root HTTP server

```bash
go run main.go                  # Run the server on :8080
go build -o bin/server main.go  # Build a binary
```

### learn-you-a-torrent

```bash
cd learn-you-a-torrent
go run ./cmd/torrent <file.torrent>   # Run the CLI
go build -o bin/torrent ./cmd/torrent # Build a binary
go test ./...                          # Run all tests
go test -race ./...                    # Run with race detector (recommended for peer goroutines)
go test ./internal/pieces/...         # Run a specific package
```

### design-kube

```bash
cd design-kube
go run ./cmd/...        # Run the API server
go build ./cmd/...      # Build all commands
go test ./...           # Run all tests
```

### coding-problems

```bash
cd coding-problems/<category>
go run main.go    # Execute solutions
go test ./...     # Run tests if present
```

## Code Style

All Go code in this repository follows standard Go conventions:

- **Format:** `gofmt` before committing. Run `gofmt -w .` in any module directory.
- **Vet:** `go vet ./...` must pass with no warnings.
- **Imports:** Use `goimports` to manage import grouping (stdlib first, then third-party).
- **TDD (learn-you-a-torrent):** Write failing tests before implementation. Each new internal package must have a `_test.go` file before adding source code.
- **Dependencies:** Keep external dependencies minimal. `learn-you-a-torrent` uses stdlib only for the core protocol; bencode must be implemented from scratch. `design-kube` uses the etcd v3 client — no other external packages.

## Branch Conventions

Based on commit history, the convention is:

```
leet(<category>): <short description>    # LeetCode / coding-problems work
feat(<project>): <short description>     # New feature in a sub-project
fix(<project>): <short description>      # Bug fix
learn(<topic>): <short description>      # Learning exercises / experiments
```

Examples from the commit log:
```
leet(linlis): maximum twin sum of a linked list
leet(linlis): reverse a list
```

## Adding a New Coding Problem

1. Place the solution in `coding-problems/<category>/main.go` (add a new category directory if needed).
2. If the category is new and needs its own module, run `go mod init github.com/tazzledazzle/go-cook/coding-problems/<category>` inside the directory.
3. Commit with `leet(<category-abbrev>): <problem name>`.

## PR Process

Before opening a pull request:

1. Run `gofmt -w .` in every modified module directory.
2. Run `go vet ./...` — zero warnings required.
3. Run `go test ./...` — all tests must pass.
4. For `learn-you-a-torrent` changes, also run `go test -race ./...`.
5. Commit message must follow the branch convention above.
