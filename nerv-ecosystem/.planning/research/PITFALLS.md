# Pitfalls Research

**Domain:** Multi-language developer-platform CLI (Modular/Nerv-style paved road) — scaffolding generator, semver-gated publish, dependency graph, local OCI registry, project search, OTel-instrumented templates
**Researched:** 2026-07-24
**Confidence:** MEDIUM-HIGH (critical pitfalls verified against official tool docs, GitHub issues/CVEs, and multiple independent sources; a few architectural judgment calls are MEDIUM/LOW and flagged inline)

## Critical Pitfalls

### Pitfall 1: Templates Are Treated as One-Shot Instead of a Living Contract ("Template Drift")

**What goes wrong:**
`generate` stamps out a project once, then the template and the generated project become permanently disconnected files on disk. Six months later the template gains a new CI step, OTel SDK version bump, or Helm value, but every project already generated from it silently rots out of compliance. Nobody notices until a security/compliance sweep or an incident reveals half the fleet is on stale scaffolding.

**Why it happens:**
Cookiecutter-style generators (and any hand-rolled `text/template` walker copying files into place) have no memory of "which template version, at which commit, produced this project." Without that provenance, there is no mechanical way to compute a diff between "what the template looks like now" and "what this project was generated with," so updates become a from-scratch manual re-read of every generated file.

**How to avoid:**
- Write a manifest into every generated project (`.nerv-manifest.json`/`.nerv/manifest.yaml`) recording: template name, template version/commit hash, language, parameters used, generation timestamp. This is the single most important artifact the generator produces — treat it as important as the generated source itself.
- Keep template logic thin. Push anything non-trivial (OTel bootstrap code, CI logic) into small, independently-versioned snippets/libraries referenced by the template rather than duplicated inline in every language's Jinja/text-template tree — this is the pattern the ecosystem converged on after learning `cruft update` conflicts are painful because there's no common ancestor for a 3-way merge.
- Even if `generate --update`/drift-detection is out of scope for v1, design the manifest format now so it is addable later without breaking existing generated projects. This is explicitly one of the "not a goal, but don't paint yourself into a corner" items.

**Warning signs:**
- Generated projects have zero record of what template/version produced them.
- Two projects generated a week apart from an "identical" template have divergent CI YAML with no way to tell why.

**Phase to address:**
`generate` slice — build the manifest-writing step as part of the first vertical slice, test-first (assert manifest exists + contains expected fields after `generate` runs).

---

### Pitfall 2: Semver/API-Diff Gate Trusts a Structural Compatibility Heuristic as Ground Truth

**What goes wrong:**
The publish gate treats "no incompatible diff reported" as "safe to publish" and "incompatible diff reported" as "must bump major." Both directions are wrong often enough to matter: the diff tool can (a) miss real breaks it structurally cannot see (behavioral changes, removed runtime guarantees, changed error semantics), and (b) flag changes that are technically detectable but never break any actual consumer (e.g., a changed exported constant nobody references, or embedding-related struct field additions).

