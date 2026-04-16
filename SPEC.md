# braind — Tiny local second-brain daemon

> "Probably won't leak your SSN."

## 1. Overview

`braind` is a local daemon that augments a Logseq-compatible markdown knowledge
base with LLM-powered tool calling. It runs entirely on-device, talks to a local
model server (Ollama / LM Studio, eventually in-process), and is scoped by
**vaults** — isolated knowledge databases with their own configuration, tool
allowlists, embedding stores, and data providers.

**Design principles:**

- **Tool-calling only, never agentic.** The LLM receives a prompt + context,
  calls tools from a strict allowlist, and stops. No shell access, no loops, no
  autonomous retry chains. Every tool invocation is validated against the
  vault's config before execution.
- **Logseq markdown is source of truth.** The daemon can be killed at any time
  and the vault remains a perfectly valid Logseq graph. All mutations go through
  a controlled `journal` tool that writes Logseq-compatible markdown.
- **Privacy by architecture.** No telemetry, no network egress to LLM APIs
  (local models only), per-vault isolation, all data stays on disk.
- **Transparency.** Every write is git-committed. A daily changelog records what
  the daemon did, why, and which model/tool was used.

---

## 2. Architecture

```
┌─────────────────────────────────────────────────┐
│                     brain (CLI)                  │
│          vault switch, ask, todos, log, ...      │
└──────────────────────┬──────────────────────────┘
                       │  Unix socket (per vault)
                       ▼
┌─────────────────────────────────────────────────┐
│                    braind (daemon)               │
│                                                  │
│  ┌──────────┐  ┌──────────┐  ┌───────────────┐  │
│  │ Scheduler │  │  Router   │  │ Tool Registry │  │
│  │ (cron)    │  │ (socket)  │  │ (per-vault)   │  │
│  └────┬─────┘  └────┬─────┘  └──────┬────────┘  │
│       │              │               │           │
│       ▼              ▼               ▼           │
│  ┌──────────────────────────────────────────┐    │
│  │              Engine (per vault)           │    │
│  │                                           │    │
│  │  ┌─────────────┐   ┌──────────────────┐  │    │
│  │  │ LLM Client   │   │ Context Builder   │  │    │
│  │  │ (tool-calling)│   │ (embeddings+graph)│  │    │
│  │  └──────┬──────┘   └──────────────────┘  │    │
│  │         │                                  │    │
│  │         ▼                                  │    │
│  │  ┌─────────────────────────────────────┐  │    │
│  │  │        Tool Executor (sandboxed)     │  │    │
│  │  │  journal · todo · search · <ext>...  │  │    │
│  │  └─────────────────────���───────────────┘  │    │
│  └──────────────────────────────────────────┘    │
│                                                  │
│  ┌──────────────────────────────────────────┐    │
│  │          Provider Pipeline (per vault)     │    │
│  │                                           │    │
│  │  RSS · filesystem · caldav · <ext>...     │    │
│  │       │                                   │    │
│  │       ▼                                   │    │
│  │  sqlite staging → embed (sqlite-vec)      │    │
│  │       │                                   │    │
│  │       ▼                                   │    │
│  │  compaction (every N intervals)           │    │
│  └──────────────────────────────────────────┘    │
│                                                  │
│  ┌──────────────┐  ┌────────────────────────┐    │
│  │ Git Versioner │  │ Changelog Writer       │    │
│  └──────────────┘  └────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

### 2.1 Components

| Component          | Responsibility                                        |
| ------------------ | ----------------------------------------------------- |
| **braind**         | Long-running daemon. Owns the lifecycle.               |
| **brain**          | CLI client. Talks to `braind` over Unix socket.        |
| **Router**         | Accepts socket connections, dispatches to vault engine. |
| **Scheduler**      | Cron-style job runner. Drives periodic tasks.           |
| **Engine**         | Per-vault orchestrator: builds context, calls LLM,      |
|                    | validates tool calls, executes them.                    |
| **Context Builder**| Assembles LLM prompt from recent journal entries,       |
|                    | embedding search results, active TODOs, provider data.  |
| **Tool Registry**  | Holds available tools per vault. Validates every call   |
|                    | against the vault's allowlist before execution.         |
| **Tool Executor**  | Runs validated tool calls. No shell. Go function calls  |
|                    | only.                                                  |
| **Provider Pipeline** | Ingests external data into sqlite staging tables,    |
|                    | generates embeddings via sqlite-vec, compacts          |
|                    | periodically. Data stays in sqlite, never in vault.     |
| **Git Versioner**  | Auto-commits every vault mutation with metadata.        |
| **Changelog Writer**| Appends to `changelog.org` in the vault with what was  |
|                    | done, why, which model/tool, and git diff summary.      |

### 2.2 Process Model

- Single `braind` process managing N vaults.
- Each vault gets its own goroutine group (engine + provider pipeline).
- One Unix socket per vault: `$XDG_RUNTIME_DIR/braind/<vault>.sock`
- Graceful shutdown: finish in-flight tool calls, flush providers, final git
  commit.

### 2.3 IPC Protocol

`braind` exposes a local-only HTTP+JSON API over Unix domain sockets.

- Transport: Unix socket only (no TCP bind by default)
- Encoding: JSON requests/responses
- Streaming: optional chunked JSON responses for long-running calls
- Goal: easy client support from Go, Rust, Python, shell, and TUI frontends

Example endpoints:

- `POST /v1/{vault}/ask`
- `GET /v1/{vault}/sources`
- `GET /v1/{vault}/todos`
- `POST /v1/{vault}/run/{job}`
- `POST /v1/{vault}/undo`

---

## 3. Vaults

A **vault** is an isolated unit containing:

- A Logseq-compatible markdown directory (the knowledge base).
- A sqlite database (embeddings via sqlite-vec, provider staging data).
- A git repository (versioning all vault mutations).
- A per-vault config section (tools, providers, schedules).

Vaults never share data. No cross-vault embedding queries, no cross-vault tool
calls. The daemon enforces this at the engine level.

### 3.1 Directory Layout

```
~/.config/braind/
├── config.yaml                  # global + per-vault config
└── providers/
    └── rss/
        └── feeds.opml           # provider-specific data (if needed)

