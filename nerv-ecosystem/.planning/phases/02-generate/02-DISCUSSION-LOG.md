# Phase 2: Generate — Discussion Log

**Date:** 2026-07-25
**Mode:** `--auto` (single pass)

## Gray areas selected
All: Output path & CLI flags, Template storage & engine, Generated project depth, Provenance manifest, Overwrite & path-safety, Store registration & schema, Public-API surface scaffolding.

## Auto-selected choices

| Area | Question | Selected |
|------|----------|----------|
| Output path & CLI flags | Flags vs prompts? | Flags-only, CI-friendly (recommended) |
| Template storage | Where do templates live? | embed.FS + optional FS override (recommended) |
| Generated depth | Stub vs runnable? | Thin compiling service with metrics endpoint (recommended) |
| Provenance | Filename/format? | `.nerv-manifest.json` at project root (recommended) |
| Safety | Overwrite policy? | Hard fail on non-empty target (recommended) |
| Store | When to register? | After successful tree+manifest write (recommended) |
| Public API hooks | How explicit? | Scaffold language-appropriate public surface from day one (recommended) |

## Notes
No interactive prompts. Decisions derived from PROJECT.md, research SUMMARY/STACK/ARCHITECTURE/PITFALLS, Phase 1 SKELETON, and GEN-01…09 requirements.
