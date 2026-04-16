# Provider Extension Point

Providers are Go packages that ingest external data into the vault's sqlite database.

## Interface

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

## Registration

To add a provider:

1. Create a new directory under `providers/` (or use builtin/providers/ for core providers)
2. Implement the `Provider` interface
3. Import the package in `cmd/braind/main.go`
4. Register via the provider registry's `Register()` call

## Security

- Providers declare `network/` capability class for network access
- Provider data lives in sqlite only, never in Logseq markdown files
