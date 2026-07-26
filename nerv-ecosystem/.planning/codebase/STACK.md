# Technology Stack

**Analysis Date:** 2026-07-25

## Languages

**Primary:**
- Go 1.25.0 (module floor), built with toolchain `go1.26.2`+ — entire codebase (`main.go`, `cmd/`, `internal/store/`)

**Secondary:**
- SQL (SQLite dialect) — schema and triggers in `internal/store/migrations/0001_init.sql`

## Runtime

**Environment:**
- Go 1.25.0 floor, `go1.26.2` toolchain declared in `go.mod:3-5`. CGO is not required (pure-Go SQLite driver).

**Package Manager:**
- Go modules (`go mod`)
- Lockfile: present (`go.sum`, implied by `go.mod`; not read directly but expected alongside `go.mod`)

## Frameworks

**Core:**
- `spf13/cobra` v1.10.2 — CLI command tree, flags, help (`cmd/root.go`, `cmd/status.go`)

**Testing:**
- Go stdlib `testing` — all test files
- `github.com/stretchr/testify` v1.11.1 (`require` package only, observed usage) — assertions across `cmd/status_test.go`, `cmd/status_failure_test.go`, `internal/store/*_test.go`

**Build/Dev:**
- `golangci-lint` v2.12.2 (v2 config schema) — `.golangci.yml`
- `go test -race` — mandated by project convention (PLAT-03), wired in `Makefile:5-6` and CI

## Key Dependencies

**Critical:**
- `modernc.org/sqlite` v1.54.0 — pure-Go (CGO-free) SQLite driver with FTS5 compiled in; sole persistence engine, used only inside `internal/store/store.go`
- `github.com/spf13/cobra` v1.10.2 — CLI framework; used only inside `cmd/`

**Infrastructure:**
- `modernc.org/libc` v1.74.1, `modernc.org/mathutil` v1.7.1, `modernc.org/memory` v1.11.0 — transitive runtime support for `modernc.org/sqlite`
- `github.com/spf13/pflag` v1.0.9 — transitive flag parsing for Cobra
- `github.com/google/uuid` v1.6.0, `github.com/dustin/go-humanize` v1.0.1, `github.com/ncruces/go-strftime` v1.0.0, `github.com/remyoudompheng/bigfft` — transitive deps of `modernc.org/sqlite`
- `github.com/inconshreveable/mousetrap` v1.1.0 — transitive Cobra dep (Windows "don't double-click a CLI" warning)
- `golang.org/x/sys` v0.46.0, `gopkg.in/yaml.v3` v3.0.1 — transitive

## Configuration

**Environment:**
- `MODULAR_HOME` env var — overrides the default store directory; resolved in `internal/store/path.go:16-25` (`DefaultPath()`)
- No `.env` files present in the module; no secrets management needed at this phase

**Build:**
- `go.mod` (module `github.com/tazzledazzle/go-cook/nerv-ecosystem`) — the only build config file
- `.golangci.yml` — golangci-lint v2 schema (`version: "2"`), `run.go: '1.25'`, `linters.default: standard` plus `gosec` and `errcheck` explicitly enabled (see `CONCERNS.md` IN-01 for a redundancy note — `errcheck` is already in `standard`)
- `Makefile` — canonical local commands: `make test` (`go test ./... -race -count=1`), `make lint` (`golangci-lint run`), `make build` (`go build -o modular .`), `make smoke` (build + run `status` against a throwaway `MODULAR_HOME`)

## Platform Requirements

**Development:**
- Go 1.25+ toolchain (no CGO/C compiler required — `modernc.org/sqlite` is pure Go)
- No Docker/database daemon required; store is a single on-disk SQLite file

**Production:**
- No deployment target yet — this phase produces a local single binary (`./modular`) run directly on the operator's laptop. No CI/CD deploy step exists; `.github/workflows/nerv-ecosystem-ci.yml` (repo root, outside this module) runs `go test -race` and `golangci-lint` only, scoped by path filter to `nerv-ecosystem/**`.

## Planned, Not Yet Present (from research)

The following stack elements are documented in `.planning/research/STACK.md` and `.planning/research/ARCHITECTURE.md` for Phases 2–5 but have **no code, dependency, or `go.mod` entry yet**:

- `spf13/viper` v1.21.0 — layered CLI config (Phase 2+, "no real setting exists to layer yet" per `01-SKELETON.md` Out of Scope)
- `oras.land/oras-go/v2` v2.6.2 — OCI artifact push/pull client (Phase 3 publish)
- `zot` / `distribution/distribution` — local OCI registry (Phase 3 publish)
- `go.opentelemetry.io/otel` v1.44.0 (+ SDK, Prometheus exporter) — instrumentation, generated into scaffolded services (Phase 2 templates)
- `github.com/Masterminds/semver/v3` v3.5.0 — semver parsing/constraint checks (Phase 3 publish gate)
- `golang.org/x/exp/apidiff` — Go API breaking-change detection (Phase 3 publish gate)
- `dominikbraun/graph` — in-memory DAG for dependency blast radius (Phase 4 deps)
- `github.com/blevesearch/bleve/v2` v2.6.0 — optional richer full-text search, only if SQLite FTS5 proves insufficient (Phase 5 search)
- `github.com/prometheus/client_golang` v1.24.0 — CLI's own optional self-metrics
- `github.com/testcontainers/testcontainers-go` v0.43.0 — integration tests against a real registry container (Phase 3)
- `github.com/google/go-cmp` — deep-equality diffs in generator/manifest tests (Phase 2)
- `sigs.k8s.io/yaml` / `gopkg.in/yaml.v3` (direct use) — validating generated Helm/K8s manifests in tests (Phase 2)
- `goreleaser` — optional cross-compile/release tooling for the `modular` binary itself

---

*Stack analysis: 2026-07-25*
