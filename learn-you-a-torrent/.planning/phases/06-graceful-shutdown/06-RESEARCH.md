# Phase 6 Research: Graceful Shutdown

**Researched:** 2026-07-10
**Confidence:** HIGH

## Cancellation Flow

```
SIGINT/SIGTERM → cancel context → peerWorker sees ctx.Done()
  → conn.Close() unblocks reads → workers exit → wg.Wait()
  → writer.Close() → return context.Canceled
  → CLI prints last progress line
```

## Testing Without Signals

```go
ctx, cancel := context.WithCancel(context.Background())
go func() {
  time.Sleep(50 * time.Millisecond)
  cancel()
}()
err := d.Download(ctx, tor, dir)
// want errors.Is(err, context.Canceled)
```

Use mock peer that blocks on `ReadMessage` until conn closed.

## Conn Close on Cancel Pattern

```go
go func() {
  <-ctx.Done()
  _ = conn.Close()
}()
```

Placed in `peerWorker` after successful dial.

## Sources

- ROADMAP Phase 6
- `.planning/research/PITFALLS.md` — shutdown checklist

---
*Phase 6 research complete*
