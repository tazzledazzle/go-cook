---
phase: 3
slug: peer-handshake-messages
status: draft
nyquist_compliant: true
created: 2026-07-10
---

# Phase 3 — Validation Strategy

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Go testing + net.Pipe |
| **Quick run** | `go test ./internal/peer/...` |
| **Full suite** | `go test ./...` |

## TDD Gate

Each plan: RED must fail before feat commit. Record failure in SUMMARY.md.

| Plan | Requirement | RED command |
|------|-------------|-------------|
| 03-01 | PEER-01 | `go test ./internal/peer/... -run TestHandshake -v` (fail) |
| 03-02 | PEER-02 | `go test ./internal/peer/... -run TestHandshakeExchange -v` (fail) |
| 03-03 | PEER-03 | `go test ./internal/peer/... -run TestReadMessage -v` (fail) |
| 03-04 | PEER-03 | `go test ./internal/peer/... -run TestConnection -v` (fail) |