~/braind-vaults/
├── personal/
│   ├── journals/                # Logseq journal pages (YYYY_MM_DD.md)
│   ├── pages/                   # Logseq pages ([[Page Name]].md)
│   ├── changelog.org            # daemon changelog (not Logseq format)
│   ├── .braind/                 # daemon metadata
│   │   └── db.sqlite            # sqlite-vec embeddings + provider staging
│   └── .git/                    # auto-versioning
└── work/
    ├── journals/
    ├── pages/
    ├── changelog.org
    ├── .braind/
    │   └── db.sqlite
    └── .git/
```

---

## 4. Configuration

Single YAML file at `$XDG_CONFIG_HOME/braind/config.yaml`.

### 4.1 Schema

```yaml
# ─── Global ──────────────────────────────────────
daemon:
  socket_dir: "${XDG_RUNTIME_DIR}/braind"
  protocol: "http+json-over-uds"
  log_level: "info"
  pid_file: "${XDG_RUNTIME_DIR}/braind.pid"

logging:
  path: "${XDG_DATA_HOME}/braind/logs"
  format: "jsonl"
  rotate_mb: 10
  max_files: 10

limits:
  interactive:
    max_turns: 6
    max_tool_calls: 12
    max_wall_time: "45s"
  maintenance:
    max_turns: 10
    max_tool_calls: 30
    max_wall_time: "10m"

embeddings:
  provider: "ollama"
  model: "nomic-embed-text"
  dimension: 768
  metric: "cosine"

llm:
  # Daytime: small fast model for interactive use
  default:
    provider: "ollama"
    model: "gemma3:4b"           # or whatever the current smol gemma is
    base_url: "http://localhost:11434"
    timeout: "30s"
    max_tokens: 2048

  # Night/heavy: bigger model for maintenance jobs
  heavy:
    provider: "ollama"
    model: "gemma3:12b"
    base_url: "http://localhost:11434"
    timeout: "120s"
    max_tokens: 4096

