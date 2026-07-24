<!-- GSD:project-start source:PROJECT.md -->
## Project

**Nerv Ecosystem**

A Modular-style developer platform CLI — inspired by Tableau's Nerv/Modular framework — that scaffolds compliant multi-language services, enforces semantic versioning on publish, surfaces dependency blast radius, and makes generated projects searchable. It is a personal playground and portfolio reconstruction: the original paved-road concepts (generator, semver-gated publish, dependency graph, project search, observability-by-default) rebuilt with a modern local-first stack you can run end-to-end on a laptop.

**Core Value:** An engineer can generate a Go, Java, or Python service that is already CI-, Helm-, and observability-wired, then publish it only when the version bump matches the API change — with `deps` and `search` making blast radius and project lookup first-class.

### Constraints

- **Languages (v1):** Go, Java, Python only — prove multi-language paved road without boiling the ocean
- **Template store:** Local filesystem (not S3) — laptop-first, no cloud dependency for core path
- **Package/deps backend:** OCI + local registry + filesystem package index — modern stand-in for Artifactory/internal PyPI
- **Observability:** OpenTelemetry → Prometheus/Grafana (not Splunk/New Relic)
- **CI templates:** GitHub Actions (not GitLab/TeamCity)
- **Orchestration templates:** Kubernetes manifests + Helm charts
- **Process:** TDD on every vertical slice; no production code without a failing test first
- **Git layout:** Planning lives under `nerv-ecosystem/.planning/`; commits track to outer `go-cook` worktree (nested subdir — do not nest a second `.git`)
- **Tech modernization:** Faithful Modular *concepts*, current implementations where prudent
<!-- GSD:project-end -->

<!-- GSD:stack-start source:research/STACK.md -->
## Technology Stack

