POST /projects
│
▼
[creation.Service.Create]
1. Resolve dependencies (depgraph.Resolver)
2. Render template     (template.Engine)
3. Wire CI              (cihook.CIHook)
4. Build Project metadata
5. Hand metadata to directory.Service.Register  ◄── only after 1–3 succeed
   │
   ▼
   [directory.Service.Register]
   a. Persist  (registry.Registrar — Bolt or Postgres)
   b. Index    (searchindex.Index — in-memory inverted index over metadata)

GET /search?q=...
│
▼
[directory.Service.Search] → ranked metadata matches
[directory.Service.CodeSearchLinks] → OpenGrok / Sourcegraph URLs for the same query