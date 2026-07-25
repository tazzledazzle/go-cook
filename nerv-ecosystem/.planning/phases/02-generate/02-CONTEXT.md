# Phase 2: Generate - Context

**Gathered:** 2026-07-25
**Status:** Ready for planning
**Source:** `/gsd-discuss-phase 2 --auto` (single-pass autonomous discuss)

[--auto] Selected all gray areas: Output path & CLI flags, Template storage & engine, Generated project depth, Provenance manifest, Overwrite & path-safety behavior, Store registration & schema extension, Public-API surface scaffolding.

<domain>
## Phase Boundary

Deliver `modular generate --lang go|java|python --name <svc> --team <team>` that scaffolds a paved-road service from local templates (Dockerfile, language-specific GHA CI invoking real toolchain commands, K8s manifests, Helm chart, OTel→Prometheus exporter stub), writes a provenance manifest, refuses unsafe/overwrite paths, and registers the project in the Phase 1 SQLite store — covering GEN-01…GEN-09 only. No publish, deps graph, or search command work.

</domain>

<decisions>
## Implementation Decisions

### Output path & CLI flags
- **D-01:** Flags-only UX (no interactive prompts). Required: `--lang`, `--name`, `--team`. Optional: `--out` (default `./<name>` under cwd), `--templates-dir` (override template root), reuse global `--store-path` / `MODULAR_HOME` from Phase 1.
- **D-02:** Default output directory is `./<name>` relative to the process cwd. `--out` may point elsewhere; after `filepath.Clean` + validation it must not escape intended roots (GEN-07).
- [auto] Output path & CLI flags — Q: "Flags vs prompts?" → Selected: "Flags-only, CI-friendly (recommended)" (matches `status` style)

### Template storage & engine
- **D-03:** Primary templates live in-repo under `templates/<lang>/…` and ship inside the binary via `embed.FS` (single-binary / laptop-first).
- **D-04:** Override search order: `--templates-dir` → `$MODULAR_HOME/templates` → embedded defaults. Override dirs are filesystem only (not re-embedded).
- **D-05:** Engine is Go `text/template` (+ `embed.FS`). Do not adopt Cookiecutter/Copier/Yeoman as runtime (locked stack "What NOT to Use").
- [auto] Template storage — Q: "Where do templates live?" → Selected: "embed.FS + optional FS override (recommended)"

### Generated project depth
- **D-06:** Each language gets a thin but **compiling** service skeleton: one metrics-exposed HTTP (or equivalent) entrypoint so the OTel→Prometheus stub is real wiring, not a dead comment block. Not a full business domain app.
- **D-07:** Per-language CI must shell out to real toolchain commands (`go test` / Gradle test / `pytest`) — GEN-04. Distinct workflow files per language (no copy-paste identical pipeline).
- **D-08:** Helm + raw K8s manifests are static YAML from templates (no Helm SDK / client-go in the CLI module).
- [auto] Generated depth — Q: "Stub vs runnable?" → Selected: "Thin compiling service with metrics endpoint (recommended)"

### Provenance manifest
- **D-09:** Every successful generate writes `.nerv-manifest.json` at the project root with at least: template name, template version/id, language, name, team, params, timestamp (UTC), generator CLI version if available.
- **D-10:** Manifest is written as part of the same successful generate transaction as the file tree (no orphan trees without manifest).
- [auto] Provenance — Q: "Filename/format?" → Selected: ".nerv-manifest.json at project root (recommended per research)"

### Overwrite & path-safety behavior
- **D-11:** If the target directory exists and is non-empty → fail with non-zero exit and a clear stderr message (GEN-06). Do not merge or overwrite.
- **D-12:** Validate `--name`, `--team`, and resolved output path before any filesystem write: reject `..`, absolute escapes when not intended, symlink escape of the target parent (GEN-07). Mirror Phase 1 failure style: wrapped error, exit 1, no usage dump on operational failure.
- [auto] Safety — Q: "Overwrite policy?" → Selected: "Hard fail on non-empty target (recommended)"

### Store registration & schema extension
- **D-13:** After successful on-disk generation, register the project via `internal/store` (GEN-08). `cmd/` must not import `database/sql`.
- **D-14:** Extend schema as needed for build-config refs (Phase 1 `projects` has name/team/language/path only). Prefer a forward migration adding nullable `build_config_refs` (JSON text or delimited paths) rather than a second store. Keep `versions`/`edges` out of Phase 2.
- **D-15:** Registration and FTS sync must happen in the same DB write path that already has INSERT triggers (Phase 1).
- [auto] Store — Q: "When to register?" → Selected: "After successful tree+manifest write (recommended)"

