# Feature Research

**Domain:** Multi-language internal developer platform CLI (Modular/Nerv-style paved road: scaffolding + semver-gated publish + dependency graph + project search)
**Researched:** 2026-07-24
**Confidence:** MEDIUM-HIGH (ecosystem/tooling claims verified against official docs/GitHub for Backstage, Copier/Cookiecutter/Yeoman, cargo-semver-checks, buf, ORAS; Tableau Modular specifics are HIGH confidence only where sourced from the project's own retrospective blueprint — flagged LOW where reconstructed)

## Feature Landscape

This domain has two reference classes worth triangulating against:

1. **Real-world IDPs (Backstage, and the historical Tableau Modular/Nerv system this project reconstructs)** — define what "paved road" table stakes look like at organizational scale.
2. **Scaffolding-tool ecosystem (Cookiecutter, Copier, Yeoman) + semver/compat tooling (cargo-semver-checks, buf breaking) + artifact tooling (ORAS/OCI)** — define the concrete, currently-idiomatic mechanics for each pillar (templating, breaking-change detection, artifact storage) that a laptop-scale rebuild should adopt instead of reinventing.

### Table Stakes (Users Expect These)

Features the demo cannot skip — Modular/Nerv's core value prop was exactly these four capabilities (generate, publish-gate, deps graph, search), so for this project "table stakes" and "MVP" are nearly the same set.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| `generate` scaffolds Go/Java/Python from local FS templates | This is the entire premise of a "paved road" generator (Backstage Software Templates, Cookiecutter/Copier all center on this) [HIGH] | MEDIUM | Use Go `text/template` or embed a Jinja-like engine; parameterize name/team/language exactly as Modular's original `modular generate --lang <x> --name <svc>` did [HIGH, sourced from README.md blueprint] |
| Generated project includes Dockerfile | Every scaffolding tool with a "service" archetype emits this; Modular did too [HIGH] | LOW | Static per-language template, minor param substitution |
| Generated project includes CI stub (GitHub Actions) | Modular's original design point was "CI is already wired to platform standards" on generate — an ungenerated CI pipeline defeats the paved-road pitch [HIGH] | LOW-MEDIUM | One workflow YAML per language (build/test/lint); GHA is the natural GitLab-runner/TeamCity substitute per PROJECT.md constraints |
| Generated project includes K8s manifests + Helm chart | Modular shipped this from day one; Backstage templates commonly scaffold a `catalog-info.yaml` + deployment manifests too [HIGH/MEDIUM] | MEDIUM | Helm chart skeleton (Chart.yaml, values.yaml, templates/) parameterized by service name; no live cluster apply required for the demo |
| Generated project includes OTel instrumentation stub | Modular's observability-by-default pillar (Splunk/New Relic originally); OTel→Prometheus/Grafana is the modern equivalent per PROJECT.md constraints [HIGH intent, MEDIUM impl specifics] | MEDIUM | Per-language OTel SDK bootstrap snippet (traces/metrics init) wired to a local OTel collector config; doesn't need real backend integration, just wiring |
| `publish` blocks breaking changes without a major bump | This is the single most load-bearing claim in the entire Modular case study — the linter+CI-gate pattern is directly precedented by `cargo-semver-checks` (`cargo semver-checks && cargo publish`) and `buf breaking` (`buf breaking --against`) [HIGH] | HIGH | Needs an API-surface diff mechanism per language (see Feature Dependencies below) — this is the hardest engineering problem in the project and should be scoped deliberately per language, not built generically first |
| `deps --graph <package>` shows consumer blast radius | Direct 1:1 mapping to Modular's original dependency-tooling pillar and its stated purpose ("consumer blast radius knowable before release") [HIGH] | MEDIUM | Requires the dependency graph data model (publisher/consumers) to exist before this command has anything to render — depends on `publish` recording consumer relationships |
| `search <query>` returns registered project + build config refs | Direct 1:1 mapping to Modular's original search index pillar [HIGH] | LOW-MEDIUM | Simplest of the four pillars once the service/project registry exists; a FS-backed or SQLite-backed index with substring/metadata search is sufficient — no need for full-text/semantic search |
| Local-first, single-binary CLI (no external services required to run the demo) | PROJECT.md explicitly constrains this to "laptop-first, no cloud dependency for core path" — a demo that requires spinning up cloud infra fails the stated goal | LOW-MEDIUM | Local registry = filesystem + embedded DB (SQLite) or a local OCI registry (e.g., `zot` or `registry:2` container) rather than a hosted registry |
| Parameterized template variables (name, team, language) | Baseline expectation of every scaffolding tool surveyed (Cookiecutter's `cookiecutter.json`, Copier's `copier.yml`, Yeoman prompts) [HIGH] | LOW | Straightforward prompt/flag collection; Copier's typed-YAML-question pattern is a good reference even if not adopting Copier itself |
| Idempotent generation (rerunning `generate` with same inputs doesn't corrupt output) | Standard scaffolding tool hygiene — Cookiecutter/Copier both treat this as baseline; failure here erodes demo trust | LOW | Fail/warn on existing target dir rather than silently overwrite |

### Differentiators (Competitive Advantage / Portfolio Value)

These aren't required for a functioning demo, but they are where this project can stand out as a portfolio piece vs. a bare scaffolding tool clone.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Real API-diff-based semver enforcement per language (not just a version-string check) | Most portfolio scaffolding clones fake semver gating with a trivial version-bump check. Doing real breaking-change detection (Go: `go/types`-based export diffing akin to `apidiff`/`golangci-lint`'s `unused`-adjacent tooling; Java: bytecode/API surface diff akin to `japicmp`; Python: signature/attribute diffing) demonstrates the actual hard engineering Modular/Nerv had to solve [precedent: `cargo-semver-checks`, `buf breaking` both do real structural diffing, not string comparison] | HIGH | This is the single best "interview-defensible" differentiator — explicitly call out in the demo narrative that the gate does structural API diffing, not string matching. Consider staging: Go first (richest tooling ecosystem: `golang.org/x/exp/apidiff`), then Python (AST-based), then Java (reflection/bytecode) if time allows |
| OCI-backed local registry for packages (via ORAS or a local `zot`/`distribution` registry) | Directly reconstructs the "AWS/Artifactory-backed" registry pillar with a modern, laptop-runnable substitute; ORAS is the de facto tool for pushing arbitrary (non-image) artifacts to OCI registries [HIGH, sourced from oras.land docs] | MEDIUM | Store package tarballs/metadata as OCI artifacts with custom media types (`application/vnd.nerv.pkg.v1+json`), same pattern ORAS demonstrates for SBOMs. Cleanly maps "OCI + local registry + FS index" constraint from PROJECT.md to a real, defensible architecture choice |
| Rich `deps --graph` visualization (ASCII tree or DOT/Graphviz export) | Backstage's catalog relies on a graph model but the visualization itself is a differentiator most bare scaffolders skip entirely | LOW-MEDIUM | A DOT export you can pipe to `dot -Tpng` is cheap and demo-worthy; ASCII tree is the zero-dependency fallback |
| `search` with build-config preview (return the actual CI/Helm/Dockerfile content, not just metadata pointers) | Modular's original search returned "language-specific build configuration and build system files" — most scaffolding tools' registries only return metadata, not artifact content [HIGH, sourced from README.md] | LOW-MEDIUM | Natural fit once the FS index stores refs to generated project files; a small value-add to implement given the registry already exists |
| Template versioning with a rollback/pointer mechanism (`active_version` per language) | Modular's blueprint flags this as an open question/reconstructed guess — implementing a clean `{language: active_version}` pointer with an explicit rollback command is both faithful to the original design intent and a nice "we solved the ambiguity" portfolio talking point [LOW confidence on original mechanism per README.md's own "Open Questions" section, but a reasonable and demo-worthy design regardless] | LOW-MEDIUM | Store as a simple JSON/SQLite record; `modular template rollback --lang go --version 1.2.0` |
| Audit trail / structured log of publish decisions (allowed/blocked + reason) | Modular's blueprint explicitly ties the linter+CI-gate combo to SOC2/audit compliance value ("a failed/blocked publish is a logged, reviewable event") [HIGH, sourced from README.md] | LOW | Append-only JSON lines log of publish attempts; cheap to build, strong narrative tie-back to the "why this mattered" part of the original case study |
| Multi-language build-system awareness in generated CI (different pipeline per language, not one generic template) | Modular's blueprint explicitly calls out that "a Kotlin gRPC service and a Rust CLI tool wouldn't get identical pipelines" as a sign of platform maturity [HIGH, sourced from README.md] | LOW-MEDIUM | Trivial to do well since scope is fixed at 3 languages — just don't copy-paste one generic GHA workflow across all three |
| Local Prometheus/Grafana demo dashboard wired to generated OTel stubs | Elevates "observability stubs" from static template files to an actually-running, visually demoable pipeline | MEDIUM | Docker Compose for local Prometheus+Grafana, pre-built dashboard JSON scraping the OTel collector — good demo payoff for moderate effort |

