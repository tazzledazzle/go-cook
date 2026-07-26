# Code Analysis Subagent System — Design Spec

**Date:** 2026-07-11
**Status:** Approved

---

## Overview

Five skill files implement a parallel code analysis suite for the `learn-you-a-torrent` Go codebase. Three analysis agents run in parallel, then a CI/CD design agent reads all three reports and produces a pipeline design. A single orchestrator skill (`/gsd-analyze`) is the normal entry point.

---

## Components

### Skills (`.claude/skills/`)

| Skill file | Invocation | Subagent type | Purpose |
|---|---|---|---|
| `analyze-smells.md` | `/analyze-smells` | `gsd-code-reviewer` | Identify code smells across all `.go` files |
| `analyze-patterns.md` | `/analyze-patterns` | `gsd-code-reviewer` | Identify Go anti-patterns |
| `analyze-syntax.md` | `/analyze-syntax` | `general-purpose` | Run `go vet` + `golangci-lint`, parse output |
| `design-cicd.md` | `/design-cicd` | `gsd-planner` | Design CI/CD pipeline from analysis reports |
| `gsd-analyze.md` | `/gsd-analyze` | Orchestrator | Dispatch parallel trio, then CI/CD agent |

---

## Execution Flow

```
/gsd-analyze
    │
    ├──[parallel]──▶ /analyze-smells   → docs/analysis/CODE-SMELLS.md
    ├──[parallel]──▶ /analyze-patterns → docs/analysis/ANTI-PATTERNS.md
    └──[parallel]──▶ /analyze-syntax   → docs/analysis/SYNTAX.md
                           │
                    [all three complete]
                           │
                           ▼
                    /design-cicd       → docs/analysis/CICD-DESIGN.md
                    (reads all three reports first)
```

All four agents also print a terminal summary on completion.

---

## Agent Specifications

### `analyze-smells`

**Subagent type:** `gsd-code-reviewer`

**Scope:** All `.go` files under `internal/` and `cmd/`

**What it looks for:**
- Functions exceeding ~40 lines
- Nesting depth greater than 3 levels
- Parameter lists with more than 4 parameters
- Duplicated logic blocks (copy-paste patterns across files)
- Unclear or abbreviated variable/function names
- Structs with more than ~8 fields

**Output:** `docs/analysis/CODE-SMELLS.md`

**Terminal summary:**
```
[analyze-smells] ✓ N high, N medium, N low — docs/analysis/CODE-SMELLS.md
```

---

### `analyze-patterns`

**Subagent type:** `gsd-code-reviewer`

**Scope:** All `.go` files under `internal/` and `cmd/`

**What it looks for:**
- Goroutine leaks (goroutines launched without a cancellation path)
- Ignored `error` return values
- `sync.Mutex` held across blocking I/O calls
- Overly broad interfaces (more than ~5 methods)
- Naked returns in functions longer than a few lines
- Returning concrete types where interfaces would decouple callers
- Global mutable state

**Output:** `docs/analysis/ANTI-PATTERNS.md`

**Terminal summary:**
```
[analyze-patterns] ✓ N high, N medium, N low — docs/analysis/ANTI-PATTERNS.md
```

---

### `analyze-syntax`

**Subagent type:** `general-purpose`

**Tools used:**
1. `go vet ./...` — catches correctness issues (printf format mismatches, unreachable code, etc.)
2. `golangci-lint run ./...` — aggregated linters (errcheck, staticcheck, gocritic, revive, etc.)

**Process:**
1. Run `go vet ./...`, capture output
2. Run `golangci-lint run ./...`, capture output
3. Parse both outputs, deduplicate overlapping findings
4. Categorize by severity: errors (vet failures) → HIGH; lint warnings → MEDIUM/LOW

**Output:** `docs/analysis/SYNTAX.md`

**Terminal summary:**
```
[analyze-syntax] ✓ N errors, N warnings — docs/analysis/SYNTAX.md
```

**Dependency:** `golangci-lint` must be installed (`brew install golangci-lint` or `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`).

---

### `design-cicd`

**Subagent type:** `gsd-planner`

**Inputs read before designing:**
- `docs/analysis/CODE-SMELLS.md`
- `docs/analysis/ANTI-PATTERNS.md`
- `docs/analysis/SYNTAX.md`
- `go.mod` (for Go version)
- `cmd/torrent/main.go` (for build entrypoint)

**What it produces:**
- Pipeline rationale (why these stages, in this order)
- Ready-to-paste `.github/workflows/ci.yml`
- `golangci-lint` config snippet (`.golangci.yml`)
- Notes on what to address before enabling branch protection rules

**Output:** `docs/analysis/CICD-DESIGN.md`

**Terminal summary:**
```
[design-cicd] ✓ Pipeline designed — docs/analysis/CICD-DESIGN.md
```

---

### `gsd-analyze` (Orchestrator)

**Process:**
1. Print banner: `GSD ► ANALYZE`
2. Dispatch `analyze-smells`, `analyze-patterns`, `analyze-syntax` in parallel (single Agent call with 3 subagent blocks)
3. Wait for all three to complete
4. Dispatch `design-cicd` (sequential, reads the three completed reports)
5. Print aggregate summary:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 Analysis complete — 4 reports in docs/analysis/
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 High severity findings: N (review before next commit)
 Reports: CODE-SMELLS.md, ANTI-PATTERNS.md, SYNTAX.md, CICD-DESIGN.md
```

---

## Output File Structure

All reports follow this format:

```markdown
# <Report Title>
Generated: <date>

## Summary
<3-5 sentence overview of findings>

## Findings

### <Finding Title> [SEVERITY: HIGH|MEDIUM|LOW]
**File:** `internal/pkg/file.go:42`
**Description:** ...
**Recommendation:** ...

## Stats
- Files scanned: N
- Findings: N high, N medium, N low
```

`CICD-DESIGN.md` uses a different structure: pipeline overview, stage definitions, tool selections, and the ready-to-paste YAML.

---

## Output Directory

```
docs/analysis/
├── CODE-SMELLS.md
├── ANTI-PATTERNS.md
├── SYNTAX.md
└── CICD-DESIGN.md
```

---

## Constraints

- Skills are stored in `.claude/skills/` to match the existing project convention
- `golangci-lint` must be available in the environment before `/analyze-syntax` runs
- Individual skills (`/analyze-smells`, `/analyze-patterns`, `/analyze-syntax`, `/design-cicd`) can be invoked standalone without the orchestrator
- No code is modified by any agent — all agents are read-only except for writing report files