# ─── Vaults ──────────────────────────────────────
vaults:
  personal:
    path: "~/braind-vaults/personal"

    git:
      auto_commit: true
      commit_name: "braind"
      commit_email: "braind@localhost"

    changelog:
      enabled: true

    tools:                       # tools available to the LLM in this vault
      - name: "journal"          # built-in: write/edit journal entries
      - name: "todo"             # built-in: manage TODOs in Logseq format
      - name: "search"           # built-in: semantic search over embeddings
      - name: "weather"          # extension package
        config:
          default_location: "Berlin,DE"
      - name: "letterboxd"       # extension package
        config:
          username: "criterioncel"

    providers:                   # data ingress pipelines
      - name: "rss"
        ttl: "14d"
        config:
          feeds:
            - url: "https://n-gate.com/index.atom"
              tag: "internet"
            - url: "https://lwn.net/headlines/rss"
              tag: "linux"
          poll_interval: "4h"
      - name: "filesystem"
        ttl: "30d"
        config:
          watch:
            - path: "~/Documents/notes"
              tag: "notes-import"

    schedule:                    # cron-like jobs for this vault
      - name: "daily-journal-prefill"
        cron: "0 6 * * *"        # 6am: generate today's journal template
        job: "journal_prefill"
        model: "default"
      - name: "todo-reminders"
        cron: "0 9 * * *"
        job: "todo_scan"         # scan logseq for DOING/NOW/TODO items
        model: "default"
      - name: "provider-ingest"
        cron: "0 */4 * * *"
        job: "provider_run"
      - name: "embedding-compact"
        cron: "0 3 * * *"        # 3am: compact embeddings
        job: "embedding_compact"
      - name: "backlink-maintenance"
        cron: "0 2 * * *"        # 2am: suggest backlinks, improve graph
        job: "backlink_review"
        model: "heavy"
      - name: "entry-cleanup"
        cron: "0 4 * * 0"        # sunday 4am: improve recent entries
        job: "entry_review"
        model: "heavy"

  work:
    path: "~/braind-vaults/work"

    git:
      auto_commit: true
      commit_name: "braind"
      commit_email: "braind@localhost"

    changelog:
      enabled: true

    tools:
      - name: "journal"
      - name: "todo"
      - name: "search"

    providers: []

    schedule:
      - name: "daily-journal-prefill"
        cron: "0 8 * * 1-5"      # weekdays 8am
        job: "journal_prefill"
        model: "default"
```

### 4.2 Config Notes

- `tools` and `providers` both reference Go packages that implement well-defined
  interfaces (see sections 5 and 6). They are registered at compile time via Go
  import — no dynamic loading, no reflection magic. You add a package, import it
  in `cmd/braind/main.go`, and it's available by name in config.
- Only tools listed under a vault are available to the LLM when operating on
  that vault. The engine validates every tool call against this list.
- `model: "heavy"` on a schedule job means it uses the `llm.heavy` config
  instead of `llm.default`.
- `limits` can be overridden per vault if needed; otherwise global defaults are used.
- Provider retention is set per provider via `ttl` (examples: `2d`, `14d`, `30d`).

---

## 5. Tool System

### 5.1 Interface

```go
// Tool is the interface every tool package implements.
type Tool interface {
    // Name returns the tool identifier used in config and LLM function defs.
    Name() string

    // Description returns what the LLM sees in the tool definition.
    Description() string

    // Parameters returns the JSON schema for the tool's input parameters.
    Parameters() json.RawMessage

    // Execute runs the tool. It receives validated parameters and returns
    // a structured result. It MUST NOT access the shell, network (unless
    // that is the tool's explicit purpose, e.g. weather), or filesystem
    // outside the vault directory.
    Execute(ctx context.Context, req ToolRequest) (*ToolResponse, error)
}

type ToolRequest struct {
    VaultPath string
    Params    json.RawMessage
    Config    json.RawMessage   // vault-level tool config from yaml
}

type ToolResponse struct {
    Result  string             // human-readable summary
    Data    json.RawMessage    // structured data (optional)
    Changes []FileChange       // files modified (for git commit + changelog)
}