### Anti-Features (Commonly Requested, Often Problematic)

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Template *update*/sync mechanism (Copier-style `copier update` with three-way diffs into already-generated projects) | Copier's comparison table shows this is considered the modern best practice over Cookiecutter's "generate once" model, so it feels like the "correct" choice to copy | High complexity (migrations, diff/merge logic, answer-file tracking) for a feature Modular's own blueprint never mentions using operationally, and PROJECT.md's Out of Scope explicitly excludes "migration tooling" | Keep `generate` as one-shot (Cookiecutter model). If asked in an interview why: "the original system's differentiator was the publish/deps/search pillars, not template lifecycle management — scope discipline over feature parity with Copier" |
| Full org-wide catalog/service ownership UI (Backstage-style web portal) | Backstage is the most visible modern IDP reference and its catalog+plugin ecosystem is the "gold standard" people benchmark against | PROJECT.md explicitly scopes this as a single-operator CLI playground, not a hosted portal; building a web UI, auth, and plugin architecture is a multi-quarter team effort at Backstage's own scale, not a laptop demo | CLI output (tables/JSON) is the UI. If a visual artifact is wanted, a static-generated HTML report from `search`/`deps --graph` output is far cheaper than a live portal |
| Real cloud-backed registry (S3, Artifactory, real container registry service) | "More realistic," and it's literally what the original Modular used (AWS-backed template store, Artifactory/PyPI dependency tooling) | Directly violates the "laptop-first, no cloud dependency for core path" constraint; adds credential/account-management surface area with zero demo value add | Local OCI registry (ORAS + `zot`/`registry:2` container) + local filesystem index — functionally equivalent for the demo, zero external accounts |
| Supporting the full historical language surface (C++, gRPC-as-primary, Groovy, Kotlin, Scala, Ruby, Rust, TypeScript/JS) | The original Modular grew to support all of these over 7 years, and it's tempting to "complete the reconstruction" | PROJECT.md explicitly scopes v1 to Go/Java/Python; each additional language multiplies the hardest problem in the project (real API-diff-based semver enforcement) by a full implementation, with diminishing portfolio value beyond "proved multi-language" | Ship Go/Java/Python well; note in the demo narrative that Modular's real strength was *staged* expansion, and v1 deliberately proves the same expansion capability on 3 languages rather than shallowly stubbing 8 |
| Fully general breaking-change detection across arbitrary API shapes (generics, macros, dynamic language edge cases) day one | `cargo-semver-checks`'s own docs candidly list categories of semver breaks it still doesn't catch (generics/lifetimes, feature-gated APIs) — it's tempting to aim for "complete" coverage to seem more rigorous | Chasing completeness on API-diff correctness is an open research problem even for mature single-language tools; attempting it across 3 languages simultaneously will consume the whole project timeline before `deps`/`search` ever get built | Scope the diff check to common breaking patterns per language (removed/renamed public symbol, changed signature/type, removed field) — same "catch the common 80%, document the known gaps" posture `cargo-semver-checks` itself takes publicly |
| Mandatory adoption/enforcement tooling (org policy engine, pre-commit hook rollout, compliance dashboards) | Feels like it "completes" the platform story since the original had SOC2/audit ties | PROJECT.md's Out of Scope explicitly excludes this ("single-operator playground," "opt-in... no adoption mandate tooling") | The `publish` gate itself *is* the enforcement mechanism for this project's one operator — no separate policy layer needed |
| Continuous deployment / GitOps runtime orchestration from generated K8s/Helm | Natural-feeling "next step" once Helm charts exist — why not also deploy them? | PROJECT.md Out of Scope explicitly excludes "runtime service mesh or continuous deployment orchestration"; Modular's own blueprint states Helm/K8s were "generated starting points only," with rollout strategy owned by consuming teams | Generated Helm chart is the deliverable; `helm install` against a local kind/minikube cluster is an optional manual demo step, not a CLI feature |
| Real-time/live-updating dependency graph (watch mode, webhooks on publish) | "Real-time everything" always sounds more impressive in a demo pitch | Adds infra complexity (event bus, watchers) with no operator other than the person running commands sequentially in one sitting — nothing to watch *for* | `deps --graph` recomputes on-demand from the current registry state each invocation; that's suffient for a single-operator, single-sitting demo |

