# Integrations

**Analysis Date:** 2026-07-25
**Scope:** Implemented Phase 1 surface only. Later phases noted as planned.

## External Systems (runtime)

| System | Role today | Notes |
|--------|------------|-------|
| Local filesystem | Store file + future templates | `$MODULAR_HOME/registry.db` or `~/.modular/registry.db`; `--store-path` override |
| SQLite (via `modernc.org/sqlite`) | Embedded registry | WAL mode, FTS5; no network daemon |
| OS user home | Default data root | Via `os.UserHomeDir` in `internal/store/path.go` |

**None required for `modular status`:** no cloud accounts, no always-on registry daemon, no Docker for the core CLI path.

## CI / Dev tooling integrations

| Integration | Path | Purpose |
|-------------|------|---------|
| GitHub Actions | `../.github/workflows/nerv-ecosystem-ci.yml` (go-cook repo root) | `go test -race` + golangci-lint on `nerv-ecosystem/**` |
| golangci-lint | `.golangci.yml` (v2 schema) | Local + CI lint |
| Make | `Makefile` | `test` / `lint` / `build` / `smoke` targets |

## Planned integrations (not implemented)

From project research / Phase 2+ CONTEXT — do not treat as present code:

| Integration | Phase | Purpose |
|-------------|-------|---------|
| `embed.FS` + optional FS template dirs | 2 Generate | Scaffold Go/Java/Python |
| OpenTelemetry → Prometheus exporter (in generated apps) | 2 | Observability stubs in templates |
| OCI layout via `oras-go` | 3 Publish | Local artifact store (no zot required for MVP) |
| Structural `apidiff` (Go) + policy gate (Java/Python) | 3 | Semver publish gate |
| Dependency graph edges in SQLite | 3–4 | `deps --graph` |
| FTS5 search over projects | 5 | `modular search` (FTS table already exists from Phase 1) |

## Integration constraints

- `cmd/` must not import `database/sql` — all DB access through `internal/store`
- Nested module under `go-cook` git worktree — commits land on outer repo; no nested `.git`
- Beads (`bd`) is the execution/work memory SoT; GSD `.planning/` holds plans/docs
