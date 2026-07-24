# Project Research Summary

**Project:** Nerv Ecosystem (Modular-style developer platform CLI)
**Domain:** Multi-language (Go/Java/Python) developer-platform CLI — project generator + semver-gated publish + OCI dependency graph + observability scaffolding
**Researched:** 2026-07-24
**Confidence:** MEDIUM-HIGH

## Executive Summary

This is a single-binary, local-first "paved road" CLI in the Backstage/Modular tradition: `generate` scaffolds Go/Java/Python services from local templates, `publish` gates releases on real semver-vs-API-diff compliance, `deps --graph` surfaces consumer blast radius, and `search` makes the registry queryable. All four research tracks converge on the same shape: a **modular monolith**, not a distributed system — one Cobra command tree over `internal/` domain packages, one embedded SQLite store (structured tables + FTS5 search) as the single system of record, and a local OCI artifact store for package versions. Nothing here needs a network boundary; the "platform" framing from the original Tableau Modular/Nerv system was an organizational artifact of many teams, not a technical requirement this laptop-scale rebuild needs to reproduce.

The recommended approach is staged, not simultaneous: build `generate` first (it's the only mechanically straightforward pillar and the prerequisite for everything else), then `publish` (the single highest-risk, highest-differentiation phase — real Go API-diffing via `apidiff`, with a simpler manifest-declared breaking-flag gate for Java/Python in v1), then `deps` and `search` as read-only query layers over the same store that `publish` populates. This mirrors both the feature-dependency graph (`deps` cannot exist before `publish` has written edges; `search` shares the registry rather than being a fourth data store) and the project's own stated demo flow.

The dominant risk is **trusting structural heuristics as ground truth** in two places: the semver/API-diff gate (which can both miss real breaks and flag harmless ones — especially for Python, where "public API" isn't language-enforced) and the local OCI registry's garbage collection (a documented, repeatedly-real upstream bug class in `distribution/distribution`). The mitigation for both is the same discipline already mandated by this project's process: write the regression test for the known failure mode *before* writing the feature (shared-blob-survives-sibling-deletion for the registry; known-tricky compatible-looking-but-breaking cases per language for the gate). A secondary risk — template drift with zero provenance — is cheap to prevent now (write a generation manifest on every `generate`) and expensive to fix later, so it belongs in the very first vertical slice even though drift-detection itself is out of scope.

## Key Findings

### Recommended Stack

Core stack: **Go 1.25/1.26**, **Cobra + Viper** for the CLI surface, **oras-go v2** as the OCI client (unifies remote-registry, on-disk OCI-layout, and in-memory targets behind one API — this is the load-bearing library for the registry-conflict resolution below), **modernc.org/sqlite** (pure Go, no CGO, FTS5 built in) as the single embedded store, **OTel Go SDK + Prometheus exporter** for generated-service instrumentation, and **Masterminds/semver/v3** + **golang.org/x/exp/apidiff** for the Go-side publish gate. `testcontainers-go` backs integration tests that need a real registry; `testify` + `go-cmp` cover unit-test ergonomics.

**Core technologies:**
- Go 1.25.x (build w/ 1.26 toolchain): every load-bearing dependency (oras-go, OTel SDK, prometheus client) has bumped its floor to 1.25 in 2026 — non-negotiable version floor.
- Cobra + Viper: matches the kubectl/docker/`gh`-style paved-road CLI feel; Viper scoped strictly to CLI config (registry URL, template root), not a general object store.
- oras-go v2: modern OCI Distribution + Image Spec client; one API for local-filesystem OCI-layout, in-memory, and remote-registry targets — this is what makes the "start local, swap to a real registry later" story work without an abstraction rewrite.
- modernc.org/sqlite + FTS5: pure-Go, single embedded store backing both the structured dependency graph and the full-text search index; avoids CGO cross-compile pain entirely.
- OTel Go SDK + `otel/exporters/prometheus`: traces + metrics are Stable in OTel-Go as of this release; direct Prometheus exporter is the simplest single-binary demo path (no Collector sidecar required per service).

### Expected Features

Table stakes and MVP are nearly identical here — Modular/Nerv's core value prop *is* generate + publish-gate + deps-graph + search, so none of the four pillars is optional.

**Must have (table stakes):**
- `generate --lang <go|java|python>` from local FS templates, producing Dockerfile + GHA CI stub + K8s/Helm skeleton + OTel instrumentation stub
- `publish` blocking breaking changes without a matching major bump (precedented by `cargo-semver-checks`/`buf breaking`'s structural-diff approach)
- `deps --graph <package>` showing consumer blast radius, fed by publish-time edge recording
- `search <query>` returning registered project metadata + language-specific build-config refs
- Local-first, single-binary operation — no cloud dependency for the core path
- Idempotent `generate` (fail/warn on existing target dir, don't silently overwrite)

**Should have (competitive differentiators):**
- Real API-diff-based semver enforcement per language (not a string version-bump check) — the single best "interview-defensible" claim in the project
- DOT/Graphviz export for `deps --graph`; audit-trail log of publish allow/block decisions
- Local Prometheus/Grafana demo dashboard wired to generated OTel stubs (once stubs are proven to compile/run)

**Defer (v2+):**
- Additional languages beyond Go/Java/Python
- Backstage-style catalog web UI / plugin architecture
- Template update/sync (Copier-style migrations into already-generated projects) — explicitly conflicts with this project's one-shot `generate` model

### Architecture Approach

Single Go binary, modular monolith: a Cobra routing layer (`cmd/`) with zero business logic calls into four cobra-free `internal/` domain packages (`generate`, `publish`, `deps`, `search`), each depending on narrow repository/port interfaces rather than concrete SQLite/OCI types (ports-and-adapters at the package level). This is what makes every vertical slice TDD-able against fakes before real adapters exist, and what would let a future "team-scale" version swap SQLite→Postgres or local-OCI→hosted-registry without touching domain logic.

**Major components:**
1. `internal/store` — the single embedded SQLite datastore (WAL mode, FTS5 virtual table); owns all schema and is the *only* package that imports `database/sql`. Every other domain package reads/writes through typed repository interfaces.
2. `internal/ociregistry` — the only package that touches OCI/registry APIs; isolates `publish`'s artifact push/pull from the domain logic that decides *whether* to publish.
3. `internal/generate` / `internal/publish` / `internal/deps` / `internal/search` — vertical, package-per-slice domain logic (not package-per-layer), matching the vertical-MVP-slice build philosophy directly.
4. `internal/templates` — template discovery/versioning, deliberately separate from `generate`'s rendering logic so template format can evolve independently.

### Critical Pitfalls

1. **Template drift with no provenance** — generated projects have no record of which template/version produced them, making future drift undetectable. Avoid by writing a generation manifest (`.nerv-manifest.json`: template name/version, params, timestamp) on every `generate`, from the first vertical slice, even though drift-*detection* itself is out of scope for v1.
2. **Semver/API-diff gate trusted as ground truth** — structural diffing (Go `apidiff`, Java `japicmp`, Python `griffe`) can neither see behavioral breaks nor avoid noisy false positives (especially Python, where "public" isn't language-enforced without `__all__`). Avoid by requiring every generated template to explicitly declare its public surface at generation time, and by testing the gate against documented tricky edge cases per language, not just obvious renames.
3. **Local OCI registry GC/reference-counting bugs** — a real, repeatedly-documented upstream bug class (`distribution/distribution` #4191, #4249) where GC deletes blobs still referenced by another manifest, or never collects at all. Avoid by keying blobs strictly by content digest and writing the shared-blob regression test (push two manifests sharing a base layer, delete one, assert the other's blob survives) before writing any storage/eviction code — or simplest of all, ship with **no GC** for v1 (unbounded local disk growth is safer than a buggy eviction policy at laptop scale).
4. **Bespoke CLI hides the native toolchain** — reimplementing `go build`/`mvn`/`pip` behavior inside the platform CLI instead of orchestrating them visibly is a top platform-adoption failure. Avoid by having generated CI invoke real native build commands and by always logging the exact underlying commands `modular` runs.
5. **OTel stubs demo one happy-path span, then silently fragment traces** — context propagation across process/async boundaries needs explicit wiring per language; auto-instrumentation only covers one blessed HTTP client/server pair. Avoid with a golden-path test: generate two services, call one from the other via the template's default client, assert a single trace ID spans both.

## Conflict Resolutions

The four research files disagree on three implementation choices. Resolved here against PROJECT.md's constraints (laptop-first single binary, TDD vertical slices, Go/Java/Python v1 scope, no unrunnable distributed system):

### 1. Local OCI registry: zot daemon vs. filesystem `ocilayout`

- **Conflict:** STACK.md recommends standing up `zot` (Docker container) as the primary local registry for its native Referrers API and search extension. ARCHITECTURE.md's Anti-Pattern 3 explicitly warns against requiring *any* running registry daemon for a laptop-first CLI, recommending filesystem-backed `ocilayout` (no listener, no process) instead.
- **Resolution — default to filesystem `ocilayout` via oras-go v2 for the core `publish`/`deps`/`search` path.** oras-go v2 already unifies OCI-layout-on-disk, in-memory, and remote-registry targets behind one `oras.Copy` API — implement `internal/ociregistry` against that interface so the storage backend is swappable without touching `publish`'s domain logic (Pattern 1, ports-and-adapters). This satisfies "zero external process" and keeps the demo a true single-binary `go run`.
- **zot remains a documented, optional escape hatch** — stand it up via docker-compose only if/when you want to demo the Referrers-API-based "attach dependency metadata as an OCI artifact" differentiator, or want real `docker`/`crane` interop for show. Treat it as a v1.x stretch demo, not an MVP dependency.

### 2. Semver gate depth for Java/Python: full API-diff vs. manifest-declared breaking flag

- **Conflict:** STACK.md explicitly recommends a two-tier gate — real `apidiff` for Go, a manifest-declared `breaking: true` policy flag for Java/Python — and lists "deep AST/bytecode diffing across all three languages" under What NOT to Use. FEATURES.md and PITFALLS.md both describe real structural diffing for Java (`japicmp`/`revapi`) and Python (`griffe check` against an explicit `__all__`) as the differentiator and as the correct fix for the "heuristic trusted as ground truth" pitfall.
- **Resolution — ship the two-tier gate for v1 MVP, staged real-diff upgrade for v1.x, Python before Java.** Build real Go `apidiff`-based enforcement first (best tooling, matches the CLI's own implementation language, highest interview-story value). For Java/Python in v1, use the manifest-declared breaking-flag + Conventional-Commits-style `BREAKING CHANGE:`/`!` convention check — explicitly documented as *policy-enforced, not compiler-verified*, which is itself a legitimate scoping talking point. As a v1.x differentiator (not blocking the four-pillar demo), upgrade **Python first** to real `griffe check` diffing, since it only requires one library and directly fixes Pitfall 2's Python-specific false-positive risk — but this *requires* scaffolding an explicit `__all__` in every generated Python `__init__.py` from day one in `generate` (cheap now, expensive to retrofit per the Technical Debt Patterns table). Upgrade Java to `japicmp`/`revapi` only if time remains after Python.
- **Regardless of tier:** never let "what is public" be an accidental language default — Go's export-letter convention already gives this for free; scaffold it explicitly for Java (module boundary) and Python (`__all__`) even under the manifest-flag tier, since it's a near-zero-cost hook for the later upgrade.

### 3. Observability pipeline: OTel Collector vs. direct Prometheus exporter

- **Conflict:** STACK.md presents both the OTel Collector (fan-out to Prometheus/Tempo, the "proper" reference pipeline) and the direct `otel/exporters/prometheus` (skip the Collector, scrape the service directly) as viable, without picking a default for generated-service templates.
- **Resolution — default generated services to the direct Prometheus exporter for v1.** This keeps the OTel instrumentation stub a single-process concern (no sidecar container, no extra compose service) for the MVP `generate` slice, matching "table stakes: OTel stub, doesn't need real backend integration, just wiring." Promote to the OTel Collector + Prometheus + Grafana docker-compose stack as the v1.x differentiator ("local Prometheus/Grafana demo dashboard") once stubs are proven to compile and emit metrics — this is also the natural point to add the cross-service trace-propagation golden-path test (Pitfall 5), since the Collector is where multi-service trace fan-in becomes visually demoable in Grafana.

## Implications for Roadmap

Based on combined research, the phase order is dictated almost entirely by the feature-dependency graph in FEATURES.md (`publish` requires `generate`'s output; `deps` requires `publish`'s edges; `search` shares the registry) — this is not a judgment call, it's a hard dependency chain, and it matches PROJECT.md's own stated demo flow exactly.

### Phase 1: `generate` — scaffolding + provenance + store foundation
**Rationale:** The only mechanically straightforward pillar and the load-bearing prerequisite for every other command; also the first place `internal/store`'s schema (Projects table) and `internal/templates` need to exist.
**Delivers:** `modular generate --lang <go|java|python> --name <svc> --team <team>` producing a project with Dockerfile, GHA CI stub, K8s/Helm skeleton, OTel instrumentation stub (direct Prometheus exporter, per resolution #3), and a generation manifest recording template/version/params.
**Addresses:** All "Launch With (v1)" `generate`-related items from FEATURES.md table stakes.
**Avoids:** Pitfall 1 (template drift — manifest written from slice one), Pitfall 4 (bespoke-toolchain drift — generated CI shells out to real `go build`/`gradle`/`pytest`), the path-traversal/symlink security mistakes (validate name/team/output-path params before any filesystem write).

### Phase 2: `publish` — semver gate + OCI artifact store
**Rationale:** Second in the dependency chain; also the highest-research-risk, highest-differentiation phase, so it should be tackled once `generate` has produced something with a public API surface to diff against.
**Delivers:** `modular publish` with real Go `apidiff`-based gate + `Masterminds/semver/v3` bump validation; manifest-declared breaking-flag gate for Java/Python (two-tier, per resolution #2); artifact push/pull via `oras-go v2` against filesystem `ocilayout` (per resolution #1); `store.Versions`/`store.Edges` writes.
**Uses:** `oras-go v2`, `golang.org/x/exp/apidiff`, `Masterminds/semver/v3` from STACK.md.
**Implements:** `internal/publish` + `internal/ociregistry` from ARCHITECTURE.md, with the push-before-metadata-write sequencing so a failed push never leaves a dangling Version row.
**Avoids:** Pitfall 2 (heuristic-as-ground-truth — explicit public-surface declaration per language, tested against documented tricky cases) and Pitfall 3 (registry GC bugs — content-digest-keyed storage, shared-blob regression test written before any eviction logic, no GC for v1).

### Phase 3: `deps --graph` — dependency graph + blast radius
**Rationale:** Cannot exist before `publish` has written publisher/consumer edges; purely a read-only query layer once that data exists.
**Delivers:** `modular deps --graph <package>` building an in-memory DAG (`dominikbraun/graph`) from `store` edges, computing direct + transitive consumer blast radius, rendered as table/ASCII tree (with DOT/Graphviz export as a fast v1.x follow-on).
**Addresses:** The `deps --graph` table-stakes requirement from FEATURES.md, and its "enhances publish" relationship (surfacing blast radius before a breaking publish).
**Avoids:** The performance trap of re-walking the full FS index per query — build the DAG from the store's persisted edges, not a directory scan.

### Phase 4: `search` — shared registry query layer
**Rationale:** Simplest of the four pillars once the registry exists; explicitly shares the same store as `generate`/`publish`/`deps` rather than being a fourth independent data store.
**Delivers:** `modular search <query>` over the SQLite FTS5 index, joined to Projects/Versions for build-config refs, language, and owning team.
**Addresses:** The `search` table-stakes requirement, staged toward the v1.x "return actual file content, not just refs" enhancement.
**Avoids:** Search-index staleness — reads must reflect a `publish` that just completed in the same session with no manual reindex step (test this explicitly).

### Phase 5 (v1.x, post-MVP): Observability pipeline polish + differentiators
**Rationale:** All four demo-flow commands must work end-to-end before this phase starts; this phase upgrades quality/depth rather than adding new commands.
**Delivers:** OTel Collector + Prometheus + Grafana docker-compose stack (resolution #3 promotion), cross-service trace-propagation golden-path test (Pitfall 5), Python real-diff upgrade via `griffe` (resolution #2 promotion), publish audit-trail JSONL log, template version pointer/rollback.

### Phase Ordering Rationale

- Order is fixed by data dependency, not by architectural preference: `generate` → `publish` → `deps` → `search` is the only order in which each phase's prerequisites already exist when it starts (confirmed independently by FEATURES.md's Feature Dependencies graph and PROJECT.md's stated demo flow).
- `internal/store` and `internal/templates` are not separate phases — they are built incrementally as each vertical slice needs them (Phase 1 needs Projects + template discovery; Phase 2 needs Versions/Edges; Phase 3/4 are read-only against what's already there). This matches the "vertical slices, not horizontal layers first" build philosophy.
- Observability polish is deliberately deferred to Phase 5 rather than folded into Phase 1, because the OTel *stub* (simple, direct-exporter) is table stakes for Phase 1, but the OTel *pipeline* (Collector, dashboards, cross-service trace tests) is a differentiator that depends on having multiple generated services to test propagation across — which doesn't exist until after Phase 1 has run at least twice.

### Research Flags

Phases likely needing deeper research during planning:
- **Phase 2 (`publish`):** Highest-research-risk phase per PITFALLS.md's own framing. Needs a dedicated research pass on: (a) Go `apidiff`'s documented blind spots and how to build a test fixture set around them, (b) the Java/Python two-tier gate's exact manifest schema and Conventional-Commits convention check, (c) the OCI registry's digest-keyed storage and shared-blob regression test design.
- **Phase 5 (Python real-diff upgrade, if pursued):** `griffe check` against an explicit `__all__` has no direct Go-ecosystem precedent in this codebase's tooling choices — worth a focused research pass before implementation if this differentiator is prioritized.

Phases with standard patterns (skip research-phase):
- **Phase 1 (`generate`):** Cobra command structure, `text/template`/`embed.FS` rendering, and manifest-writing are all well-documented, conventional patterns (confirmed HIGH confidence in both STACK.md and ARCHITECTURE.md).
- **Phase 3 (`deps --graph`):** `dominikbraun/graph`'s DAG/traversal API is a standard, well-documented library usage; no novel design questions.
- **Phase 4 (`search`):** SQLite FTS5 virtual tables are a standard, well-documented pattern; the only design decision (share the registry, don't build a fourth store) is already resolved by ARCHITECTURE.md.

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH (core tooling) / MEDIUM (composition judgment calls) | Versions/dates verified against pkg.go.dev/GitHub release metadata; no single reference project combines exactly this stack, so how components compose together is a judgment call, not a documented fact |
| Features | MEDIUM-HIGH | Backstage/Copier/Cookiecutter/cargo-semver-checks/buf/ORAS claims verified against official docs; Tableau Modular-specific specifics are HIGH only where directly quoted from the project's own README blueprint, LOW where reconstructed (explicitly flagged in FEATURES.md itself) |
| Architecture | HIGH (Go ecosystem/tooling) / MEDIUM (scale-specific judgment calls) | Cobra/OTel/OCI library claims verified against official docs; the "modular monolith, not microservices" recommendation is a judgment call specific to this project's single-operator/laptop scale, not an external fact |
| Pitfalls | MEDIUM-HIGH | Critical pitfalls (registry GC bugs, apidiff blind spots, scaffolding-CLI path-traversal CVEs) are verified against official docs, upstream issue trackers, and published CVEs; the OTel context-propagation failure mode is corroborated across multiple independent vendor/practitioner sources but not a single canonical spec citation |

**Overall confidence:** MEDIUM-HIGH

### Gaps to Address

- **Zot vs. `ocilayout` is a resolved default, not a validated one** — the `ocilayout`-first decision (resolution #1) should be sanity-checked against oras-go v2's actual local-layout ergonomics early in Phase 2 planning, before committing to it as the only backend.
- **Java gate depth is genuinely under-specified** — all four research docs converge on "Go first, Python second" for real-diff upgrades, but none proposes a concrete Java plan beyond naming `japicmp`/`revapi` as candidates; treat Java as manifest-flag-only unless a future milestone explicitly prioritizes it.
- **Generation manifest schema has no canonical design yet** — PITFALLS.md flags this as "design the format now so it's addable later without breaking existing projects" but doesn't propose field-level schema; this should be a concrete Phase 1 planning decision, not left implicit.
- **No load-bearing decision yet on which OCI push/pull test double to use in Phase 1/2 unit tests vs. `testcontainers-go` integration tests** — STACK.md recommends `testcontainers-go` against a real registry container for integration coverage, but the ports-and-adapters interface implies unit tests should use an in-memory fake; both are needed, and the split needs to be made explicit in Phase 2 planning.
- **OTel Collector promotion timing (Phase 5) is a recommendation, not a hard rule** — if demo-day visual impact matters more than MVP velocity, this could be pulled earlier; flagged here as a deliberate, revisitable scope choice rather than a fixed constraint.

## Sources

### Primary (HIGH confidence)
- pkg.go.dev/github.com/spf13/cobra, pkg.go.dev/oras.land/oras-go/v2, pkg.go.dev/go.opentelemetry.io/otel, pkg.go.dev/github.com/prometheus/client_golang, pkg.go.dev/github.com/Masterminds/semver/v3, pkg.go.dev/modernc.org/sqlite — official module registry version/compatibility verification
- go.dev/doc/devel/release, go.dev/doc/go1.25, go.dev/doc/go1.26 — Go release history
- go.googlesource.com/exp/+/master/apidiff/README.md — apidiff tool semantics, documented blind spots
- Backstage official docs (backstage.io/docs/overview/what-is-backstage, Software Templates overview)
- github.com/obi1kenobi/cargo-semver-checks, buf.build/docs/breaking — structural-diff semver gate precedent
- oras.land/docs — ORAS/OCI artifact push/pull reference
- Cobra official docs (cobra.dev/docs/how-to-guides/working-with-commands) — command organization pattern
- github.com/google/go-containerregistry/pkg/registry, github.com/docker/oci (`ocilayout`) — in-process/no-daemon OCI registry options
- `distribution/distribution` GitHub issues #4191, #4249 — verified upstream registry GC bugs
- `google/agents-cli` security issues #50/#51, Kiota CVE-2026-59866, `dromara/RuoYi-Vue-Plus` #33, Backstage Scaffolder CVE-2026-24046 — published path-traversal/symlink CVEs in comparable scaffolding CLIs
- `nerv-ecosystem/README.md` and `nerv-ecosystem/.planning/PROJECT.md` — this project's own authoritative source blueprint and scope decisions

### Secondary (MEDIUM confidence)
- Community CLI-framework comparison articles (Cobra vs. urfave/cli vs. Kong) — used only to confirm ecosystem-fit rationale, not to override the project's stated Cobra preference
- OTel Collector + Prometheus + Grafana docker-compose reference architecture (multiple 2026-dated blog sources converging on the same shape)
- Marc Dougherty, "Go dependencies and API diffs" — independent corroboration of apidiff false positives
- Reqhiem.dev/Blenddata Copier-vs-Cookiecutter/Cruft comparisons — template-drift mechanics corroboration
- GitPlumbers/Upsun/InfraZen/zop.dev platform-engineering retrospectives — "bespoke CLI hides real tooling" adoption-failure pattern

### Tertiary (LOW confidence)
- Reconstructed details of the original Tableau Modular/Nerv system beyond what's directly quoted in `README.md` (e.g., exact rollback mechanism internals, original CLI syntax) — explicitly flagged LOW in the source blueprint itself and treated as design inspiration, not fact, throughout this synthesis

---
*Research completed: 2026-07-24*
*Ready for roadmap: yes*
