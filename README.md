<!-- generated-by: gsd-doc-writer -->
# go-cook

A personal Go learning repository containing hands-on projects, system design explorations, and algorithm practice — covering everything from HTTP servers and BitTorrent clients to Kubernetes-style orchestration and LeetCode-style problem categories.

## Repository Structure

```
go-cook/
├── main.go                  # Root HTTP server (gorilla/mux, port 8080)
├── first-go-application/    # Introductory Go concepts and exercises
├── learn-you-a-torrent/     # From-scratch BitTorrent client (TDD-first)
├── design-kube/             # Kubernetes-style container orchestrator (etcd backend)
└── coding-problems/         # Algorithm and data structure practice by category
    ├── arrays-and-strings/
    ├── graphs/
    ├── hash-map-set/
    ├── linked-lists/
    ├── prefix-sum/
    ├── queues/
    ├── sliding-window/
    ├── stacks/
    └── two-pointers/
```

## Prerequisites

- Go >= 1.22

## Installation

```bash
git clone https://github.com/tazzledazzle/go-cook.git
cd go-cook
```

Each sub-project manages its own `go.mod`. Navigate to the relevant directory and install dependencies:

```bash
# Example: design-kube (uses etcd client)
cd design-kube
go mod download

# Example: learn-you-a-torrent
cd learn-you-a-torrent
go mod download
```

## Quick Start

**Root HTTP server**

```bash
go run main.go
# Listens on :8080 — visit http://localhost:8080/{any-text}
```

**design-kube (Kubernetes-style orchestrator)**

```bash
cd design-kube
go run ./cmd/...
```

**learn-you-a-torrent (BitTorrent client)**

```bash
cd learn-you-a-torrent
go run ./cmd/...
```

## Running Tests

```bash
# Root package tests
go test ./...

# Per sub-project
cd learn-you-a-torrent && go test ./...
cd design-kube        && go test ./...
cd coding-problems/arrays-and-strings && go test ./...
```

## Projects

### Root HTTP Server

A minimal HTTP server using `github.com/gorilla/mux` that serves a template response for any `/{text}` route on port 8080.

### first-go-application

Introductory exercises covering core Go concepts: time, functions with multiple return values, defer, and structs.

### learn-you-a-torrent

A from-scratch BitTorrent client written in Go using TDD. Implements bencode parsing, tracker communication, the peer wire protocol (BEP 3), piece verification via SHA1, and disk writes — all using the Go standard library with minimal external dependencies.

Modules: `bencode`, `tracker`, `peer`, `pieces`, `downloader`, `file`.

### design-kube

A Kubernetes-style container orchestration platform built from scratch. Uses etcd as the state store and targets declarative desired state, self-healing, and an API-first control plane. Tech stack: etcd, containerd (CRI), eBPF networking, Envoy/Gateway API ingress.

### coding-problems

Algorithm and data structure solutions organized by technique category: arrays & strings, graphs, hash maps & sets, linked lists, prefix sums, queues, sliding window, stacks, and two pointers.
