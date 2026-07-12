# Phase 6 Context: Graceful Shutdown

**Gathered:** 2026-07-10
**Status:** Ready for planning
**Source:** TDD + ROADMAP Phase 6

<domain>
## Phase Boundary

SIGINT/SIGTERM cancels an in-progress download cleanly: peer goroutines stop via `context.Context`, connections close, writer flushes/closes, exit message shows last progress. File contains only **verified complete pieces** (already true — no change to write path).

Out of scope: resume/checkpoint, seeding teardown, new protocol features.
</domain>

<decisions>
## Implementation Decisions

### TDD discipline
- RED → GREEN commits per plan
- Test cancellation with blocking mock peer + `context.WithCancel` (no real signals in unit tests)
- CLI signal wiring tested via injectable `notifySignals` hook

### Downloader (CLI-03 core)
- On `ctx.Done()`: wait for peer workers (`WaitGroup`), then `writer.Close()`
- `peerWorker`: goroutine closes `conn` when `ctx` cancelled (unblocks `ReadMessage`)
- Return `ctx.Err()` on cancellation (not a generic error)
- Do not `MarkComplete` unless piece fully validated and written

### CLI
- `signal.Notify` for SIGINT/SIGTERM
- `context.WithCancel` passed to `Downloader.Download`
- On graceful cancel: print `\nShutdown: <progress>` using last `OnProgress` snapshot
- Extract `runDownload(ctx, args)` for testability from `Run`

### File consistency
- Relies on existing rule: `WritePiece` only after SHA1 validate
- Cancel mid-piece leaves prior verified pieces intact; incomplete piece not written
</decisions>

---
*Phase: 06-graceful-shutdown*
