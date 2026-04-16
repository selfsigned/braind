# Tool Extension Point

Tools are Go packages that implement the `Tool` interface and are registered at compile time.

## Interface

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

## Registration

To add a tool:

1. Create a new directory under `tools/` (or use builtin/tools/ for core tools)
2. Implement the `Tool` interface
3. Import the package in `cmd/braind/main.go`
4. Register via the tool registry's `Register()` call

## Security

- Tools must NOT use `os/exec` or shell execution
- Tools must NOT access network unless they declare `network/pull` or `network/push` capability
- Tools must only access the vault directory passed in `ToolRequest.VaultPath`