type FileChange struct {
    Path   string
    Action string             // "create", "modify", "delete"
    Diff   string             // unified diff
}
```

### 5.2 Built-in Tools

| Tool        | Description                                                       |
| ----------- | ----------------------------------------------------------------- |
| `journal`   | Write/edit Logseq journal entries and pages. Creates markdown     |
|             | files with proper `[[backlinks]]`, property drawers, and          |
|             | timestamps.                                                       |
| `todo`      | Add/complete/reprioritize TODOs in Logseq format (`TODO`, `DOING`,|
|             | `DONE`, `LATER`, `NOW`). Scans existing entries for state changes.|
| `search`    | Semantic search over the vault's embedding store. Returns ranked  |
|             | chunks with source file references.                               |
| `page_read` | Read a specific page or journal entry by name/date. Returns raw   |
|             | markdown content.                                                 |

### 5.3 Extension Tools (examples)

| Package           | Tool Name      | What it does                                  |
| ----------------- | -------------- | --------------------------------------------- |
| `tool/weather`    | `weather`      | Fetches weather for a location (wttr.in)      |
| `tool/letterboxd` | `letterboxd`   | Scrapes user's watched films + ratings        |
| `tool/wikipedia`  | `wikipedia`    | Looks up article summaries                    |
| `tool/caldav`     | `caldav`       | Reads upcoming events from a CalDAV calendar  |

### 5.4 Security Boundary

The engine enforces the following invariants at runtime:

1. **Allowlist validation.** Before executing any tool call returned by the LLM,
   the engine checks the tool name against the vault's configured `tools` list.
   Unknown tools → rejected, logged, and the LLM receives an error.
2. **No shell.** Tools are Go functions. There is no `exec.Command`, no
   `os/exec`, no subprocess spawning anywhere in the tool execution path.
   The engine refuses to load tools that attempt it (build-tag enforced).
3. **No cross-vault access.** Tool requests receive only the vault's own path.
   The engine does not expose other vault paths.
4. **No network unless declared.** Network capability is explicit.
   Tools declare `network/pull` (read-only HTTP like GET) and/or
   `network/push` (state-changing calls like POST/PATCH).
   Providers use a single `network/` capability class.
5. **Bounded execution.** Invocation budgets are strict and configurable
   through `limits` (`max_turns`, `max_tool_calls`, `max_wall_time`).
   Tools may request model callbacks, but only within budget
   (still no shell, still allowlist-validated).

---

## 6. Provider System

Providers are Go packages that ingest external data into the vault's sqlite
database. They do **not** write to the Logseq markdown files. Their data lives
in sqlite and is consumed by the context builder for embedding-augmented
retrieval.

### 6.1 Interface

```go
type Provider interface {
    // Name returns the provider identifier.
    Name() string

    // Ingest pulls data from the external source and stages it in sqlite.
    // Called on the provider's configured poll_interval.
    Ingest(ctx context.Context, db *sql.DB, cfg json.RawMessage) error

    // Sources returns available data sources for this provider.
    // Used by CLI to discover @mentions and by context builder.
    Sources() ([]ProviderSource, error)
}

type ProviderData struct {
    SourceID   string            // unique ID from the source
    SourceType string            // e.g. "rss", "filesystem"
    Title      string
    Content    string
    URL        string
    Tags       []string
    FetchedAt  time.Time
}

type ProviderSource struct {
    ID    string            // e.g. "n-gate", "lwn"
    Name  string            // e.g. "n-gate.com", "LWN"
    Type  string            // parent provider name
}
```

Provider lifecycle and retention are daemon-managed:

- Provider data is stored in sqlite only (content + metadata).
- Provider `ttl` in vault config controls retention.
- The daemon periodically purges expired and orphaned provider rows.
- Heavy/binary attachments are out of scope for MVP.

### 6.2 CLI @mention Syntax

Provider data can be explicitly included in context via CLI:

```bash
brain ask @rss/n-gate "summarize the latest internet trends"
brain ask @rss "summarize what mattered this week"
brain ask @calendar/today "what do I have scheduled"
```

`@provider` includes all sources under that provider.
`@provider/source` scopes to a single source.

The daemon exposes a `sources` command to list available providers and their sources:

```bash
$ brain sources
rss/
  n-gate      https://n-gate.com/index.atom
  lwn         https://lwn.net/headlines/rss
calendar/
  work       caldav+https://cal.example.com/caldav
  personal    caldav+https://cal.example.com/Personal
filesystem/
  notes      ~/Documents/notes
```

### 6.2 Pipeline Flow

```
Provider.Ingest()
    │
    ▼
sqlite staging table (raw items, deduplicated by source_id)
    │
    ▼
Embedding worker (batch embed new items via local model)
    │
    ▼
sqlite-vec (vector index for semantic search)
    │
    ▼
Compaction (every N intervals, summarize old items,
            keep embeddings of summaries, drop raw text)