## Feature Dependencies

```
generate (per language: Go/Java/Python templates + params)
    └──requires──> local FS template store (versioned dir per language)

publish (semver gate)
    └──requires──> generate having produced a project with a discoverable public API surface
    └──requires──> per-language API-diff engine (Go/Java/Python each need their own)
    └──requires──> OCI + local registry + FS package index (to store "last published version" as the diff baseline)

deps --graph <package>
    └──requires──> publish (each publish call must record publisher + consumer edges into the dependency graph)
    └──enhances──> publish (surfacing blast radius *before* a breaking publish is attempted is the whole point — ideally deps data feeds a warning inside publish itself, not just a standalone query)

search <query>
    └──requires──> generate (to have registered a project + build-config refs) 
    └──requires──> the same service/project registry that publish and deps write into (shared registry, not three separate stores)

OTel instrumentation stub ──enhances──> generate (adds template content, no new command)
CI/Helm/Dockerfile stubs ──enhances──> generate (adds template content, no new command)

Real API-diff engine (Go) ──unlocks──> Real API-diff engine (Java, Python)
    (design the diff *interface* generically — e.g., ExtractPublicSurface(lang, path) -> Surface, Diff(old, new) -> []BreakingChange —
     so the Go implementation isn't a one-off; Java/Python plug into the same publish-gate contract)

Template update/sync (anti-feature) ──conflicts──> one-shot generate model chosen for this project
Real cloud registry (anti-feature) ──conflicts──> laptop-first / OCI+local registry+FS index constraint
```

