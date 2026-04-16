package page_read

import (
	"context"
	"encoding/json"

	"github.com/braind/braind/internal/interfaces"
)

type PageReadTool struct{}

func New() *PageReadTool { return &PageReadTool{} }

func (t *PageReadTool) Name() string { return "page_read" }

func (t *PageReadTool) Description() string {
	return "Read a specific page or journal entry by name/date. Returns raw markdown content."
}

func (t *PageReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"page": {
				"type": "string",
				"description": "Page name to read (without .md extension)"
			},
			"date": {
				"type": "string",
				"description": "Journal date in YYYY-MM-DD format"
			}
		}
	}`)
}

func (t *PageReadTool) Execute(ctx context.Context, req interfaces.ToolRequest) (*interfaces.ToolResponse, error) {
	return nil, nil
}