```

### 6.3 Built-in Providers

| Provider      | Description                                             |
| ------------- | ------------------------------------------------------- |
| `rss`         | Fetches and parses Atom/RSS feeds. Stages articles.     |
| `filesystem`  | Watches directories for new/changed files. Stages text. |

### 6.4 Extension Providers (examples)

| Package             | Name         | What it does                                  |
| ------------------- | ------------ | --------------------------------------------- |
| `provider/caldav`   | `caldav`     | Reads upcoming events                         |
| `provider/git-log`  | `git-log`    | Ingests commit messages from local repos      |
| `provider/imap`     | `imap`       | Read-only ingest of email subjects/bodies     |

---

## 7. LLM Integration

### 7.1 Tool-Calling Protocol

`braind` uses the OpenAI-compatible tool-calling API that Ollama exposes. The
engine:

1. Builds a system prompt describing the vault's current state (date, active
   TODOs count, recent journal topics).
2. Builds a user prompt from the CLI input or scheduled job.
3. Includes tool definitions for every tool in the vault's allowlist.
4. Sends the request. Expects either a tool call or a text response.
5. If tool call: validates, executes, returns result as a follow-up message.
   Single turn. The model sees the tool result and produces a final text
   response.
6. If text response: returns to the user.

### 7.2 Context Assembly

The context builder assembles the LLM prompt from:

- **System prompt:** vault metadata, available tools, date, user preferences.
- **Recent journal entries:** last 7 days of journal markdown (configurable).
- **Active TODOs:** all `TODO`/`DOING`/`NOW` items across the vault.
- **Semantic search results:** embedding queries from the user's input against
  sqlite-vec to pull relevant prior entries and provider data.
- **User message:** the actual input.

### 7.3 Model Tiers

| Tier     | When used                        | Typical model      | Purpose                        |
| -------- | -------------------------------- | ------------------ | ------------------------------ |
| `default`| Interactive use, scheduled day   | gemma3:4b (Q4)     | Journal prefill, todos, ask    |
| `heavy`  | Scheduled night jobs             | gemma3:12b (Q4)    | Backlink review, entry cleanup |

Future: in-process GGUF runtime (via `llama.cpp` Go bindings or `cgofuse`),
eliminating the Ollama dependency entirely.

---

## 8. Versioning & Changelog

### 8.1 Git Auto-Commits

Every vault mutation triggers an auto-commit:

```
commit <hash>
Author: braind <braind@localhost>
Date:   2025-04-15 09:32:01

    braind: journal write (tool=journal, model=gemma3:4b)
    
    Updated journals/2025_04_15.md
    - Added tiramisu note under [[Hoffman Recipe]]
    - Added TODO: pay taxes tomorrow
```

### 8.2 Changelog Format

`changelog.org` at the vault root. Appended to, never overwritten.

```org
* 2025-04-15

** 09:32 — journal write (interactive)
   - Model: gemma3:4b
   - Tools used: journal, letterboxd
   - Changes:
     - ~journals/2025_04_15.md~: Added tiramisu note, Letterboxd entry for "The Substance" (★★★★☆)
     - ~pages/Hoffman Recipe.md~: Added backlink from today's journal
   - Rationale: User asked to journal tiramisu experience and fetch last watched film.

** 02:00 — backlink review (scheduled)
   - Model: gemma3:12b
   - Tools used: search, journal
   - Changes:
     - ~pages/Coffee.md~: Added backlink to [[Hoffman Recipe]]
     - ~pages/Hoffman Recipe.md~: Added "See also" section with [[Tiramisu]]
   - Rationale: Nightly backlink scan found unlinked references between coffee and recipe pages.

** 06:00 — daily journal prefill (scheduled)
   - Model: gemma3:4b
   - Tools used: journal
   - Changes:
     - ~journals/2025_04_15.md~: Created daily template with weather, calendar items, open TODOs
   - Rationale: Scheduled daily journal template generation.