### Dependency Notes

- **`publish` requires a per-language API-diff engine**: this is the load-bearing dependency for the entire project. Unlike `generate`/`search`, which are mechanically straightforward (file templating, index lookup), the semver gate needs to actually understand each language's public API surface. Recommend building the diff-engine *interface* once (extract surface → diff → classify breaking/non-breaking) and implementing Go first (best tooling: `golang.org/x/exp/apidiff`, or a hand-rolled `go/types` walk), proving the interface, then Java and Python.
- **`deps --graph` requires `publish` to exist first and to write graph edges as a side effect**: you cannot demo blast radius without publish history to graph. This is why the demo flow in PROJECT.md is ordered generate → publish → deps → search, and the phase/roadmap ordering should mirror it exactly.
- **`search` shares the registry with `publish`/`deps` rather than being a fourth independent data store**: Modular's own data model (README.md) shows one service registry + one project search index feeding off the same publish events — building three siloed stores would be both extra work and a divergence from the reconstructed design.
- **OTel/CI/Helm/Dockerfile stubs enhance `generate` but add zero new commands**: these are template content, not features with their own CLI surface — track them as "generate template completeness" work, not separate roadmap phases.
- **Anti-feature conflicts**: template-update/sync and real cloud registry are flagged as directly conflicting with explicit PROJECT.md constraints (one-shot generate model; laptop-first/local-only backend) — including them would require walking back a stated project decision, not just "nice to have later."

## MVP Definition

### Launch With (v1)

Minimum viable product — mirrors the four demo-flow commands in PROJECT.md exactly; nothing here is optional without breaking the stated demo.