### Public-API surface scaffolding
- **D-16:** Go templates rely on exported identifiers (capitalized) as the public surface — no extra file required beyond idiomatic package layout.
- **D-17:** Python templates must declare an explicit `__all__` in the package `__init__.py` (GEN-09 / future griffe upgrade hook).
- **D-18:** Java templates declare a clear public module/package boundary (e.g. dedicated `api` package or module-info / documented public package) as the Phase 3 policy-gate hook.
- [auto] Public API hooks — Q: "How explicit?" → Selected: "Scaffold language-appropriate public surface from day one (recommended)"

### Claude's Discretion
- Exact template file tree layout under `templates/go|java|python/`
- Exact JSON schema fields beyond the minimum in D-09
- Whether `build_config_refs` stores relative paths to `.github/workflows/*.yml`, `Dockerfile`, `Chart.yaml`, etc.
- Package naming inside generated projects (as long as they compile and meet GEN-02…04)
- Whether generate prints a short success summary (path + language) — prefer yes, matching `status` clarity

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 2 goal, success criteria, Mode: mvp
- `.planning/REQUIREMENTS.md` — GEN-01…GEN-09 (v1), GEN-V2-* deferred
- `.planning/PROJECT.md` — core value, constraints, out of scope

### Prior research & Phase 1 contracts
- `.planning/research/SUMMARY.md` — conflict resolutions (ocilayout, two-tier semver later, direct Prometheus exporter for stubs)
- `.planning/research/STACK.md` — Cobra, text/template+embed.FS, OTel Prometheus exporter, anti-Cookiecutter
- `.planning/research/ARCHITECTURE.md` — `internal/generate` + `internal/templates` package boundaries; store as sole DB owner
- `.planning/research/FEATURES.md` — generate table stakes / anti-features
- `.planning/research/PITFALLS.md` — template provenance, path-traversal CVEs in scaffolders, bespoke-toolchain trap
- `.planning/phases/01-platform-foundation/01-SKELETON.md` — locked module path, store location, cmd/ vs internal/store, schema incrementality
- `.planning/phases/01-platform-foundation/01-VERIFICATION.md` — Phase 1 must-haves that Generate builds on
- `README.md` — Modular blueprint (generate path, paved-road stubs)

### Existing code to extend
- `cmd/root.go` — Cobra root, global `--store-path`
- `internal/store/store.go` — `Open`/`Close`; add project insert API here
- `internal/store/migrations/0001_init.sql` — existing `projects` + FTS5 triggers

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd.NewRootCommand` / global `--store-path` + `MODULAR_HOME` — generate attaches as a sibling of `status`
- `internal/store.Open` — generate opens the same registry; need new exported insert/query helpers (no raw SQL in cmd)
- Migration runner (`embed.FS` + `schema_migrations`) — pattern to copy for `0002_*.sql` if build_config_refs is added
- Failure style from `status` — exit 1, wrapped errors, no Usage dump on operational failure

### Established Patterns
- Cobra-only `cmd/`; domain logic in `internal/*`
- Table-driven tests + `t.Parallel()` + real temp dirs + `go test -race`
- TDD RED commit `test(02-XX):` before GREEN `feat(02-XX):`

### Integration Points
- New packages: `internal/generate`, `internal/templates` (per architecture research)
- New command: `cmd/generate.go` (+ tests)
- Store: INSERT into `projects` (and possibly migration for build_config_refs) so Phase 5 search can find generates later
- FTS5 triggers already sync name/team/language on INSERT

</code_context>

<specifics>
## Specific Ideas

- Binary remains `modular`; subcommand `generate` matches PROJECT.md demo flow.
- Prefer demonstrating OTel stub by exposing a Prometheus scrape endpoint (or documented metrics route) consistent with "direct Prometheus exporter" resolution in research SUMMARY.
- Java build system in templates: Gradle (research/stack mentions Gradle variants; keep one clear choice — Gradle — unless planner finds a strong reason otherwise).

</specifics>

<deferred>
## Deferred Ideas

- Template active-version pointer + rollback (`GEN-V2-01`) — Phase later / v2
- Copier-style template update/sync (`GEN-V2-02`) — out of scope / conflicts with one-shot generate
- zot daemon, publish/semver, deps graph, search commands — Phases 3–5
- OTel Collector + Grafana compose stack — OBS-V2 / post-MVP
- Fixing Phase 1 review WR-01/WR-02 (DSN `?` escape, TOCTOU chmod) — optional hygiene, not GEN scope unless it blocks generate tests

None — discussion stayed within phase scope (auto pass).

</deferred>

---

*Phase: 02-generate*
*Context gathered: 2026-07-25 via --auto*