```

---

## 9. CLI

```
brain --help
```

### 9.1 Commands

| Command                    | Description                                         |
| -------------------------- | --------------------------------------------------- |
| `brain vault list`         | List configured vaults and their status.             |
| `brain vault switch <name>`| Set active vault for subsequent commands.            |
| `brain ask <prompt>`       | Send a prompt to the active vault's engine.          |
| `brain ask @<source> <prompt>` | Same as above, but include provider data in context     |
|                            | (e.g. `@rss/n-gate`).                            |
| `brain journal [date]`     | Open or create a journal entry (via $EDITOR or       |
|                            | ask the LLM to prefill).                             |
| `brain todos`              | Show active TODOs sorted by urgency/category.        |
| `brain sources`            | List available provider sources for active vault.      |
| `brain log`                | Show today's changelog entries.                      |
| `brain log --since <date>` | Show changelog since a given date.                   |
| `brain diff [commit]`      | Show git diff for a changelog entry (defaults to     |
|                            | last daemon commit).                                 |
| `brain undo [commit]`      | Revert a daemon commit.                              |
| `brain status`             | Show daemon status, active vault, model info.        |
| `brain run <job>`          | Manually trigger a scheduled job.                    |
| `brain provider run <name>`| Manually trigger a provider ingestion.               |

### 9.2 Active Vault

The CLI tracks the active vault in `$XDG_CONFIG_HOME/braind/active_vault`
(default: first vault in config, or set via `brain vault switch`).

### 9.3 Example Session

```bash
$ brain vault switch personal
Switched to vault: personal

$ brain ask 'add that I [[made tiramisu with the hoffman recipe today]], \
  goddamn the espresso really did hit. Also add the movie I watched on \
  letterboxd to my journal. And I need to pay my taxes tomorrow.'
```

Engine flow:
1. Context builder: loads recent journals, active TODOs, searches embeddings
   for "Hoffman Recipe" and "tiramisu".
2. LLM receives prompt + tool defs: `journal`, `todo`, `search`, `weather`,
   `letterboxd`.
3. LLM calls `letterboxd.get_last_watched()` → returns film + rating.
4. Engine validates `letterboxd` is in allowlist → executes → gets result.
5. LLM calls `journal.write(date="2025-04-15", content="...")` with tiramisu
   note, film entry, and tax TODO, all in Logseq format with backlinks.
6. Engine validates `journal` is in allowlist → executes → writes markdown.
7. Git auto-commit. Changelog entry created.
8. CLI displays summary.

```bash
$ brain todos
⚡ URGENT
   ○ TODO Pay taxes (due: tomorrow)                    journals/2025_04_15.md
   
📋 DOING
   ○ DOING Fix bike chain                              pages/Bike Repair.md
   
🕐 LATER
   ○ LATER Research espresso machines                  pages/Coffee.md
   ○ LATER Call mom about birthday plans               journals/2025_04_14.md
```

---

## 10. Scheduling

### 10.1 Built-in Jobs

| Job Name             | Description                                                  |
| -------------------- | ------------------------------------------------------------ |
| `journal_prefill`    | Creates today's journal template: weather, calendar items,   |
|                      | open TODOs carried over, provider digest highlights.          |
| `todo_scan`          | Scans vault for TODO items, checks for deadlines, sends      |
|                      | reminders via the changelog (and eventually system notify).   |
| `provider_run`       | Runs all configured providers for the vault.                  |
| `embedding_compact`  | Summarizes old provider data, replaces raw embeddings with    |
|                      | summary embeddings, frees space.                              |
| `backlink_review`    | Scans pages for mentions that should be `[[backlinked]]`.       |
|                      | Uses heavy model. Auto-applies, logged in changelog.          |
| `entry_review`       | Reviews recent entries for consistency, formatting,          |
|                      | duplicates. Uses heavy model. Auto-applies, logged.          |

### 10.2 Timer Integration

```ini
# braind.service
[Unit]
Description=braind — local second-brain daemon
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/braind
Restart=on-failure
Environment=XDG_CONFIG_HOME=%h/.config
Environment=XDG_RUNTIME_DIR=/run/user/%U

[Install]
WantedBy=default.target
```

MVP recommendation: use external timers (`systemd --user`, launchd) and call
`brain run <job>`. This keeps daemon complexity lower while preserving full
automation.

File sync behavior uses `fsnotify` to track vault changes and keep sqlite
manifests in sync with user edits.

```bash
systemd-run --user --on-calendar="*-*-* 06:00:00" brain run journal_prefill
```

---

## 11. Embedding Store

### 11.1 Schema

```sql
-- Staging table for provider data
CREATE TABLE provider_items (
    id          INTEGER PRIMARY KEY,
    source_type TEXT NOT NULL,
    source_id   TEXT NOT NULL UNIQUE,
    title       TEXT,
    content     TEXT NOT NULL,
    url         TEXT,
    tags        TEXT,              -- JSON array
    fetched_at  DATETIME NOT NULL,
    embedded_at DATETIME,
    compacted   BOOLEAN DEFAULT FALSE
);

