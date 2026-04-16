# Braind

A privacy-first *nix daemon for organizing Logseq-formatted second brains with local LLMs on consumer hardware. Basic personal assistant features included. 

> [!IMPORTANT]
> This is a vibeslopped™ repository that actually serves a purpose. Start with [SPEC.md](SPEC.md) for the full design.

> [!CAUTION]
> Nothing works yet, come back later

## What?

`braind` runs as a local daemon scoped to **vaults** , isolated knowledge bases with their own config. It:

- Augments your Logseq markdown with LLM tool-calling (journal, todos, search, and more)
- Keeps everything local: no telemetry, no cloud, no daddy Altman
- Versions every write with git; `brain undo` reverts any daemon commit instantly
- Maintains a `CHANGELOG.org` recording what changed and why

**Design philosophy:** tool-calling #1, no shell access for idiot LLMs, no cross-vault access, read-only outside of the vault. 

## But why

Most "AI second brain" tools are either:
- Cloud-hosted (privacy? what privacy)
- Agentic by default (burns tokens, hand over your data to whoever politely asks)
- Annoying to run locally 
- Designed for normies and backed by huge VCs

`braind` uses Go for portability, relies on cool lightweight models like `Gemma 4 E4B`, and aims to ship the whole runtime in one binary.

What this is *NOT*:
- An `OpenClaw` competitor,
- A silver bullet (Expect some AI slop in your notes, don't try to run this on a x220)
- For mass consumption,
  - If you hate:
    - DWM
    - Yaml
    - systemd user units

  ... then this probably isn't the right tool for you

Think of this as the bastard child of `taskwarrior`, `mpd`, `org-mode` and an IRC bot
