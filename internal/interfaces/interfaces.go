package interfaces

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type ToolRequest struct {
	VaultPath string
	Params    json.RawMessage
	Config    json.RawMessage
}

type FileChange struct {
	Path   string
	Action string
	Diff   string
}

type ToolResponse struct {
	Result  string
	Data    json.RawMessage
	Changes []FileChange
}

type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage
	Execute(ctx context.Context, req ToolRequest) (*ToolResponse, error)
}

type ProviderSource struct {
	ID   string
	Name string
	Type string
}

type ProviderData struct {
	SourceID   string
	SourceType string
	Title      string
	Content    string
	URL        string
	Tags       []string
	FetchedAt  time.Time
}

type Provider interface {
	Name() string
	Ingest(ctx context.Context, db *sql.DB, cfg json.RawMessage) error
	Sources() ([]ProviderSource, error)
}