-- Embeddings via sqlite-vec
CREATE VIRTUAL TABLE vec_items USING vec0(
    id INTEGER PRIMARY KEY,
    embedding float[768]           -- match embedding model dimension
);
```

### 11.2 Compaction

On schedule, the engine:
1. Selects provider items older than N days (configurable, default 14).
2. Groups by tag/source_type.
3. Asks the LLM (heavy model) to summarize each group into a compact note.
4. Embeds the summaries, stores new vectors, marks originals as compacted.
5. Compact items are excluded from future searches (or returned with lower weight).

---

## 12. Logseq Compatibility

### 12.1 Markdown Format

`braind` writes Logseq-compatible markdown:

```markdown
- ## [[2025-04-15]] Tuesday

- **09:32** Made [[tiramisu]] with the [[Hoffman Recipe]] today.
  Goddamn the espresso really did hit. Should brew less next time.
  - Rating: ☕☕☕☕
  - Related: [[Coffee Experiments]]

- **09:32** Watched: [[The Substance]] (2024)
  - Rating: ★★★★☆
  - Logged via [[Letterboxd]]

- TODO Pay taxes
  SCHEDULED: <2025-04-16>
  :PROPERTIES:
  :urgency: high
  :END:
```

### 12.2 Supported Logseq Features

- `[[backlinks]]`
- `TODO` / `DOING` / `DONE` / `LATER` / `NOW` task states
- `SCHEDULED:` and `DEADLINE:` timestamps
- `:PROPERTIES:` drawers
- `#+tags:` page properties
- Journal page naming: `YYYY_MM_DD.md`
- Page naming: `Page Name.md` (titlecased, spaces)

---

## 13. MVP Scope

### Phase 1 — "It breathes"

- [ ] Daemon + CLI skeleton (socket communication)
- [ ] Vault model (config loading, directory init, git init)
- [ ] LLM client (Ollama, tool-calling, single turn)
- [ ] Built-in tools: `journal`, `todo`, `search`, `page_read`
- [ ] Context builder (recent journals + TODOs, no embeddings yet)
- [ ] Git auto-commit on every mutation
- [ ] Changelog writer
- [ ] `brain ask`, `brain todos`, `brain log`, `brain diff`, `brain undo`
- [ ] Basic scheduling (journal_prefill, todo_scan)

### Phase 2 — "It remembers"

- [ ] sqlite-vec embedding store
- [ ] Provider pipeline (RSS provider)
- [ ] Embedding-based semantic search tool
- [ ] Provider compaction
- [ ] `brain provider run`, `brain sources`
- [ ] `@<provider>/<source>` prefix in `brain ask` (v2/v3)

### Phase 3 — "It extends"

- [ ] Extension tool packages (weather, letterboxd, wikipedia)
- [ ] Extension provider packages (filesystem, caldav)
- [ ] Night maintenance jobs (backlink_review, entry_review)

### Phase 4 — "It runs alone"

- [ ] In-process GGUF runtime (no Ollama dependency)
- [ ] System notification integration (dbus/libnotify)
- [ ] `brain journal` (interactive or LLM-prefilled)
- [ ] Web UI? (TUI via bubbletea first)

---

## 14. Key Design Decisions

| Decision                      | Rationale                                            |
| ----------------------------- | ---------------------------------------------------- |
| Go packages, not plugins      | Compile-time safety, no `plugin.*` fragility,        |
| (import-registered)           | explicit is better than implicit.                    |
| Single-turn tool calling      | No agentic loops = no prompt injection escalation.   |
| Auto-apply + git versioning   | Frictionless UX with undo. `brain undo` reverts any  |
|                               | daemon commit instantly.                              |
| sqlite-vec for embeddings     | Zero-dependency, single file, ships with Go via      |
|                               | `sqlite3` C bindings. No external vector DB needed.  |
| Logseq markdown source of     | Interoperable with Logseq directly. Kill the daemon,  |
| truth                         | open the folder in Logseq, everything works.          |
| Per-vault sockets             | Clean isolation, natural multiplexing, unix                   |
|                               | philosophy composability.                             |
| Provider data in sqlite only  | External data is augmented context, not first-class       |
|                               | knowledge. Vault markdown stays human-authored +          |
|                               | LLM-assisted, not a data dump.                       |