**Why it happens:**
Structural API diffing (Go's `apidiff`, Java's `japicmp`/`revapi`, Python's `griffe check`) can only reason about the shape of the public surface, not program behavior. Go's own `apidiff` README states this explicitly: "No tool can detect behavioral changes... even a change that causes code to fail to compile may not be considered breaking by developers or their users." In practice this produces two failure modes in a homegrown gate: false negatives (bug fix that changes return semantics ships as patch, breaks a consumer) and false positives (noisy diff on a symbol nobody actually imports, engineers start ignoring the gate — the exact "too strict → users stop trusting it" trap the apidiff docs warn about).
For Python specifically, "public API" is not language-enforced — Griffe's own docs call this "subjective," and it defaults to "everything reachable and not underscore-prefixed is public" unless you define `__all__`, which will produce false-positive majors on generated Python projects that don't explicitly declare `__all__`.

**How to avoid:**
- Pick one diff strategy per language and match it to what the language actually enforces: Go → `golang.org/x/exp/apidiff` reasoning over exported package symbols; Java → `japicmp`/`revapi`-style bytecode/classfile comparison; Python → `griffe check` against an explicit `__all__`-defined public surface, not "everything not underscored."
- Require every generated project template to explicitly declare its public surface at generation time (Go: a documented top-level package convention; Java: explicit `public` API module boundary; Python: mandatory `__all__` in `__init__.py`, scaffolded by `generate` itself). Do not let "what is public" be an undocumented accident of language defaults — this removes an entire class of false positives before the gate ever runs.
- Provide an explicit, logged, justification-required override path (mirrors `revapiAcceptBreak --justification "..."`) rather than a silent bypass flag — this is also what gives the audit trail the original Modular design leaned on for SOC2/compliance purposes.
- Never treat "diff tool found nothing" as "definitely non-breaking" in test assertions. Test the gate against known-tricky cases per language (Go: adding an exported struct field that breaks unkeyed-literal or embedding cases; Java: adding a default method to an interface; Python: changing a keyword-vs-positional parameter kind) — these are the documented edge cases where naive tools disagree with intuition.

**Warning signs:**
- The gate has 100% agreement between "diff says compatible" and "actual consumer code still compiles/passes" on every test fixture — that's suspicious, not reassuring; it likely means your fixtures aren't exercising the known edge cases.
- Engineers start reflexively bumping major "just to be safe" or bypassing the gate — a sign the tool is too noisy to trust (the exact failure Go's apidiff authors call out).

**Phase to address:**
`publish` slice (semver gate). This is the highest-research-risk phase in the whole roadmap — flag it for phase-specific deep-dive research before implementation, especially the Python `__all__`-enforcement design, since it has no language-level equivalent to Go's capital-letter export rule or Java's `public` keyword.

---

### Pitfall 3: Local OCI Registry Garbage Collection / TTL Semantics Silently Corrupt or Leak State

**What goes wrong:**
A local/ephemeral OCI registry (whether a real `distribution/distribution` or `zot` instance, or a hand-rolled local-registry stand-in) either (a) garbage-collects blobs that are still referenced by another manifest because GC doesn't correctly account for shared layers/digests across images, or (b) never garbage-collects anything because TTL/GC wiring silently no-ops, filling disk over a long demo/dev session, or (c) a proxy/cache layer removes a blob from storage without invalidating a metadata cache, so a subsequent pull returns a "manifest exists, blob missing" error that looks like registry corruption.

**Why it happens:**
These are real, documented upstream bugs in `distribution/distribution` (TTLExpirationScheduler deleting blobs still referenced by other images; `proxy.ttl>0` leaking blobs that are never collected because the scheduler-state and actual GC enablement flag can disagree) — not hypothetical. A homegrown "local registry" for this project inherits the same class of bug the moment it does content-addressable storage with any reference counting or expiry, because the fundamental hard part (shared-blob reference counting across manifests) is exactly what upstream got wrong multiple times.

**How to avoid:**
- Store blobs keyed strictly by content digest (sha256), never by (repo, tag) or (repo, mediatype) — the upstream `oras-go`-adjacent bug class here is GC misidentifying blobs because storage was keyed by descriptor/media-type instead of digest.
- If implementing any GC/TTL at all for the local registry, make deletion strictly reference-counted: a blob is only eligible for deletion when zero manifests reference its digest, verified by a full manifest scan, not a scheduler timestamp. For a laptop-scale demo, prefer no GC at all (accept unbounded local disk growth) over a buggy GC — an intentionally simple always-keep policy is safer than a subtly wrong eviction policy.
- Test-first: write a failing test that pushes two images sharing a common base layer digest, deletes one image, and asserts the shared blob is still retrievable via the second image's manifest — this is the exact upstream bug class (#4191) reproduced as a regression test before any GC code exists.
- Keep push/pull idempotent and digest-addressed end-to-end so `deps`/`search` reads are never surprised by a manifest that references a since-deleted blob.

**Warning signs:**
- `deps --graph` or `search` intermittently returns "manifest found, content missing" — a strong signal of a GC/reference-counting bug, not a data-model bug.
- Local registry disk usage either never shrinks (GC never runs) or shrinks unexpectedly right after an unrelated push (GC is too aggressive).

**Phase to address:**
`deps`/publish-backing slice (OCI + local registry + FS index). Write the shared-blob regression test described above before writing any storage/eviction code — this is a natural TDD anchor point.

---

### Pitfall 4: Bespoke CLI Abstraction Drifts From and Hides the Underlying Toolchain

**What goes wrong:**
`modular generate`/`publish`/`deps`/`search` becomes an opaque wrapper that reimplements what `go build`, `mvn`/`gradle`, `pip`/`twine`, `docker`, and `helm` already do, rather than orchestrating them. Over time the wrapper's behavior diverges from the underlying tool's actual behavior (e.g., the CLI's notion of "the build passed" stops matching what `go test ./...` would actually report), and when something breaks, engineers can't debug it because the abstraction hides the real command that ran.

**Why it happens:**
It's tempting to reimplement version-bump logic, dependency resolution, or build invocation directly inside the platform CLI instead of shelling out to (or explicitly documenting the underlying) native language tool. Platform-engineering retrospectives consistently flag "building a bespoke CLI to hide `kubectl`/`helm`" as a top adoption-and-maintainability failure — the wrapper drifts, and only the person who wrote it understands it.

**How to avoid:**
- `generate` should scaffold projects that use the native toolchain (Go modules, Maven/Gradle, pip/poetry) unmodified — Modular's own non-goal is explicit: "not a replacement for language-specific build tools... scaffolds into them, does not replace them." The generated CI YAML should literally invoke `go build`/`gradle build`/`pytest`, not a wrapped equivalent.
- Where the platform CLI must add behavior on top (the semver gate, the dependency graph), make it an explicit, visible extra step in generated CI (e.g., a named `modular-publish-gate` CI job that engineers can read, not a hidden pre-hook), so `modular publish` is legible as "run the real build, then run this specific extra check," not a black box.
- Always print/log the exact underlying commands the CLI is orchestrating (build tool invocations, `docker`/registry calls) — cheap to add, and directly prevents the "only two people understand it" failure mode.

**Warning signs:**
- A generated project's CI passes locally with the native tool but fails (or vice versa) when run through the platform CLI, with no clear reason why.
- Engineers ask "what does `modular publish` actually run?" and the answer isn't discoverable from a `--verbose`/`--dry-run` flag or the generated CI file.

**Phase to address:**
`generate` slice (toolchain integration) and `publish` slice (gate visibility). Verify with an integration test that generated CI YAML for each language shells out to the real, standard build command for that language, not a platform-specific substitute.

---

### Pitfall 5: OTel/CI/Helm Instrumentation Stubs Are Copy-Pasted Boilerplate That Silently Breaks at the First Non-Trivial Boundary

**What goes wrong:**
The generated OTel stub wires up an SDK and exporter and demonstrates one in-process span, which looks complete in a demo. The moment a generated service calls another generated service, or does anything async (a queue, a background job, a goroutine/thread pool), traces silently fragment into disconnected single-span traces with zero errors raised — because context propagation across process or concurrency boundaries was never actually wired, only the "happy path" single-request span.

**Why it happens:**
OTel auto-instrumentation handles context injection/extraction automatically for one blessed HTTP client/server pair, but everything else — message queues, background workers, manual goroutines, cross-language calls (Go service calling a Python service) — requires explicit `inject`/`extract` calls that a generic multi-language template generator is easy to under-implement or skip per language. This is a widely documented, silent failure mode: "the trace breaks silently — you get two disconnected traces instead of one," and teams have been reported running OTel for months with broken propagation through async workers without ever noticing.

**How to avoid:**
- Define "instrumented" precisely per generated template as a testable contract, not a vibe: (1) global propagator registered at startup, (2) outbound calls from the generated starter code inject context, (3) inbound entrypoints extract context, (4) a documented extension point for queues/async that is exercised by at least one example in the stub, not left as a comment.
- For the demo flow specifically, since `deps --graph` and `search` are the CLI's own cross-cutting calls, dogfood context propagation in the CLI's own OTel stub as the first real multi-hop test case, since the CLI's own commands are the only guaranteed "service A calls service B" flow available to test against before any generated services exist.
- Golden-path test: generate two services, wire a request from one to the other using the template's default HTTP client (not a hand-modified one), assert a single trace ID spans both — this directly targets the "someone uses a raw HTTP client instead of the instrumented one" bug class.

**Warning signs:**
- Demo dashboards show many single-span traces instead of connected multi-hop traces the moment more than one generated service is involved.
- The OTel stub "looks done" (SDK initialized, one span visible in Grafana) but nothing in the generated template ever calls `propagation.Inject`/`Extract`.

**Phase to address:**
`generate` slice (OTel stub content) — write the cross-service trace-continuity test before writing the instrumentation stub content, per TDD philosophy.

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|-----------------|-----------------|
| No generation manifest/provenance in generated projects | Faster first `generate` slice | Impossible to ever add drift-detection/update later without breaking existing projects | Never — cost to add later is much higher than the ~1 field write now |
| Single "compatible/incompatible" boolean from the diff tool with no override/justification path | Simpler publish gate logic | Engineers bypass or distrust the gate the first time it's wrong (either direction) | Only for a throwaway spike, never for the slice that's meant to demo the paved-road value prop |
| Local registry with no reference counting (delete-on-tag-removal only) | Simple GC-free implementation | Unbounded disk growth over long dev sessions | Acceptable for v1/laptop-scale — explicitly prefer this over a buggy reference-counted GC |
| One shared "generic" OTel stub copy-pasted across Go/Java/Python with no per-language propagation wiring | Looks feature-complete faster | Traces silently fragment across language boundaries — exactly the failure this project is meant to demonstrate solving | Never for the OTel-stub deliverable itself; acceptable only for genuinely out-of-scope stretch languages |
| Treating "public API" implicitly (no explicit `__all__`/module boundary) in generated Python/Java templates | Less generator code | Semver gate produces majority-false-positive major bumps on Python, unpredictable on Java without an explicit API module | Never — cost is directly on the critical path of the publish-gate demo |

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|-------------------|
| Go `apidiff`/`x/exp/apidiff` | Treating "no incompatible changes reported" as proof of safety; ignoring known blind spots (embedded-field ambiguity, merged named types breaking type switches) | Treat apidiff output as a lower bound on risk; keep a table-driven test fixture set covering the documented "five ways code can still break" from the apidiff README |
| Java `japicmp`/`revapi` | Running the raw tool with no exclude/accept-list, so every internal-only or annotation-driven change fails the build with noise | Configure explicit include/exclude filtering for internal packages up front; use the documented `revapiAcceptBreak --justification` escape hatch instead of disabling the check |
| Python `griffe check` | Relying on default "no leading underscore = public" inference | Scaffold every generated Python project with an explicit `__all__` in `__init__.py` from day one; run `griffe check` against that explicit surface only |
| OCI registry (local/distribution-style) | Assuming a TTL/proxy-cache config option ("`proxy.ttl`", GC toggles) behaves symmetrically in both directions (enabling vs. disabling) | Explicitly test both the "GC enabled" and "GC disabled → re-enabled" transitions; upstream `distribution/distribution` has multiple open issues where this exact transition leaks or over-deletes blobs |
| Ephemeral/pull-through registries in CI/dev loops (ttl.sh-style patterns) | Depending on an external zero-SLA ephemeral registry for a reliability-sensitive dev loop | Keep the local registry fully local/offline for this project (matches the stated "local FS templates, local registry" constraint) — don't introduce any network-dependent registry even for convenience |

## Performance Traps

| Trap | Symptoms | Prevention | When It Breaks |
|------|----------|------------|-----------------|
| Dependency graph computed by re-walking the full FS package index on every `deps --graph` call | `deps --graph` gets slower as more packages/projects are generated during a demo session | Build/maintain an adjacency structure (in-memory or SQLite) incrementally on publish, not recomputed from scratch on every query | Noticeable once dozens of packages are published in one session — plausible within a single demo/dev day given TDD iteration volume |
| Search index is a linear scan over all generated project metadata files | `search <query>` latency grows linearly with number of generated projects | Maintain a simple indexed store (even a SQLite FTS table or in-memory map keyed by name/language) updated on `generate`/`publish`, not a directory walk per query | Breaks perceptibly once the project count exceeds what's comfortable to `ls`/grep by hand — low threshold for a CLI meant to feel instant |
| Template rendering re-reads and re-parses every template file from disk on every `generate` invocation with no caching | Negligible at laptop scale for v1 | Not worth optimizing prematurely — flag as a non-issue for v1 | Only matters if templates grow very large or `generate` is called in a tight loop (e.g., a future batch-generation feature) |

## Security Mistakes

| Mistake | Risk | Prevention |
|---------|------|------------|
| Template parameters (project name, team, output directory) used verbatim as filesystem path components without validation | Path traversal / arbitrary file write outside the intended project directory — a documented, actively-exploited bug class in real scaffolding CLIs (Google's `agents-cli`, Microsoft's `kiota`, RuoYi's code generator, Backstage's scaffolder all had CVEs/advisories for exactly this) | Reject any parameter containing `..` path segments, absolute paths, or drive-letter prefixes before joining into a filesystem path; canonicalize the final destination and assert it is still inside the target project root before any write — apply this uniformly across Go/Java/Python code paths, not just the "default" language (the `agents-cli` CVE existed specifically because the check silently no-op'd for non-Python languages) |
| Copying template files/directories via APIs that dereference symlinks | Symlink-based arbitrary file read (reading a file outside the project into the generated output) or, on extraction of a fetched template, arbitrary write | If templates are ever fetched from anywhere other than the trusted local FS store (even in future scope), explicitly detect and refuse to follow symlinks during copy; for the local-FS-only v1, still avoid `os.Symlink`-following copy helpers as a defensive habit given multi-language template trees may contain vendored symlinks |
| No validation that a "publish" actually targets the package/team the invoking engineer is authorized for (even in a single-operator playground, the design should not assume this away silently) | Not a v1 blocker given single-operator scope, but worth an explicit note rather than a silent gap, per the project's own "opt-in playground, no adoption mandate" scoping — LOW priority, MEDIUM confidence this matters for the stated scope | Document as an explicit out-of-scope decision (already partially covered by PROJECT.md's Out of Scope section) rather than an accidental omission |

## UX Pitfalls

| Pitfall | User Impact | Better Approach |
|---------|-------------|-------------------|
| Publish gate fails with a raw diff-tool error dump (raw apidiff/japicmp/griffe output) | Engineer can't tell what to actually do — "which symbol, what's the fix, do I bump major or is this a false positive" | Translate diff-tool output into a consistent, per-language "here's what changed, here's the semver implication, here's the override command" format, mirroring `revapiAcceptBreak`'s explicit justification pattern |
| `generate` fails halfway through (e.g., mid-render error) and leaves a partially-written project directory | Confusing partial state; re-running `generate` may error on "directory already exists" without telling the user it's from a failed prior attempt | Render to a temp directory first, then atomically move/rename into place only on full success — classic transactional-scaffold pattern, cheap to implement, prevents an entire class of confusing bug reports |
| `deps --graph` returns a flat list instead of visually communicating blast radius | Engineer can't quickly judge "is this a big breaking change or a small one" before deciding to publish | Even a simple ASCII tree or depth-annotated list (direct vs. transitive consumers) meaningfully improves the "should I be scared of this publish" decision — this is the entire value proposition of the `deps` command per the original design intent |

## "Looks Done But Isn't" Checklist

- [ ] **`generate`:** Often missing a written manifest/provenance record — verify a fresh `generate` run produces a file recording template version + params, not just the rendered project files.
- [ ] **`publish` semver gate:** Often missing an explicit, tested "public API surface" definition per language (Go export convention, Java module boundary, Python `__all__`) — verify the gate is tested against at least one known-tricky compatible-looking-but-breaking case per language, not just an obviously-breaking rename.
- [ ] **Local OCI registry:** Often missing a shared-blob-across-manifests regression test — verify deleting one image doesn't corrupt a second image sharing a base layer.
- [ ] **OTel stubs:** Often missing actual context propagation code (only SDK init + one span exist) — verify a two-service demo call produces one connected trace, not two fragments.
- [ ] **`search`:** Often missing index consistency with the underlying FS/registry state — verify `search` reflects a `publish` that just happened without needing a restart or manual reindex.
- [ ] **CI/Helm/Dockerfile templates:** Often generated once and never actually run — verify the generated GitHub Actions workflow and Dockerfile are executed (not just present) as part of the `generate` slice's test suite, e.g., via a smoke build.

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|-----------------|
| Discovering template drift after many projects generated without provenance | HIGH | Requires manually diffing generated projects against current template by hand, or accepting the gap and only fixing it forward via a manifest added starting now |
| Semver gate false-positive/negative discovered after several publishes | MEDIUM | Add the missed edge case as a regression test fixture immediately (TDD-appropriate), then re-run the gate against publish history if audit trail exists |
| Local registry GC bug corrupts shared blobs mid-demo | LOW (for this project's scale) | Wipe and re-seed the local registry from the FS package index/re-publish, since it's a laptop-local store with no external consumers — cheap because scope is explicitly single-operator |
| OTel propagation gap discovered late (traces fragmenting) | MEDIUM | Isolate to the specific boundary type (async/cross-language/queue) via the golden-path multi-hop test, then patch that one template's stub — cost scales with how many languages/boundaries already shipped without the fix |

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|-------------------|---------------|
| Template drift / no provenance | `generate` slice | Test asserts manifest file exists post-`generate` with template version + params recorded |
| Semver gate heuristic trusted blindly | `publish` slice | Table-driven tests per language covering documented tool blind spots (Go embedding/type-switch cases, Java default-method addition, Python `__all__`-based public surface) |
| OCI local registry GC/reference-counting bugs | `deps`/registry-backing slice | Regression test: two manifests sharing a blob digest, delete one, assert the other's blob is still retrievable |
| Bespoke CLI hides/diverges from native toolchain | `generate` + `publish` slices | Integration test asserts generated CI YAML invokes the real native build command per language; `--verbose`/dry-run surfaces underlying commands |
| OTel stub missing cross-boundary propagation | `generate` slice | Two-service smoke test asserts a single trace ID spans a call between two generated services using the template's default client |
| Search index staleness/inconsistency | `search` slice | Test asserts `search` reflects a `publish` that just completed within the same process/session with no manual reindex step |

## Sources

- Go `apidiff` official README/docs (`go.googlesource.com/exp/+/refs/heads/master/apidiff/README.md`, `pkg.go.dev/golang.org/x/exp/apidiff`) — HIGH confidence, official Go tooling documentation on structural-diff limitations.
- `golang/go` issue #65112 (`apidiff` incorrectly reporting incompatible type-alias changes) — HIGH confidence, verified upstream bug tracker.
- Marc Dougherty, "Go dependencies and API diffs" (marcdougherty.com, 2025) — MEDIUM confidence, independent practitioner account of `apidiff` false positives, corroborates official docs.
- `revapi/gradle-revapi`, `palantir/gradle-revapi`, `melix/japicmp-gradle-plugin`, `siom79/japicmp` GitHub docs — HIGH confidence, official plugin/tool documentation for Java semver gating and override/accept-break patterns.
- Griffe official docs (`mkdocstrings.github.io/griffe`) + GitHub discussion #230 — HIGH confidence, official docs explicitly stating "public API is subjective" and `__all__`-based resolution.
- `distribution/distribution` GitHub issues #4191 (TTL scheduler deleting in-use blobs) and #4249 (`proxy.ttl` leak) — HIGH confidence, verified upstream open-source registry bug reports.
- `tektoncd/plumbing` issue #3380 (deploying local `zot` OCI registry for CI, GC/retention design notes) — MEDIUM confidence, practitioner design doc, corroborates registry GC complexity.
- `google/agents-cli` GitHub security issues #50 and #51 (path traversal + symlink-follow in scaffolding CLI) — HIGH confidence, disclosed vulnerabilities in a directly comparable scaffolding-CLI project.
- Kiota EUVD-2026-44929 / CVE-2026-59866 etc. (path traversal in OpenAPI code generator) — HIGH confidence, published CVE data for a comparable code-generator tool.
- `dromara/RuoYi-Vue-Plus` issue #33 (path traversal in code generator, CVSS 7.5) — HIGH confidence, disclosed vulnerability, directly analogous "genPath" scaffolding bug.
- Backstage Scaffolder symlink path traversal advisory (GitLab Advisory Database, CVE-2026-24046) — HIGH confidence, published security advisory for a widely-used internal-developer-platform scaffolder — strongly relevant precedent given this project's IDP framing.
- Reqhiem.dev "Copier vs. Cookiecutter" and Blenddata "Cruft vs copier" blog posts — MEDIUM confidence, independent practitioner sources, mutually corroborating on template-drift mechanics.
- Dominique Dumont, "Drawbacks of using Cookiecutter with Cruft" (2025) — MEDIUM confidence, practitioner retrospective on template-update merge-conflict pain; corroborates "keep templates thin, push logic to versioned libraries" recommendation.
- GitPlumbers "Stop Building a Platform; Build a Paved Road," Upsun "The paved road to production," InfraZen "Why Developers Bypass Your Platform," zop.dev "The IDP Adoption Problem" — MEDIUM confidence, independent platform-engineering practitioner sources, mutually corroborating on the "bespoke CLI hides real tooling" and adoption-funnel pitfalls.
- OneUptime, Last9, SigNoz, and Andrew Odendaal OpenTelemetry context-propagation guides — MEDIUM confidence, independent vendor/practitioner sources mutually corroborating on the "raw HTTP client / unpropagated async boundary breaks traces silently" failure mode; cross-checked against OTel's own documented propagator model (composite propagator, inject/extract API).

---
*Pitfalls research for: Multi-language developer-platform CLI (Modular/Nerv-style paved road)*
*Researched: 2026-07-24*
