# Concerns

**Analysis Date:** 2026-07-25

## Open technical debt (from Phase 1 code review)

| ID | Severity | Issue | Suggested fix | Bead |
|----|----------|-------|---------------|------|
| WR-01 | Warning | Store path containing literal `?` breaks SQLite DSN query parsing and can silently disable WAL | Percent-escape path before splicing into DSN | Track as tech-debt chore if not already |
| WR-02 | Warning | TOCTOU: DB file created under process umask before `chmod 0600` | Pre-create with `os.OpenFile(..., 0o600)` before opening via driver | Same |
| IN-01 | Info | `.golangci.yml` redundantly enables `errcheck` (already in standard preset) | Drop redundant enable | Low priority |
| IN-02 | Info | Migration version parsing assumes 4-digit zero-padded filenames | Document ceiling or switch to numeric sort | Low priority |

Source: `.planning/phases/01-platform-foundation/01-REVIEW.md`

## Process / coordination concerns

- **API quota interruptions:** Phase 2 research and this codebase-map agent hit `resource_exhausted`. Resume with `/gsd-plan-phase 2 --auto` and remapping only after quota resets — do not tight-loop retries.
- **Dual tracking:** GSD STATE/ROADMAP vs beads must stay in sync on phase close (close REQ beads + phase epic when VERIFICATION passes).
- **Incomplete maps historically:** INTEGRATIONS/CONCERNS were missing until this follow-up; planners should prefer `.planning/codebase/` over stale assumptions.

## Architectural risks (upcoming)

- Java/Python semver gate is policy-tier in v1 (not full AST/bytecode) — document clearly in demos
- FTS5 content sync relies on triggers — any future bulk import must INSERT through SQL that fires triggers (or maintain index explicitly)
- Template path-traversal (GEN-07) is a known scaffolder CVE class — must validate before any write in Phase 2

## What is healthy

- Pure-Go SQLite + WAL verified under `-race`
- Store dir `0700` / DB file `0600` enforced
- TDD RED→GREEN history present for Phase 1
- Single-binary laptop path with no daemon requirement