## Recommended Stack
### Core Technologies
| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Go | 1.25.x (module floor `go 1.25`, build with 1.26.5 toolchain) | Implementation language / runtime | Every load-bearing dependency below (oras-go v2, OTel Go SDK, prometheus/client_golang, testcontainers-go) has bumped its `go.mod` floor to Go 1.25 in 2026 and only supports the "two latest" minor versions (1.25/1.26 as of July 2026). Pinning the module floor at 1.25 while building with the latest 1.26 patch (1.26.5, 2026-07-07) maximizes both compatibility and security-patch currency. |
| spf13/cobra | v1.10.2 (2025-12-04) | CLI command tree, flags, help, shell completion | De facto standard for Kubernetes-ecosystem-flavored CLIs (kubectl, Docker CLI, `gh`, Hugo). Matches the paved-road/platform-CLI feel this project is going for (`modular generate`, `modular publish`, `modular deps --graph`, `modular search` map directly onto Cobra's command/subcommand model) and is what the downstream roadmap explicitly asked to default to. |
| spf13/viper | v1.21.0 (2025-09-08) | Layered config (flags > env > file > defaults) for CLI-wide settings (registry URL, local template root, OTel collector endpoint) | Pairs natively with Cobra/pflag; needed because this CLI has real environment-dependent config (where's the local registry? where's the OTel collector?) beyond what flags alone should carry. Keep it CLI-config only — do not use it as a general-purpose object store. |
| oras.land/oras-go/v2 | v2.6.2 (2026-07-10) | OCI artifact push/pull/tag client — talks to the local registry for package publish/fetch and to the filesystem/in-memory stores for local-first caching | This *is* the modern (2025–2026) way to speak OCI Distribution + OCI Image Spec from Go without hand-rolling HTTP. It unifies remote-registry, OCI-layout-on-disk, and in-memory targets behind one `oras.Copy` API, which maps cleanly onto "local filesystem template store" + "OCI + local registry" in one client. |
| zot (`ghcr.io/project-zot/zot-linux-amd64`) | latest (CNCF Sandbox, actively released 2026) | Local-first OCI-native registry for `publish`/`deps` | Purpose-built OCI-only registry (not a Docker-v2 relic): single Go binary, no external DB, built-in auth, native OCI Referrers API (needed for attaching dependency/consumer metadata to published artifacts), and a REST search extension you can reuse for the `search` command's registry-side metadata. This is the more faithful 2025–2026 stand-in for "Artifactory" than the legacy `registry:2` image. |
| distribution/distribution (`registry:3.1.1`) | v3.1.1 (2026-05-01) | Fallback/simplest-possible local registry if zot is more than you want to stand up for a demo | Still the reference OCI Distribution Spec implementation and what most tutorials assume; keep as a documented fallback, not the primary choice, because it lacks native OCI-artifact/Referrers ergonomics and has no built-in auth or search. |
| go.opentelemetry.io/otel (+ `/sdk`, `/trace`, `/metric`) | v1.44.0 (2026-05-27) | OTel Go API/SDK — traces + metrics instrumentation stubs generated into every scaffolded service, and in the CLI itself | Traces and metrics are both "Stable" in OTel-Go as of this release (logs remain Beta — do not generate log-pipeline stubs as a hard requirement yet). This is the only viable "OpenTelemetry → Prometheus/Grafana" bridge that isn't vendor-specific (replaces Splunk/New Relic per the project's own constraints). |
| go.opentelemetry.io/otel/exporters/prometheus | matched to v1.44.0 line | Exposes an OTel `MeterProvider`'s metrics on a `/metrics` endpoint in Prometheus exposition format | Lets generated services skip running their own OTel Collector sidecar for the demo path — the app itself can be scraped directly by Prometheus. Use the OTel Collector (below) for the "proper" pipeline story; use this exporter for the simplest single-binary demo path. |
| otel/opentelemetry-collector-contrib (Docker image) | latest (rolling) | Central OTLP receiver that fans out to Prometheus (metrics) and optionally Tempo/Jaeger (traces) for the local demo stack | Standard reference architecture for "OTel → Prometheus/Grafana": services push OTLP to the Collector; Collector exposes a `prometheus` exporter endpoint that Prometheus scrapes; Grafana visualizes. Keeps generated services simple (just emit OTLP) while centralizing the pull-based Prometheus bridge. |
| prom/prometheus + grafana/grafana (Docker images) | latest stable tags | Local metrics storage + dashboards | Exactly what the project's own constraints specify; both ship official, well-maintained images and a documented `docker-compose` pattern with the Collector. |
### Supporting Libraries
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| github.com/Masterminds/semver/v3 | v3.5.0 (2026-04-30) | Parse/compare/validate semantic versions for the `publish` gate | Use for all "is this bump valid given the previous version and the detected change class" logic. Actively maintained, explicitly built for constraint-checking (unlike the smaller stdlib `golang.org/x/mod/semver`, which only does comparison/validation of strings, not constraint ranges — fine as a lighter alternative if you don't need constraints). |
| golang.org/x/exp/cmd/apidiff (+ `golang.org/x/exp/apidiff` as a library) | latest (`golang.org/x/exp` is unversioned/rolling) | Detects Go API-breaking changes between two versions of a Go package to recommend/enforce the correct semver bump | This is the real, Google-maintained tool for exactly the "breaking API change without a major bump should fail publish" requirement — but **only for Go packages**. See Stack Patterns by Variant below for how to handle Java/Python. |
| modernc.org/sqlite | latest (pure-Go, tracks upstream SQLite) | Embedded storage for the filesystem package index (`{package, semver, publisher, consumers[]}`) and the project search index/metadata store | Pure-Go, CGO-free SQLite driver — critical because this project's whole point is a single, easily cross-compiled, `-race`-testable Go binary; CGO drivers break that on every axis (cross-compile, Alpine/scratch images, `go test -race`, CI matrix simplicity). It compiles SQLite with FTS5, JSON1, and R-Tree enabled by default, so the same DB file backs both structured package/dependency records and the full-text `search` index (SQLite `CREATE VIRTUAL TABLE ... USING fts5(...)`) without a second search engine. |
| github.com/blevesearch/bleve/v2 | v2.6.0 (2026-04-30) | Optional richer full-text search (fuzzy matching, BM25 ranking tuning, faceting) for `modular search` | Reach for this only if SQLite FTS5's `MATCH`/BM25 turns out to be insufficient (e.g., you want typo-tolerant fuzzy search or faceted filters by language/team). Don't start here — it's a second storage engine and index-maintenance burden the MVP doesn't need. |
| github.com/prometheus/client_golang (`/prometheus`, `/prometheus/promauto`, `/prometheus/promhttp`) | v1.24.0 (2026-07-20) | Direct Prometheus instrumentation for the CLI's own optional self-metrics, and as the underlying exposition format the OTel Prometheus exporter emits | Use directly only for the CLI's own lightweight metrics (e.g., `modular publish` durations if you want a metrics endpoint on the CLI itself). Generated *service* stubs should go through the OTel SDK + OTel Prometheus exporter above for pipeline consistency — don't mix both instrumentation APIs inside generated templates. |
| github.com/stretchr/testify (`/assert`, `/require`) | v1.11.1 (2025-08-27) | Test assertions/require semantics on top of table-driven `testing.T` tests | Reduces assertion boilerplate in table-driven tests without fighting Go's idioms; explicitly still maintained at v1 (no v2 breaking changes planned), so no churn risk. |
| github.com/google/go-cmp | latest (`v0.7.0`, pulled transitively by OTel already) | Deep-equality diffs in tests (comparing generated file trees, parsed manifests, dependency-graph structs) | Better failure output than `reflect.DeepEqual` for the kind of structured comparisons this project's tests do a lot of (generated project skeletons, parsed OCI manifests). |
| github.com/testcontainers/testcontainers-go | v0.43.0 (2026-06-19) | Integration tests that spin up a real zot/registry container (and optionally an OTel Collector container) and drive `publish`/`deps` against it | This is how you test the OCI publish/pull path and registry-backed `deps --graph` logic honestly (real HTTP against a real OCI registry) instead of mocking the registry API, while still keeping tests hermetic and CI-friendly. |
| opencontainers/image-spec, opencontainers/go-digest | pulled transitively by oras-go v2 | OCI manifest/digest types | You'll touch these directly whenever you build custom OCI manifests/annotations for the dependency-graph metadata (e.g., embedding `consumers[]` as an OCI annotation or a Referrers-API-attached artifact). |
| sigs.k8s.io/yaml (or `gopkg.in/yaml.v3`) | latest | Round-trip YAML marshal/unmarshal for validating generated Helm/K8s manifests in tests | Use in generator tests to unmarshal emitted Helm/K8s YAML and assert on structure, rather than string-matching raw template output. `sigs.k8s.io/yaml` round-trips through JSON tags, which matters if you ever want the generated structs to also satisfy `apimachinery` types later; plain `yaml.v3` is fine if you're only asserting map/slice shape. |
### Development Tools
| Tool | Purpose | Notes |
|------|---------|-------|
| golangci-lint | Meta-linter (aggregates govet, staticcheck, gosec, revive, etc.) | Pin to **v2.12.2** (2026-05-06). v2's config schema is a breaking change from v1 — config files must start with `version: "2"` and linter config moved under a `linters:` block; run `golangci-lint migrate` if you ever start from a v1-era `.golangci.yml` example. Set `run.go` in the config to match your module's Go floor (e.g. `go: '1.25'`) rather than trusting the tool's own fallback default. |
| `go test -race` | Data-race detection | Non-negotiable per this project's own TDD/`golang-pro` constraints; run in CI on every push, not just locally. |
| `go vet` / `staticcheck` (via golangci-lint) | Static analysis | Already covered by golangci-lint's default linter set — don't add a separate standalone staticcheck step, it's redundant and doubles CI time. |
| `gofmt` / `goimports` | Formatting | Wire as a pre-commit or CI-gate step (`gofmt -l .` failing the build), not just an editor setting, so generated-template Go code and hand-written CLI code stay consistently formatted. |
| GitHub Actions (`actions/setup-go`, `golangci/golangci-lint-action`) | CI for the CLI's own repo, and the template this project generates into every scaffolded service | Use the official `golangci-lint-action` (auto-detects v2 config) rather than hand-rolling a `go install golangci-lint` step — it caches correctly and matches the pinned version above. |
| goreleaser (optional) | Cross-compile + release the `modular` CLI binary itself | Not required for the MVP demo, but worth flagging early since the roadmap likely wants a "ship the CLI" phase eventually; goreleaser is the standard tool for multi-platform Go binary releases + GitHub Release publishing. |
## Installation
# Core
# Observability
# Storage / search
# Supporting
# Dev dependencies
# Local infra (docker-compose services, not go.mod deps)
## Alternatives Considered
| Category | Recommended | Alternative | Why Not (as primary) |
|----------|-------------|-------------|--------------------|
| CLI framework | Cobra + Viper | urfave/cli/v3 | urfave/cli/v3 is genuinely simpler (no `init()` magic, plain structs, less ceremony) and is a legitimate choice for small internal tools — but this project's UX explicitly mimics kubectl/docker/gh-style paved-road tooling (`generate`, `publish`, `deps --graph`, `search`), and Cobra's ecosystem alignment/completion/doc-gen tooling is the better fit for that positioning. Revisit only if the command surface stays permanently small (it won't — `deps`/`search` alone imply growth). |
| CLI framework | Cobra + Viper | Kong | Kong's struct-tag-driven type safety is attractive, but it's a smaller ecosystem with less prior art for the specific "Kubernetes-adjacent paved-road CLI" shape this project is going for; Cobra's shell-completion and doc-generation are more mature. |
| Local registry | zot | distribution/distribution (`registry:3`) | `registry:2`/`registry:3` is the reference implementation and simplest to `docker run`, but has no OCI Referrers API, no built-in auth, and no search — you'd end up bolting all three onto it by hand. Keep it documented as a "if zot feels heavy" fallback only. |
| Local registry | zot | Harbor | Harbor is enterprise-grade (Postgres, Redis, Trivy, multi-container) — total overkill for a laptop demo and works against this project's own "no unrunnable distributed system" constraint. |
| Package/search index | SQLite (modernc.org/sqlite) + FTS5 | bbolt/BoltDB | BoltDB is a fine embedded KV store, but you'd have to hand-roll indexing/query logic for both the dependency graph and full-text search that SQLite gives you for free via SQL + FTS5 virtual tables. Only reach for bbolt if you want zero SQL and pure key-value semantics. |
| Package/search index | SQLite (modernc.org/sqlite) + FTS5 | bleve | bleve is a better *pure* search engine (BM25 tuning, faceting, fuzzy queries) but is a second storage engine to operate and keep in sync with the SQLite-backed dependency graph. Start with FTS5; graduate to bleve only if search quality genuinely becomes the bottleneck. |
| SQLite driver | modernc.org/sqlite (pure Go) | mattn/go-sqlite3 (CGO) | mattn is faster on write-heavy workloads and is still the most widely used driver, but requires CGO — which breaks trivial cross-compilation, complicates Docker multi-stage builds, and fights `-race` and Alpine/scratch images. Not worth it for this project's single-binary, CI-heavy, multi-platform-demo goals. |
| Breaking-change detection (Go) | `golang.org/x/exp/cmd/apidiff` | Custom AST diffing | apidiff already solves "did the exported surface change incompatibly" correctly and is Google-maintained; writing your own `go/ast`/`go/types` walker to reinvent this would be a multi-week side project with no payoff over the existing tool. |
| Metrics exposition (generated services) | OTel SDK + OTel Collector + Prometheus exporter | Direct `prometheus/client_golang` in every generated service | Direct instrumentation is simpler per-service but breaks the stated "OpenTelemetry → Prometheus/Grafana" pipeline goal and forgoes traces entirely (you'd need a second instrumentation API for tracing anyway). Standardizing on OTel end-to-end means every generated service gets both metrics and traces from one SDK. |
## What NOT to Use
| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Cookiecutter / Yeoman / Copier as the templating engine | Wrong runtime (Python/Node) for a Go-implemented CLI meant to ship as a single static binary; adds a hard external-interpreter dependency just to scaffold files, and fights the "single Go binary, easy laptop demo" goal. | Go stdlib `text/template` + `embed.FS` — embed all language/Dockerfile/Helm/GHA/OTel templates directly into the compiled `modular` binary at build time. No runtime dependency, trivially cross-compiles, and template files are versioned alongside the CLI's own source. |
| mattn/go-sqlite3 (or any CGO-based SQLite driver) as the package/search index backend | Requires a C toolchain at build time; breaks easy cross-compilation (`GOOS=darwin GOARCH=arm64 go build` style laptop-to-anywhere builds), complicates minimal/scratch Docker images, and is known to be awkward with `go test -race`. | `modernc.org/sqlite` — pure Go, FTS5/JSON1 built in, `CGO_ENABLED=0` works everywhere. |
| `registry:2` (classic Docker Distribution v2) as the *primary* dependency-management backend | Built around Docker's legacy image format with OCI support bolted on later; no OCI Referrers API (needed to attach dependency/consumer metadata to artifacts), no built-in auth, no search, no garbage-collection scheduling. | zot as primary; keep `registry:3` (current distribution/distribution) only as a documented minimal fallback. |
| `helm.sh/helm/v3` SDK or `k8s.io/client-go` as *generation-time* dependencies | This project explicitly generates Helm charts/K8s manifests as a starting point and is **not** a deployment orchestrator (stated non-goal). Pulling in the full Helm SDK or a Kubernetes API client just to emit static YAML text is a large, versioning-fragile dependency (client-go's version has to track a specific K8s server API version) for zero runtime benefit. | Plain `text/template` + `embed.FS`, same as other generated artifacts. If you want CI-time validation of generated charts, shell out to the standalone `helm lint`/`helm template` binaries in a test or GitHub Actions step — that's dependency-free from the Go module's perspective. |
| golangci-lint v1.x config examples found in older blog posts/tutorials | v1's flat config schema was replaced by v2's versioned, sectioned schema (`version: "2"`, linters grouped under `linters:`, new `formatters:` block) — copying a v1-era `.golangci.yml` verbatim into a v2 install either no-ops or errors. | golangci-lint v2.12.2 with a `version: "2"` config from the start, or run `golangci-lint migrate` on any v1 config you find in a reference project. |
| Splunk / New Relic clients (or any proprietary APM agent) | Explicitly out of scope per this project's own constraints — the whole point is replacing them with the open, laptop-runnable OTel → Prometheus/Grafana pipeline. | OTel Go SDK + OTel Collector + Prometheus + Grafana. |
| Deep AST/bytecode API-diffing across Go **and** Java **and** Python for the semver gate | Building or wiring three separate breaking-change detectors (Go: apidiff-class tooling; Java: something like `japicmp`; Python: something like `griffe`) is a multi-milestone distributed-systems-grade investment that doesn't fit a single-operator portfolio playground, and this project's own scope explicitly caps v1 at three languages without promising deep cross-language tooling parity. | Two-tier gate: (1) real `apidiff`-based enforcement for Go packages, since that's the CLI's own implementation language and the highest-value place to prove the concept end-to-end; (2) a manifest-declared breaking-change flag (a `breaking: true/false` field in the package's publish manifest, populated by a Conventional-Commits-style `BREAKING CHANGE:`/`!` convention check) for Java/Python, which is honest about being a *policy* gate rather than a *static-analysis* gate for those two languages. Document this asymmetry explicitly — it's a legitimate, defensible scoping decision, not a gap to hide. |
## Stack Patterns by Variant
- Use `golang.org/x/exp/cmd/apidiff` (or its library form) to diff the previous published version's exported API against the working tree.
- Feed the incompatible/compatible/no-change verdict into `Masterminds/semver/v3`'s constraint logic to check the proposed version bump is *at least* as large as required.
- Because it does real static analysis, this is the tier worth spending your interview-story energy on.
- Do not attempt Go-side AST parsing of Java/Python source.
- Require a manifest field (e.g. `breaking: true` in a `package.yaml`/`modular.yaml` published alongside the artifact) and a lightweight commit-message/changelog-fragment convention check (Conventional Commits `!`/`BREAKING CHANGE:` footer) as the enforcement signal instead.
- Be explicit in generated docs/READMEs that this tier is policy-enforced, not compiler-verified — that asymmetry is itself a good talking point about staged, pragmatic paved-road investment (mirrors the real Modular framework's own staged-expansion history in the README source material).
- Use `kind` or `k3d` to spin up a throwaway local cluster and `helm install` the generated chart against it in a demo script — but keep this entirely out of the Go module (shell scripts / Makefile targets), since importing `k8s.io/client-go` into the CLI itself would violate the "not a deployment orchestrator" non-goal.
- Layer a filesystem override: check an on-disk `~/.modular/templates/<lang>` directory first, fall back to the `embed.FS`-baked defaults. This keeps the "local filesystem template store" framing from PROJECT.md accurate (it's not *only* compiled-in) while keeping the zero-dependency single-binary default.
## Version Compatibility
| Package A | Compatible With | Notes |
|-----------|-----------------|-------|
| oras.land/oras-go/v2@v2.6.2 | Go 1.25 / 1.26 | `main`/`v2` branches officially support only the two latest Go minor versions; do not target Go 1.24 or older with this dependency. |
| go.opentelemetry.io/otel@v1.44.0 | Go 1.25.0+ | go.mod floor is exactly 1.25.0 — this is the binding constraint that sets this project's Go floor. |
| github.com/prometheus/client_golang@v1.24.0 | Go 1.25 / 1.26 only | v1.24.0 (2026-07-20) explicitly dropped support for <1.22 and now only supports the two latest Go releases — pin your CI matrix to 1.25.x and 1.26.x, drop any 1.23/1.24 CI lanes. |
| github.com/testcontainers/testcontainers-go@v0.43.0 | Go 1.25.0+ | Also requires a working Docker daemon in CI (GitHub Actions `ubuntu-latest` runners have this by default); no extra setup needed beyond that. |
| golangci-lint v2.12.2 | Any Go the linter itself targets via `run.go` config key | The linter binary and the Go version it *lints* are decoupled — just make sure `.golangci.yml`'s `run.go: '1.25'` matches your actual module floor so vet/staticcheck rules apply correctly. |
| modernc.org/sqlite | Go 1.21+ (no unusually high floor) | The one dependency in this stack with a notably *lower* Go floor than the rest — not a blocker, just worth knowing it won't force your Go version up on its own. |
## Sources
- pkg.go.dev/github.com/spf13/cobra, pkg.go.dev/github.com/spf13/viper, cobra.dev — version/date verification (HIGH confidence, official module registry)
- pkg.go.dev/oras.land/oras-go/v2, github.com/oras-project/oras-go — version/date + Go-support-window verification (HIGH confidence)
- github.com/project-zot (zot), distribution.github.io/distribution, github.com/distribution/distribution/releases, hub.docker.com/_/registry — registry comparison + `registry:3.1.1` version (MEDIUM-HIGH: zot ecosystem-analysis blog posts corroborated by CNCF project page and zot's own OCI-native design claims; distribution/registry data is HIGH via official GitHub releases)
- pkg.go.dev/go.opentelemetry.io/otel, github.com/open-telemetry/opentelemetry-go/releases, opentelemetry.io/docs/languages/go — v1.44.0 version + traces/metrics-stable/logs-beta status (HIGH confidence, official OTel docs)
- pkg.go.dev/github.com/prometheus/client_golang, github.com/prometheus/client_golang — v1.24.0 version + Go-support-window change (HIGH confidence, official changelog)
- github.com/golangci/golangci-lint (CHANGELOG.md, .golangci.reference.yml), golangci-lint.run/docs — v2.12.2 + v2 config schema (HIGH confidence, official docs/changelog)
- go.dev/doc/devel/release, go.dev/doc/go1.25, go.dev/doc/go1.26 — Go 1.25/1.26 release dates and current patch levels (HIGH confidence, official Go release history)
- pkg.go.dev/github.com/Masterminds/semver/v3, github.com/Masterminds/semver — v3.5.0 version (HIGH confidence)
- go.googlesource.com/exp/+/master/apidiff, pkg.go.dev/golang.org/x/exp/cmd/apidiff — apidiff tool semantics and scope (HIGH confidence, official Google Go docs; MEDIUM on "no Java/Python equivalent" framing, which is an architectural judgment call, not a documented fact)
- pkg.go.dev/modernc.org/sqlite ecosystem pages (gosqlite.com/docs, pkg.go.dev/gosqlite.org/fts) — FTS5/JSON1 compiled in by default (MEDIUM-HIGH: corroborated by multiple independent third-party sources describing the same modernc build-tag behavior, no single canonical modernc README quote captured verbatim)
- github.com/blevesearch/bleve releases, blevesearch.com — v2.6.0 version (HIGH confidence)
- github.com/testcontainers/testcontainers-go/releases, pkg.go.dev — v0.43.0 version + Go 1.25 floor (HIGH confidence)
- Community comparison articles (adaptive-enforcement-lab.com CLI framework matrix, dev.to/devgenius Cobra-vs-urfave pieces) — Cobra vs urfave/cli vs Kong positioning (MEDIUM confidence: consistent across multiple independent sources but not an official benchmark; used only to confirm the ecosystem-fit rationale, not to override the downstream consumer's explicit Cobra preference)
- OTel Collector + Prometheus + Grafana docker-compose reference architecture (oneuptime.com, github.com/samzhu/grafana-otel-lgtm-stack, evoila.com) — local observability stack wiring pattern (MEDIUM confidence: multiple independent 2026-dated blog sources converge on the same `otlp receiver -> prometheus exporter -> Prometheus scrape -> Grafana` shape)
<!-- GSD:stack-end -->

<!-- GSD:conventions-start source:CONVENTIONS.md -->
## Conventions

Conventions not yet established. Will populate as patterns emerge during development.
<!-- GSD:conventions-end -->

<!-- GSD:architecture-start source:ARCHITECTURE.md -->
## Architecture

Architecture not yet mapped. Follow existing patterns found in the codebase.
<!-- GSD:architecture-end -->

<!-- GSD:skills-start source:skills/ -->
## Project Skills

No project skills found. Add skills to any of: `.claude/skills/`, `.agents/skills/`, `.cursor/skills/`, `.github/skills/`, or `.codex/skills/` with a `SKILL.md` index file.
<!-- GSD:skills-end -->

<!-- GSD:workflow-start source:GSD defaults -->
## GSD Workflow Enforcement

Before using Edit, Write, or other file-changing tools, start work through a GSD command so planning artifacts and execution context stay in sync.

Use these entry points:
- `/gsd-quick` for small fixes, doc updates, and ad-hoc tasks
- `/gsd-debug` for investigation and bug fixing
- `/gsd-execute-phase` for planned phase work

Do not make direct repo edits outside a GSD workflow unless the user explicitly asks to bypass it.
<!-- GSD:workflow-end -->



<!-- GSD:profile-start -->
## Developer Profile

> Profile not yet configured. Run `/gsd-profile-user` to generate your developer profile.
> This section is managed by `generate-claude-profile` -- do not edit manually.
<!-- GSD:profile-end -->
