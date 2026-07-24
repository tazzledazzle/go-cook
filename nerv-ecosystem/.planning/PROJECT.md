# Nerv Ecosystem

## What This Is

A Modular-style developer platform CLI — inspired by Tableau's Nerv/Modular framework — that scaffolds compliant multi-language services, enforces semantic versioning on publish, surfaces dependency blast radius, and makes generated projects searchable. It is a personal playground and portfolio reconstruction: the original paved-road concepts (generator, semver-gated publish, dependency graph, project search, observability-by-default) rebuilt with a modern local-first stack you can run end-to-end on a laptop.

## Core Value

An engineer can generate a Go, Java, or Python service that is already CI-, Helm-, and observability-wired, then publish it only when the version bump matches the API change — with `deps` and `search` making blast radius and project lookup first-class.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Engineer can `generate` a new Go, Java, or Python project from local filesystem templates with parameterized name/team/language
- [ ] Generated projects include Dockerfile, Kubernetes + Helm templates, GitHub Actions CI stubs, and OpenTelemetry instrumentation stubs (Prometheus/Grafana-ready)
- [ ] Engineer can `publish` a package; breaking API changes are blocked unless the semver major bump matches (linter locally + publish gate)
- [ ] Dependency tooling tracks publishers/consumers via OCI + local registry + filesystem package index
- [ ] Engineer can `deps --graph <package>` to see consumer blast radius before a breaking release
- [ ] Engineer can `search <query>` to find a registered project and retrieve metadata including language-specific build config references
- [ ] Features ship as end-to-end vertical slices; each slice is implemented test-first (TDD) with failing tests before production code

### Out of Scope

- Replacing language build tools (Bazel, Maven, Gradle, Go toolchain) — Modular scaffolds into them, does not replace them
- Runtime service mesh or continuous deployment orchestration — Helm/K8s are generated starting points only
- Org-wide mandatory adoption / migration tooling — this is a single-operator playground
- Full historical language surface (C++, gRPC-as-primary, Groovy, Kotlin, Scala, Ruby, Rust, etc.) — v1 is Go + Java + Python only
- Real AWS S3 template store, Splunk, New Relic, Artifactory, or Tableau-internal registries — concepts preserved via local FS, OTel/Prometheus/Grafana, OCI + local registry + FS index
- Pixel-perfect reconstruction of Tableau CLI syntax or internal APIs

## Context

**Source blueprint:** `nerv-ecosystem/README.md` — retrospective technical design of Tableau's Modular framework (Nerv team, ~2017–2020 build window; ~7-year production life). Original problems: inconsistent scaffolding, unmanaged dependency sprawl, fragmented observability across Desktop/Server/Online.

**Audience:** You (Terence) — primary operator and extender; interview-ready as a defensible portfolio case study when slices are verified.

**Build philosophy:**
- Vertical MVP slices per meaningful capability (generate → publish gate → deps graph → search), not horizontal layers first
- Test-driven development is non-negotiable: assumptions and assertions are verified with failing tests before implementation; without verification the solution is not considered useful or meaningful
- Go as the CLI/platform implementation language (`golang-pro` standards: table-driven tests, `-race`, context propagation, golangci-lint)
- Architecture informed by microservices/platform boundaries (generator, registry, dependency graph, search index) without over-splitting a laptop demo into an unrunnable distributed system
- SRE/DevOps bar: golden-path observability stubs, CI templates, health-minded generated artifacts; local Prometheus/Grafana for demo dashboards

**Intended demo flow (one sitting):**
1. `modular generate` (Go/Java/Python) → skeleton with CI, Dockerfile, Helm, OTel stubs
2. `modular publish` → blocked on breaking change without major bump; succeeds when bump matches
3. `modular deps --graph` → consumer blast radius
4. `modular search` → registered project metadata + build config refs

## Constraints

- **Languages (v1):** Go, Java, Python only — prove multi-language paved road without boiling the ocean
- **Template store:** Local filesystem (not S3) — laptop-first, no cloud dependency for core path
- **Package/deps backend:** OCI + local registry + filesystem package index — modern stand-in for Artifactory/internal PyPI
- **Observability:** OpenTelemetry → Prometheus/Grafana (not Splunk/New Relic)
- **CI templates:** GitHub Actions (not GitLab/TeamCity)
- **Orchestration templates:** Kubernetes manifests + Helm charts
- **Process:** TDD on every vertical slice; no production code without a failing test first
- **Git layout:** Planning lives under `nerv-ecosystem/.planning/`; commits track to outer `go-cook` worktree (nested subdir — do not nest a second `.git`)
- **Tech modernization:** Faithful Modular *concepts*, current implementations where prudent

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Working Modular-style CLI (not docs-only or marketing site) | Blueprint is an engineering platform; value is runnable generate/publish/deps/search | — Pending |
| v1 languages: Go + Java + Python | Multi-language proof without full historical surface area | — Pending |
| Local FS templates; OCI + local registry + FS index; OTel→Prom/Grafana; GHA; K8s+Helm | Modern stand-ins for S3/Artifactory/Splunk/NR/GitLab while keeping paved-road semantics | — Pending |
| Vertical end-to-end slices + mandatory TDD | Incremental verification; each capability must prove useful before the next | — Pending |
| Go for the platform CLI | Idiomatic systems CLI; strong testing/race tooling; matches portfolio language | — Pending |
| Opt-in playground scope (no adoption mandate tooling) | Matches README non-goal; single-operator use | — Pending |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-07-24 after initialization*
