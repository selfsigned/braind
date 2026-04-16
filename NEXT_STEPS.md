# NEXT STEPS

1. **Implement `cmd/braind` real daemon bootstrap**  
   Load config, initialize per-vault servers/sockets from `daemon.socket_dir` (`$XDG_RUNTIME_DIR/braind/<vault>.sock`), and add graceful shutdown hooks.

2. **Implement `builtin/tools/page_read` + `builtin/tools/search` real behavior**  
   - `page_read`: read page/journal raw markdown safely within vault boundaries.  
   - `search`: wire to sqlite-backed retrieval (initial lexical fallback, later embeddings/sqlite-vec).

3. **Wire allowlist-enforced tool execution path in `internal/engine` and connect `/ask`**  
   Route `POST /v1/{vault}/ask` through engine validation + tool registry + tool execution response flow (single-turn protocol boundary, no agentic loop).
