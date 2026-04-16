# AGENTS.md

## Mission

Build `braind` as a small, reliable Unix daemon in Go.
Priorities, in order:

1. Correctness
2. Safety/privacy
3. Simplicity
4. Performance

## Source of Truth

Always follow `@SPEC.md` for product and architecture decisions.
If this file conflicts with `@SPEC.md`, `@SPEC.md` wins.

## Ground Rules

- Keep code boring and explicit.
- Prefer standard library unless a dependency clearly reduces complexity.
- No speculative abstractions.
- No dynamic plugin loading for MVP.
- No shell execution (`os/exec`) in daemon/tool execution paths.
- Logseq markdown is source of truth for vault content.
- Provider-ingested data lives in sqlite.

## Directory Layout

```
braind/
├── cmd/
│   ├── braind/              # daemon entrypoint
│   └── brain/               # CLI entrypoint
│
├── tools/                   # extension point (README.md only)
│   └── README.md            # explains how to write + register a tool package
│
├── providers/               # extension point (README.md only)
│   └── README.md            # explains how to write + register a provider package
│
├── builtin/                 # always-compiled core
│   ├── tools/               # built-in tools
│   │   ├── journal/
│   │   ├── todo/
│   │   ├── search/
│   │   └── page_read/
│   └── providers/            # built-in providers
│       ├── rss/
│       └── filesystem/
│
├── pkg/                     # public library packages
│   └── ipc/                # client lib for TUI/frontends
│
├── internal/                 # core daemon internals
│   ├── config/             # config loading + validation
│   ├── vault/              # vault model, directory layout, manifest
│   ├── engine/             # per-vault: context, LLM loop, tool execution
│   ├── llm/                # Ollama client (tool-calling protocol)
│   ├── store/              # sqlite + sqlite-vec operations
│   ├── gitops/             # git auto-commit, revert
│   ├── changelog/          # changelog.org writer
│   ├── watch/              # fsnotify vault watcher
│   ├── jobs/               # job definitions
│   └── daemon/             # server lifecycle, UDS HTTP router
│
└── example/
    └── config.yaml         # full example config with all defaults + all features
```

## Architecture Boundaries

- `cmd/braind`: daemon entrypoint
- `cmd/brain`: CLI entrypoint
- `tools/`, `providers/`: extension points — empty dirs with README.md contract docs
- `builtin/`: always-compiled tools and providers
- `pkg/`: public library packages importable by external projects
- `internal/`: core daemon internals, not importable from outside
- Keep responsibilities separated:
  - config loading/validation
  - IPC server/client
  - tool registry + execution
  - provider ingestion
  - git/changelog operations

Do not mix CLI concerns into daemon internals.

## IPC and Runtime

- IPC is HTTP+JSON over Unix domain sockets only.
- No TCP listener by default.
- External timers (systemd/launchd) trigger `brain run <job>` in MVP.
- Per-vault isolation is mandatory (data, tools, config scope).

## Go Code Style

- Follow `gofmt` and `go vet`.
- Small functions, clear names, early returns.
- Return errors with context (`fmt.Errorf("...: %w", err)`).
- Avoid global mutable state.
- Prefer interfaces at boundaries, concrete types internally.
- Keep package APIs minimal and cohesive.

## Testing Requirements

For every non-trivial change, add/update tests.

Minimum expectations:

- Unit tests for pure logic and parsing.
- Table-driven tests where practical.
- Deterministic tests (no sleeps, no external network).
- Use temp dirs/files for filesystem tests.
- Mock/stub external dependencies (LLM, git ops, providers).
- Cover failure paths, not just happy paths.

Before finishing work, run:

- `go test ./...`
- `go vet ./...` (if enabled in project workflow)

## Git and Changelog Semantics

- One invocation that mutates vault files => one git commit.
- `undo` uses `git revert`, never destructive reset.
- Changelog entries should describe:
  - what changed
  - why
  - tools/model involved

## Security and Privacy

- Treat all model/tool inputs as untrusted.
- Enforce per-vault tool allowlist on every tool call.
- Never allow cross-vault reads/writes.
- Network access must be explicit capability:
  - `network/pull`
  - `network/push`
- Providers use `network/` capability class.

## Definition of Done

A change is done when:

1. Code is formatted and readable.
2. Tests added/updated and passing.
3. Error handling is explicit.
4. No boundary/security rule is violated.
5. Brief docs/comments updated if behavior changed.

## Non-Goals (MVP)

- In-daemon scheduler
- Hot-loaded plugins
- Heavy attachment processing
- Cloud-only dependencies

## Extension Model

User extensions (tools/providers) live in `tools/` and `providers/` as git submodules.
Each is a Go package imported in `cmd/braind/main.go` and registered via a compile-time
`Register()` call. No dynamic loading. To add an extension:

```bash
git submodule add <git-url> tools/weather
# add import to cmd/braind/main.go
go build ./cmd/braind
```

See `tools/README.md` and `providers/README.md` for the package interface contract.
