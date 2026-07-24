# Roadmap: Nerv Ecosystem

## Overview

Nerv Ecosystem is a single-binary Go CLI that rebuilds Tableau's Modular/Nerv paved-road platform at laptop scale: `generate` scaffolds a compliant Go/Java/Python service, `publish` gates releases on real semver-vs-API-diff compliance, `deps --graph` surfaces consumer blast radius, and `search` makes the registry queryable. The five phases below follow the project's own data-dependency chain — a shared SQLite store and CLI skeleton first, then `generate` (the only source of projects to publish), then `publish` (which writes the version/edge data `deps` and `search` read), then `deps`, then `search`. Every phase is a vertical, demoable slice; no phase is a horizontal layer (all-models, all-APIs, all-UI). **Process note: every plan in every phase is implemented test-first per PLAT-03 / `workflow.tdd_mode` — a failing test exists before the corresponding production code is written.**

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Platform Foundation** - Single-binary CLI skeleton, embedded SQLite store (WAL + FTS5), and TDD scaffolding every later phase builds on
- [ ] **Phase 2: Generate** - Engineer scaffolds a provenance-tracked Go/Java/Python service from local templates in one command
- [ ] **Phase 3: Publish** - Engineer publishes a package to a local OCI store; breaking changes are blocked without a matching semver major bump
- [ ] **Phase 4: Dependency Graph** - Engineer sees direct and transitive consumer blast radius for a package before a breaking release
- [ ] **Phase 5: Search** - Engineer finds any registered project and its build-config references by query, live in the same session

## Phase Details

### Phase 1: Platform Foundation
**Goal**: Operator has a working CLI binary with a persistent local store and enforced TDD conventions that every later feature phase builds on
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: PLAT-01, PLAT-02, PLAT-03
**Success Criteria** (what must be TRUE):
  1. Operator can install and run a single `modular` (or project-chosen name) binary with zero required cloud accounts or always-on daemons
  2. Running any command opens/initializes one embedded SQLite store file (WAL mode) with the FTS5 search table already created
  3. Domain packages have table-driven tests that pass under `go test -race`, proving the TDD scaffolding (failing test before production code) is in place before feature work begins
**Plans**: 2 plans (test-first / TDD mandatory per PLAT-03)
**UI hint**: no

Plans:
- [ ] 01-01-PLAN.md — Walking skeleton: Go module + Cobra CLI + WAL/FTS5 store, `modular status` proving it end-to-end (wave 1)
- [ ] 01-02-PLAN.md — Reopen safety, live FTS5 trigger sync, permission/exit-code hardening, `-race` + lint enforcement harness (wave 2)

### Phase 2: Generate
**Goal**: Engineer can scaffold a compliant, provenance-tracked Go/Java/Python service from local filesystem templates with one command
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: GEN-01, GEN-02, GEN-03, GEN-04, GEN-05, GEN-06, GEN-07, GEN-08, GEN-09
**Success Criteria** (what must be TRUE):
  1. Engineer can run `generate --lang go|java|python --name <svc> --team <team>` and get a new project directory populated from local filesystem templates
  2. The generated project contains a Dockerfile, a GitHub Actions CI stub that invokes the real native toolchain (`go test`/`gradle`/`pytest`), Kubernetes manifests, a Helm chart skeleton, and an OpenTelemetry instrumentation stub exporting metrics via the direct Prometheus exporter
  3. Every generate writes a provenance manifest (template name/version, params, timestamp) into the project
  4. Generate refuses to overwrite an existing non-empty target directory, and rejects name/team/output-path inputs that attempt path traversal or symlink escape — both before any filesystem write occurs
  5. The new project is registered in the shared store with its language and build-config references, and declares an explicit public API surface per language (Go export rules, Java module boundary, Python `__all__`)
**Plans**: TBD (test-first / TDD mandatory per PLAT-03)
**UI hint**: no

Plans:
- [ ] 02-01: TBD
- [ ] 02-02: TBD
- [ ] 02-03: TBD

### Phase 3: Publish
**Goal**: Engineer can publish a package to a local OCI artifact store, and breaking API changes are blocked unless the semver major bump matches
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: PUB-01, PUB-02, PUB-03, PUB-04, PUB-05, PUB-06
**Success Criteria** (what must be TRUE):
  1. Engineer can run `publish` on a generated/registered project and the artifact is pushed to a local filesystem OCI-layout store (via `oras-go`), with blobs keyed strictly by content digest
  2. For Go packages, publish runs a structural API diff (`apidiff`) against the last published version and blocks the publish when a breaking change is detected without a matching semver major bump
  3. For Java and Python packages, publish enforces the documented two-tier policy gate (manifest `breaking` flag and/or Conventional-Commits-style breaking marker) plus semver bump validation
  4. A successful publish records version metadata and consumer/publisher edges in the shared store only after the artifact push succeeds, so a failed push never leaves a dangling version row
  5. Every publish decision (allow or block, with reason) is appended to an audit record the engineer can inspect
**Plans**: TBD (test-first / TDD mandatory per PLAT-03)
**UI hint**: no

Plans:
- [ ] 03-01: TBD
- [ ] 03-02: TBD
- [ ] 03-03: TBD

### Phase 4: Dependency Graph
**Goal**: Engineer can see consumer blast radius for a package before making a breaking release
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: DEPS-01, DEPS-02, DEPS-03
**Success Criteria** (what must be TRUE):
  1. Engineer can run `deps --graph <package>` and see both direct and transitive consumers of that package
  2. The graph is built from persisted publish edges in the store, not a filesystem scan of projects
  3. Output renders as a human-readable table or ASCII tree suitable for a terminal demo
**Plans**: TBD (test-first / TDD mandatory per PLAT-03)
**UI hint**: no

Plans:
- [ ] 04-01: TBD
- [ ] 04-02: TBD

### Phase 5: Search
**Goal**: Engineer can find any registered project and its language-specific build-config references by query
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: SRCH-01, SRCH-02, SRCH-03
**Success Criteria** (what must be TRUE):
  1. Engineer can run `search <query>` and find registered projects by name/metadata via the FTS5 index
  2. Search results include language, owning team, and language-specific build-config references
  3. A project generated or published earlier in the same session appears in search results immediately, with no manual reindex step
**Plans**: TBD (test-first / TDD mandatory per PLAT-03)
**UI hint**: no

Plans:
- [ ] 05-01: TBD
- [ ] 05-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Platform Foundation | 0/2 | Planned    |  |
| 2. Generate | 0/3 | Not started | - |
| 3. Publish | 0/3 | Not started | - |
| 4. Dependency Graph | 0/2 | Not started | - |
| 5. Search | 0/2 | Not started | - |
