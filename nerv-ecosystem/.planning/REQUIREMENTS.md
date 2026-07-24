# Requirements: Nerv Ecosystem

**Defined:** 2026-07-24
**Core Value:** An engineer can generate a Go, Java, or Python service that is already CI-, Helm-, and observability-wired, then publish it only when the version bump matches the API change — with `deps` and `search` making blast radius and project lookup first-class.

## v1 Requirements

Requirements for initial release. Each maps to roadmap phases.

### Platform Foundation

- [x] **PLAT-01**: Operator can install/run a single Go binary CLI (`modular` or project-chosen name) with no required cloud accounts or always-on daemons for the core path
- [x] **PLAT-02**: Platform persists projects, versions, dependency edges, and search index in one embedded SQLite store (WAL + FTS5)
- [x] **PLAT-03**: Every vertical slice is implemented test-first (failing test before production code); race detector and table-driven tests cover domain packages

### Generate

- [ ] **GEN-01**: Engineer can `generate --lang go|java|python --name <svc> --team <team>` from local filesystem templates
- [ ] **GEN-02**: Generated project includes Dockerfile, language-specific GitHub Actions CI stub, Kubernetes manifests, and Helm chart skeleton
- [ ] **GEN-03**: Generated project includes OpenTelemetry instrumentation stub exporting metrics via the direct Prometheus exporter (no Collector required for v1)
- [ ] **GEN-04**: Generated CI invokes real native toolchain commands (`go test`/`gradle`/`pytest` as appropriate), not a bespoke reimplementation
- [ ] **GEN-05**: Every generate writes a provenance manifest (template name/version, params, timestamp) into the project
- [ ] **GEN-06**: Generate is idempotent: refuses to overwrite an existing non-empty target directory (fail/warn, no silent clobber)
- [ ] **GEN-07**: Generate validates name/team/output path against path-traversal and symlink escape before any filesystem write
- [ ] **GEN-08**: Generate registers the new project in the shared store with language and build-config refs
- [ ] **GEN-09**: Generated Python packages declare an explicit `__all__`; Go relies on export rules; Java templates declare a clear public module boundary (hooks for later real-diff upgrades)

### Publish

- [ ] **PUB-01**: Engineer can `publish` a package from a generated (or registered) project to the local OCI artifact store (filesystem `ocilayout` via oras-go)
- [ ] **PUB-02**: For Go packages, publish runs structural API diff (`apidiff`) against the last published version and blocks breaking changes unless the semver major bump matches
- [ ] **PUB-03**: For Java and Python packages, publish enforces a documented two-tier policy gate (manifest `breaking` flag and/or Conventional-Commits-style breaking markers) plus semver bump validation — explicitly policy-enforced, not full AST/bytecode verified in v1
- [ ] **PUB-04**: Successful publish records version metadata and consumer/publisher edges in the shared store (push artifact before writing metadata so failures leave no dangling version rows)
- [ ] **PUB-05**: Publish writes an append-only audit record of allow/block decisions with reason
- [ ] **PUB-06**: Local OCI storage keys blobs by content digest; v1 ships with no GC (shared-blob integrity preferred over eviction)

### Dependencies

- [ ] **DEPS-01**: Engineer can `deps --graph <package>` and see direct and transitive consumer blast radius from store edges
- [ ] **DEPS-02**: Graph is built from persisted publish edges (not a filesystem scan of projects)
- [ ] **DEPS-03**: Output includes a human-readable table or ASCII tree suitable for a terminal demo

### Search

- [ ] **SRCH-01**: Engineer can `search <query>` and find registered projects by name/metadata via FTS5
- [ ] **SRCH-02**: Search results include language, owning team, and language-specific build-config refs
- [ ] **SRCH-03**: Search reflects generate/publish updates in the same session without a manual reindex step

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Generate / Templates

- **GEN-V2-01**: Template active-version pointer per language with explicit rollback command
- **GEN-V2-02**: Optional Copier-style template update/sync into already-generated projects

### Publish / Semver

- **PUB-V2-01**: Real structural API diff for Python via `griffe` (requires `__all__` already scaffolded in v1)
- **PUB-V2-02**: Real structural API diff for Java via `japicmp`/`revapi` if time after Python
- **PUB-V2-03**: Optional zot (or equivalent) daemon backend for Referrers API / `crane` interop demos

### Dependencies / Search

- **DEPS-V2-01**: DOT/Graphviz export for `deps --graph`
- **SRCH-V2-01**: Search returns actual build-config file content, not only refs

### Observability

- **OBS-V2-01**: Optional docker-compose stack: OTel Collector + Prometheus + Grafana with demo dashboard
- **OBS-V2-02**: Cross-service multi-hop trace continuity golden-path test across two generated services

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Replacing Bazel/Maven/Gradle/Go toolchain | Modular scaffolds into native tools; does not replace them |
| Runtime service mesh / CD / GitOps apply | Helm/K8s are generated starting points only |
| Org-wide adoption mandate / policy portal | Single-operator playground |
| Languages beyond Go/Java/Python in v1 | Prove multi-language without boiling the ocean |
| Real AWS S3, Artifactory, Splunk, New Relic | Concepts preserved via local FS, OCI layout, OTel→Prometheus |
| Backstage-style hosted catalog web UI | Out of proportion to laptop CLI portfolio |
| Always-on registry daemon required for MVP | Conflicts with single-binary laptop-first path |
| Full behavioral/semver completeness across all API shapes | Open research problem; catch common 80%, document gaps |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| PLAT-01 | Phase 1 | Complete |
| PLAT-02 | Phase 1 | Complete |
| PLAT-03 | Phase 1 | Complete |
| GEN-01 | Phase 2 | Pending |
| GEN-02 | Phase 2 | Pending |
| GEN-03 | Phase 2 | Pending |
| GEN-04 | Phase 2 | Pending |
| GEN-05 | Phase 2 | Pending |
| GEN-06 | Phase 2 | Pending |
| GEN-07 | Phase 2 | Pending |
| GEN-08 | Phase 2 | Pending |
| GEN-09 | Phase 2 | Pending |
| PUB-01 | Phase 3 | Pending |
| PUB-02 | Phase 3 | Pending |
| PUB-03 | Phase 3 | Pending |
| PUB-04 | Phase 3 | Pending |
| PUB-05 | Phase 3 | Pending |
| PUB-06 | Phase 3 | Pending |
| DEPS-01 | Phase 4 | Pending |
| DEPS-02 | Phase 4 | Pending |
| DEPS-03 | Phase 4 | Pending |
| SRCH-01 | Phase 5 | Pending |
| SRCH-02 | Phase 5 | Pending |
| SRCH-03 | Phase 5 | Pending |

**Coverage:**
- v1 requirements: 24 total
- Mapped to phases: 24
- Unmapped: 0 ✓

---
*Requirements defined: 2026-07-24*
*Last updated: 2026-07-24 after roadmap creation (traceability mapped)*