- [ ] `generate --lang <go|java|python> --name <svc> --team <team>` from local FS templates — essential: this is the entry point to every other pillar
- [ ] Generated project includes Dockerfile + GHA CI stub + K8s/Helm skeleton + OTel instrumentation stub — essential: this is what makes `generate` a "paved road" generator rather than a bare stub
- [ ] `publish` with real per-language API-diff-based semver gate (start with Go; Java/Python can follow within v1 scope per PROJECT.md's "Go, Java, or Python" requirement) — essential: this is the single most differentiating and most load-bearing claim of the whole project
- [ ] `deps --graph <package>` showing consumer blast radius from recorded publish events — essential: explicitly named in PROJECT.md Active requirements
- [ ] `search <query>` returning registered project metadata + language-specific build config refs — essential: explicitly named in PROJECT.md Active requirements
- [ ] OCI + local registry + filesystem package index backing `publish`/`deps`/`search` — essential: the shared data backbone all three later pillars depend on
- [ ] TDD discipline (failing test before implementation) on every vertical slice — essential: explicitly stated as non-negotiable process constraint, not an optional feature

### Add After Validation (v1.x)

Features to add once the four core pillars work end-to-end in one sitting.

- [ ] DOT/Graphviz export for `deps --graph` — trigger: once the ASCII/table output of the graph command works, visual export is a cheap, high-payoff follow-on
- [ ] `search` returning actual build-config file content (not just refs) — trigger: once the registry stores refs, upgrading to content retrieval is incremental
- [ ] Template version pointer + rollback command — trigger: once `generate` is stable across all 3 languages, versioning the template store itself becomes worth the investment
- [ ] Local Prometheus/Grafana demo dashboard wired to generated OTel stubs — trigger: once OTel stubs are proven to compile/run in generated projects, wiring a real local dashboard is a strong demo-day addition
- [ ] Publish audit-trail log (structured JSONL of allow/block decisions) — trigger: once `publish` gate logic is stable, this is a small addition with strong "SOC2/audit" narrative payoff

### Future Consideration (v2+)

Features to defer — genuinely valuable in a real IDP but out of proportion to a laptop portfolio demo.

- [ ] Additional languages beyond Go/Java/Python (Kotlin, Rust, TypeScript, etc.) — defer: v1 scope is deliberately fixed per PROJECT.md; each new language re-triggers the hardest problem in the project (a new API-diff engine)
- [ ] Backstage-style catalog web UI / plugin architecture — defer: out of scope per PROJECT.md ("single-operator playground"); would turn a CLI portfolio piece into a multi-quarter platform-team effort
- [ ] Template update/sync (Copier-style migrations into already-generated projects) — defer: explicitly conflicts with the one-shot generate model chosen for this project

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| `generate` (Go/Java/Python, local FS templates) | HIGH | MEDIUM | P1 |
| Generated CI/Dockerfile/Helm/OTel stubs | HIGH | LOW-MEDIUM | P1 |
| `publish` semver gate w/ real API diff (Go first) | HIGH | HIGH | P1 |
| `publish` semver gate extended to Java + Python | HIGH | HIGH | P1 |
| OCI + local registry + FS package index | HIGH | MEDIUM | P1 |
| `deps --graph` blast radius | HIGH | MEDIUM | P1 |
| `search` metadata + build-config refs | HIGH | LOW-MEDIUM | P1 |
| DOT/Graphviz export for deps graph | MEDIUM | LOW | P2 |
| `search` returning actual file content | MEDIUM | LOW-MEDIUM | P2 |
| Publish audit-trail log | MEDIUM | LOW | P2 |
| Template version pointer + rollback | LOW-MEDIUM | LOW-MEDIUM | P2 |
| Local Prometheus/Grafana live dashboard | MEDIUM | MEDIUM | P2 |
| Additional languages beyond v1 scope | LOW (for this portfolio goal) | HIGH | P3 |
| Backstage-style catalog web UI | LOW (for this portfolio goal) | HIGH | P3 |
| Template update/sync (Copier-style) | LOW (conflicts with chosen design) | HIGH | P3 |

**Priority key:**
- P1: Must have for launch (demo fails without it)
- P2: Should have, add when possible (strengthens the portfolio narrative)
- P3: Nice to have / explicitly deferred, future consideration only

## Competitor Feature Analysis

| Feature | Backstage (Software Templates + Catalog) | Cookiecutter/Copier/Yeoman | This Project (Nerv-ecosystem) |
|---------|--------------------------------------------|-----------------------------|-------------------------------|
| Project scaffolding | YAML-defined templates, form-driven wizard UI, auto-registers into catalog on generate | Cookiecutter: one-shot Jinja render from JSON config; Copier: YAML-typed questions + update/migration support; Yeoman: JS generator, most flexible/most code required | CLI-driven `generate --lang <go\|java\|python>`, local FS templates, parameterized name/team/language, one-shot (Cookiecutter model), no update/sync |
| Semver / breaking-change enforcement | Not a built-in concern (catalog tracks metadata, doesn't gate publishes) | Not applicable — these are scaffolding tools, not package registries | Real per-language API-diff-based gate at `publish`, modeled on `cargo-semver-checks`/`buf breaking`'s structural-diff approach rather than string version comparison — this is the project's core differentiator vs. both reference classes |
| Dependency / consumer tracking | Catalog models relations between entities (APIs, components, systems) but not semver-aware publish gating | Not applicable | `deps --graph <package>` shows consumer blast radius, fed by publish-time edge recording — directly reconstructs Modular's own dependency-tooling pillar |
| Central registry/catalog search | Full Software Catalog with searchable UI, ownership, entity relationships | Not applicable (no registry concept) | `search <query>` over a local FS/SQLite index returning metadata + build-config refs — scoped-down, CLI-only equivalent of Backstage's catalog search |
| Artifact storage backend | Pluggable via integrations (GitHub/GitLab, cloud storage) | N/A | OCI + local registry (ORAS-style generic artifact push/pull) + filesystem index — laptop-runnable stand-in for Artifactory/S3 |
| Deployment scope | Full hosted platform (Node.js backend + React frontend + plugin ecosystem), meant for many-team orgs | Local CLI tool, single developer, no server | Local CLI, single operator, no server, one-sitting demo — deliberately narrower than Backstage, deliberately more opinionated/enforcement-heavy than Cookiecutter/Copier/Yeoman |

## Sources

- Backstage documentation: [What is Backstage?](https://backstage.io/docs/overview/what-is-backstage), [Software Templates overview](https://github.com/backstage/backstage/blob/master/docs/features/software-templates/index.md), [Technical overview](https://github.com/backstage/backstage/blob/a723b8a1/docs/overview/technical-overview.md), [Red Hat: Build your first Software Template](https://developers.redhat.com/articles/2025/08/12/build-your-first-software-template-backstage) — MEDIUM-HIGH confidence, official/vendor docs
- Copier vs. Cookiecutter vs. Yeoman comparisons: [Copier official comparison](https://copier.readthedocs.io/en/latest/comparisons/), [reqhiem.dev feature table](https://reqhiem.dev/blog/copier-vs-cookiecutter-template-management), [RecallStack: Yeoman and Cookiecutter are dead](https://recallstack.gitlab.io/en/2020/04/18/yeoman-and-cookiecutter-are-dead-long-live-copier/) — HIGH confidence for Copier's own docs, MEDIUM for third-party blog framing
- `cargo-semver-checks`: [GitHub README](https://github.com/obi1kenobi/cargo-semver-checks), [FOSDEM 2024 talk writeup](https://predr.ag/blog/semver-in-rust-tooling-breakage-and-edge-cases/) — HIGH confidence, official repo + conference talk from maintainer-adjacent source
- `buf breaking`: [Buf GitHub](https://github.com/bufbuild/buf), [Buf breaking-change docs](https://buf.build/docs/breaking/), [AuthZed case study](https://authzed.com/blog/buf) — HIGH confidence, official docs + real adopter case study
- ORAS / OCI artifact storage: [oras.land docs](https://oras.land/docs/), [oras push command reference](https://oras.land/docs/1.1/commands/oras_push), [Loft Labs generic artifact stores writeup](https://www.vcluster.com/blog/leveraging-generic-artifact-stores-with-oci-images-and-oras) — HIGH confidence, official project docs
- Tableau Modular/Nerv original design: `nerv-ecosystem/README.md` (this project's own retrospective blueprint) — HIGH confidence for stated facts/quotes from the document itself; explicitly flagged LOW confidence within that document for reconstructed/uncertain details (rollback mechanism internals, exact CLI syntax, build-vs-buy origin) — treated as LOW here too where cited
- Project scope and constraints: `nerv-ecosystem/.planning/PROJECT.md` — HIGH confidence, authoritative for this project's own decisions

---
*Feature research for: multi-language developer-platform CLI (Modular/Nerv-style paved road)*
*Researched: 2026-07-24*
